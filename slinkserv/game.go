package slinkserv

import (
	"fmt"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/lologarithm/slink/slinkserv/messages"
	"github.com/lologarithm/survival/physics"
	"github.com/lologarithm/survival/physics/quadtree"
)

// Entity type constants
const (
	ETypeUnknown uint16 = 0
	ETypeHead    uint16 = 1
	ETypeSegment uint16 = 2
	ETypeFood    uint16 = 3
)

// GameSession represents a single game
type GameSession struct {
	ID uint32

	// map character ID to client
	Clients map[uint32]*User

	IntoGameManager chan<- GameMessage     // Game can only write to this channel, not read.
	FromGameManager chan InternalMessage   // Messages from the game Manager.
	FromNetwork     <-chan GameMessage     // FromNetwork is read only here, messages from players.
	ToNetwork       chan<- OutgoingMessage // Messages to players!

	Exit   chan int
	Status GameStatus

	// Private
	World          *GameWorld    // Current world state
	prevWorlds     []*GameWorld  // Last X seconds of game states
	commandHistory []interface{} // Last X seconds of commands
}

// GameWorld represents all the data in the world.
// Physical entities and the physics simulation.
type GameWorld struct {
	Entities       map[uint32]*Entity
	Snakes         map[uint32]*Snake
	Tree           quadtree.QuadTree
	TickID         uint32
	TicksPerSecond float64
	TickLength     float64
}

// Clone returns a deep copy of the game world at this time.
func (gw *GameWorld) Clone() *GameWorld {
	// TODO: make this work.
	return &GameWorld{}
}

// EntitiesMsg converts all entities in the world to a network message.
func (gw *GameWorld) EntitiesMsg() []*messages.Entity {
	es := make([]*messages.Entity, len(gw.Entities))
	for idx, e := range gw.Entities {
		es[idx] = e.toMsg()
	}

	return es
}

// SnakesMsg converts all snakes in the world to a network message.
func (gw *GameWorld) SnakesMsg() []*messages.Snake {
	es := make([]*messages.Snake, len(gw.Snakes))
	for idx, e := range gw.Snakes {
		es[idx] = e.toSnakeMsg()
	}

	return es
}

func (gw *GameWorld) Tick() []Collision {
	for _, snake := range gw.Snakes {
		for i := len(snake.Segments) - 1; i != 0; i-- {
			snake.Segments[i].Position = snake.Segments[i-1].Position
		}
		snake.Segments[0].Position = snake.Position
		tickmv := int32(float64(snake.Speed) / gw.TicksPerSecond)
		newpos := physics.Vect2{
			X: snake.Position.X + snake.Facing.X*tickmv,
			Y: snake.Position.Y + snake.Facing.Y*tickmv,
		}
		snake.Position = newpos
	}

	// TODO: calculate collisions!
	// 1. check quad tree for possible bounding box collisions
	// 2. actually calculate exact distance and see if circles collide.
	// The only collisions that matter are snake heads!
	// If head touches body, head dies
	// If head touches head, bigger head wins.

	gw.TickID++
	return nil
}

type Collision struct {
	Entity1 *Entity
	Entity2 *Entity
}

// Run starts the game!
func (g *GameSession) Run() {
	g.World.TickLength = 1000.0 / g.World.TicksPerSecond
	waiting := true
	waitms := int64(float64(time.Millisecond) * g.World.TickLength)
	nextTick := time.Now().UTC().UnixNano() + waitms

	var timeout time.Duration
	for {
		waiting = true
		// figure out timeout before each waiting loop.
		timeout = time.Duration(nextTick - time.Now().UTC().UnixNano())
		for waiting {
			select {
			case <-time.After(timeout):
				waiting = false
				nextTick += waitms // Set next tick exactly
				break
			case msg := <-g.FromNetwork:
				switch msg.mtype {
				case messages.SetDirectionMsgType:
					g.setDirection(msg)
				default:
					fmt.Printf("game.go:Run(): UNKNOWN MESSAGE TYPE: %T\n", msg)
				}
			case imsg := <-g.FromGameManager:
				switch timsg := imsg.(type) {
				case AddPlayer:
					newid := uint32(len(g.World.Entities))
					g.World.Snakes[newid] = NewSnake(newid)
					g.World.Entities[newid] = g.World.Snakes[newid].Entity
					for _, s := range g.World.Snakes[newid].Segments {
						g.World.Entities[s.ID] = s
					}
					g.Clients[timsg.Client.ID] = &User{
						Account: nil,
						SnakeID: newid,
						GameID:  g.ID,
						Client:  timsg.Client,
					}
					cgr := &messages.GameConnected{
						ID:       g.ID,
						TickID:   g.World.TickID,
						Entities: g.World.EntitiesMsg(),
					}
					outgoing := NewOutgoingMsg(timsg.Client, messages.GameConnectedMsgType, cgr)
					timsg.Client.ToNetwork <- outgoing
				case RemovePlayer:
					log.Printf("Disconnecting player: %d", timsg.Client.ID)
					user := g.Clients[timsg.Client.ID]
					delete(g.Clients, timsg.Client.ID)
					snake := g.World.Snakes[user.SnakeID]
					delete(g.World.Entities, snake.ID)
					for _, v := range snake.Segments {
						delete(g.World.Entities, v.ID)
					}
					delete(g.World.Snakes, snake.ID)
				}
			case <-g.Exit:
				fmt.Print("EXITING: Run in Game.go\n")
				return
			}
		}
		collisions := g.World.Tick()
		for range collisions {

		}
		// Now create a clone of the world to add to the historical world data
		// Now scan through command history looking for old commands that are
		// before the oldest stored tick state.
		if g.World.TickID%100 == 0 {
			g.SendMasterFrame()
		}
	}
}

func (g *GameSession) setDirection(msg GameMessage) {
	dirmsg := msg.net.(*messages.SetDirection)

	client := g.Clients[msg.client.ID]
	lat := atomic.LoadInt64(&client.Client.latency)
	oldtick := g.World.TickID - uint32(float64(lat)/g.World.TickLength)
	// TODO: Allow a maximum turn rate so it can't just snap any direction
	// TODO: Apply facing change request at old tick instead of 'now'

	snake := g.World.Snakes[client.SnakeID]
	snake.Facing = physics.Vect2{X: dirmsg.Facing.X, Y: dirmsg.Facing.Y}
	snake.Facing = physics.NormalizeVect2(snake.Facing, 1)

	dirmsg.TickID = oldtick
	dirmsg.ID = client.SnakeID
	frame := messages.Frame{
		// Seq Doesn't matter because its set by server when its sent.
		MsgType:       messages.SetDirectionMsgType,
		ContentLength: uint16(dirmsg.Len()),
	}
	g.sendToAll(OutgoingMessage{
		msg: messages.Packet{
			Frame:  frame,
			NetMsg: dirmsg,
		},
	})
}

func (g *GameSession) sendToAll(msg OutgoingMessage) {
	for _, c := range g.Clients {
		msg.dest = c.Client
		g.ToNetwork <- msg
	}
}

// SendMasterFrame will create a 'master' state of all things and send to each client.
func (g *GameSession) SendMasterFrame() {
	mf := &messages.GameMasterFrame{
		ID:       g.ID,
		Entities: g.World.EntitiesMsg(),
		Snakes:   g.World.SnakesMsg(),
		Tick:     g.World.TickID,
	}

	frame := messages.Frame{
		MsgType:       messages.GameMasterFrameMsgType,
		Seq:           1,
		ContentLength: uint16(mf.Len()),
	}
	g.sendToAll(OutgoingMessage{
		msg: messages.Packet{
			Frame:  frame,
			NetMsg: mf,
		},
	})
}

// NewGame constructs a new game and starts it.
func NewGame(toGameManager chan<- GameMessage, fromNetwork <-chan GameMessage, toNetwork chan<- OutgoingMessage) *GameSession {
	seed := uint64(rand.Uint32())
	seed = seed << 32
	seed += uint64(rand.Uint32())
	g := &GameSession{
		IntoGameManager: toGameManager,
		FromGameManager: make(chan InternalMessage, 100),
		FromNetwork:     fromNetwork,
		ToNetwork:       toNetwork,
		World: &GameWorld{
			Entities:       map[uint32]*Entity{},
			Snakes:         map[uint32]*Snake{},
			TicksPerSecond: 50,
		},
		Exit:    make(chan int, 1),
		Clients: make(map[uint32]*User, 16),
	}
	return g
}

type Snake struct {
	*Entity            // Snake entity itself is the head.
	Segments []*Entity // Segments are the body!
	Speed    int32     // Velocity!
}

func (s *Snake) toSnakeMsg() *messages.Snake {
	segIDs := make([]uint32, len(s.Segments))
	for i, seg := range s.Segments {
		segIDs[i] = seg.ID
	}
	return &messages.Snake{
		ID:       s.ID,
		Segments: segIDs,
		Speed:    s.Speed,
	}
}

// NewSnake creates a new snake at a random location.
func NewSnake(id uint32) *Snake {
	pos := physics.Vect2{
		X: int32(rand.Intn(1000) - 500),
		Y: int32(rand.Intn(1000) - 500),
	}
	return &Snake{
		Entity: &Entity{
			ID:       id,
			EType:    ETypeHead,
			Position: pos,
			Facing: physics.Vect2{
				X: 0,
				Y: 1,
			},
			Size: 1000,
		},
		Segments: []*Entity{
			&Entity{
				ID:          id + 1,
				EType:       ETypeSegment,
				ContainerID: id,
				Position: physics.Vect2{
					X: pos.X,
					Y: pos.Y - 300,
				},
				Size: 1000,
			},
		},
		Speed: 100,
	}
}

// Entity represents a single object in the game.
type Entity struct {
	ID          uint32
	Name        string
	EType       uint16
	Size        int32 // Radius
	ContainerID uint32
	Position    physics.Vect2 // Center of entity
	Facing      physics.Vect2
}

func (e *Entity) toMsg() *messages.Entity {
	o := &messages.Entity{
		ID:    e.ID,
		EType: e.EType,
		X:     e.Position.X,
		Y:     e.Position.Y,
		Size:  e.Size,
		Facing: &messages.Vect2{
			X: e.Facing.X,
			Y: e.Facing.Y,
		},
		ContainerID: e.ContainerID,
	}

	return o
}

// Intersects calculates if two entities overlap -- used currently for chunk generation.
func (e *Entity) Intersects(o *Entity) bool {
	// TODO: (R0-R1)^2 <= (x0-x1)^2+(y0-y1)^2 <= (R0+R1)^2
	return true
}

func (e *Entity) BoundingBox() quadtree.BoundingBox {
	return quadtree.BoundingBox{
		MinX: e.Position.X - e.Size,
		MaxX: e.Position.X + e.Size,
		MinY: e.Position.Y - e.Size,
		MaxY: e.Position.Y + e.Size,
	}
}

// type BoundingBoxer interface {
// 	BoundingBox() BoundingBox
// 	BoxID() uint32
// }

// GameMessage is a message from a client to a game.
type GameMessage struct {
	net    messages.Net
	client *Client
	mtype  messages.MessageType
}

// InternalMessage is for messages between internal components (gamesession and gamemanager) that never leaves the server.
type InternalMessage interface {
}

// ConnectedGame is sent to client to notify they are now connected to a game.
type ConnectedGame struct {
	ID     uint32
	ToGame chan<- GameMessage
}

// RemovePlayer is sent to remove a player from a game.
type RemovePlayer struct {
	Client *Client
}

// AddPlayer is sent to add a player to a game.
type AddPlayer struct {
	Entity *Entity
	Client *Client
}
