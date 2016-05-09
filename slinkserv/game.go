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
	FromNetwork     chan GameMessage       // FromNetwork is read only here, messages from players.
	ToNetwork       chan<- OutgoingMessage // Messages to players!

	Exit   chan int
	Status GameStatus

	// Private
	World *GameWorld // Current world state

	// Historical state
	prevWorlds     [50]*GameWorld // Last 1 second of game states
	prevHead       uint32
	commandHistory []GameMessage // Last 1 second of commands
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

func NewWorld() *GameWorld {
	ticks := 50.0
	tickLength := 1000.0 / ticks
	return &GameWorld{
		Entities:       map[uint32]*Entity{},
		Snakes:         map[uint32]*Snake{},
		TicksPerSecond: ticks,
		TickLength:     tickLength,
		Tree: quadtree.NewQuadTree(quadtree.BoundingBox{
			MinX: -1000000,
			MaxX: 1000000,
			MinY: -1000000,
			MaxY: 1000000,
		}),
	}
}

// Clone returns a deep copy of the game world at this time.
func (gw *GameWorld) Clone() *GameWorld {
	nw := NewWorld()
	for k, e := range gw.Entities {
		ne := &Entity{}
		*ne = *e
		nw.Entities[k] = ne
		nw.Tree.Add(ne)
	}

	for k, s := range gw.Snakes {
		ns := &Snake{}
		*ns = *s
		*ns.Entity = *s.Entity
		nw.Snakes[k] = ns
	}
	return nw
}

// EntitiesMsg converts all entities in the world to a network message.
func (gw *GameWorld) EntitiesMsg() []*messages.Entity {
	es := make([]*messages.Entity, len(gw.Entities))
	idx := 0
	for _, e := range gw.Entities {
		es[idx] = e.toMsg()
		idx++
	}

	return es
}

// SnakesMsg converts all snakes in the world to a network message.
func (gw *GameWorld) SnakesMsg() []*messages.Snake {
	es := make([]*messages.Snake, len(gw.Snakes))
	idx := 0
	for _, snake := range gw.Snakes {
		es[idx] = snake.toSnakeMsg()
		idx++
	}
	return es
}

func (gw *GameWorld) Tick() []Collision {
	for _, snake := range gw.Snakes {
		snakeDist := snake.Size / 2
		tickmv := (float64(snake.Speed) / gw.TicksPerSecond) / 100.0 // Dir vector normalizes to 100, so divide speed by 100
		newpos := physics.Vect2{
			X: snake.Position.X + int32(float64(snake.Facing.X)*tickmv),
			Y: snake.Position.Y + int32(float64(snake.Facing.Y)*tickmv),
		}
		snake.Position = newpos

		for _, seg := range snake.Segments {
			movevect := physics.SubVect2(newpos, seg.Position)
			mag := int32(movevect.Magnitude())
			if mag > snakeDist {
				movevect = physics.NormalizeVect2(movevect, mag-snakeDist)
				newpos = physics.Vect2{
					X: seg.Position.X + movevect.X,
					Y: seg.Position.Y + movevect.Y,
				}
				seg.Position = newpos
			}
			seg.Facing = movevect
			newpos = seg.Position
		}
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

// func Bezier2(start physics.Vect2, control physics.Vect2, end physics.Vect2, t int32) physics.Vect2 {
// 	return physics.AddVect2(physics.AddVect2(physics.MultVect2(start, ((1-t)*(1-t))), physics.MultVect2(control, 2 * t * (1 - t))), physics.MultVect2(end, (t * t));
// }

func (g *GameSession) replayHistory(ticks uint32) {
	for i := uint32(0); i < ticks; i++ {
		// 1. Apply and commands from this tick
		for _, msg := range g.commandHistory {
			if msg.currentTick == g.World.TickID {
				g.applyCommand(msg, false)
			}
		}
		// 2. call tick
		g.World.Tick()
		// 3.
		g.createHistoryPoint()
	}
}

func (g *GameSession) applyCommand(msg GameMessage, echo bool) {
	switch msg.mtype {
	case messages.SetDirectionMsgType:
		dirmsg := msg.net.(*messages.SetDirection)
		g.setDirection(dirmsg.Facing, dirmsg.ID)
		if echo {
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
	case messages.DisconnectedMsgType:
		s := g.World.Snakes[msg.clientID]
		g.removeSnake(s)
	default:
		fmt.Printf("game.go:Run(): UNKNOWN MESSAGE TYPE: %T\n", msg)
	}
}

func (g *GameSession) createHistoryPoint() {
	g.prevWorlds[g.prevHead] = g.World.Clone()
	g.prevHead++
	if g.prevHead == 50 {
		g.prevHead = 0 // ring buffer!
	}
}

func (g *GameSession) resetToHistory(u *User) uint32 {
	lat := atomic.LoadInt64(&u.Client.latency)
	ticklag := uint32(float64(lat) / g.World.TickLength)
	if ticklag > 50 {
		ticklag = 50
	}
	// Reset world back a bit.
	if ticklag > 0 {
		if g.World.TickID < ticklag {
			ticklag = g.World.TickID
		}
		laghead := g.prevHead - ticklag
		if laghead < 0 {
			laghead += 50
		}
		g.World = g.prevWorlds[laghead].Clone()
	}
	return ticklag
}

type Collision struct {
	Entity1 *Entity
	Entity2 *Entity
}

// Run starts the game!
func (g *GameSession) Run() {
	waiting := true
	waitms := int64(float64(time.Millisecond) * g.World.TickLength)
	nextTick := time.Now().UTC().UnixNano() + waitms

	var timeout time.Duration
	for {
		waiting = true

		// Now create a clone of the world to add to the historical world data
		g.createHistoryPoint()

		// Now scan through command history looking for old commands that are
		// before the oldest stored tick state.
		newHist := make([]GameMessage, len(g.commandHistory))
		hidx := 0
		for _, m := range g.commandHistory {
			if m.currentTick >= g.World.TickID-50 {
				newHist[hidx] = m
				hidx++
			}
		}
		g.commandHistory = newHist[:hidx]

		// figure out timeout before each waiting loop.
		timeout = time.Duration(nextTick - time.Now().UTC().UnixNano())
		for waiting {
			select {
			case <-time.After(timeout):
				waiting = false
				nextTick += waitms // Set next tick exactly
				break
			case msg := <-g.FromNetwork:
				msg.currentTick = g.World.TickID

				// First check if this message is out of date!
				isOld := false
				for _, m := range g.commandHistory {
					if m.clientID == msg.clientID && m.mtype == msg.mtype {
						if m.currentTick >= msg.currentTick {
							isOld = true
							break
						}
					}
				}
				if isOld {
					// Exit this msg processing now.
					break
				}
				client := g.Clients[msg.clientID]
				if client == nil {
					break // Client is no longer connected.
				}
				ticklag := g.resetToHistory(client)

				// Apply the new message
				g.applyCommand(msg, true)

				// Now replay history!
				g.replayHistory(ticklag)
				g.commandHistory = append(g.commandHistory, msg)
			case imsg := <-g.FromGameManager:
				switch timsg := imsg.(type) {
				case AddPlayer:
					g.addPlayer(timsg)
				case RemovePlayer:
					log.Printf("Removing player %d from game %d.", timsg.Client.ID, g.ID)
					user := g.Clients[timsg.Client.ID]
					delete(g.Clients, timsg.Client.ID)

					snake := g.World.Snakes[user.SnakeID]
					g.removeSnake(snake)
					removecmd := GameMessage{
						clientID:    snake.ID,
						mtype:       messages.DisconnectedMsgType,
						currentTick: g.World.TickID,
					}
					g.commandHistory = append(g.commandHistory, removecmd)
				}
			case <-g.Exit:
				fmt.Print("EXITING: Run in Game.go\n")
				return
			}
		}
		collisions := g.World.Tick()
		for range collisions {

		}
		if g.World.TickID%100 == 0 {
			g.SendMasterFrame()
		}
	}
}

func (g *GameSession) removeSnake(snake *Snake) {
	// Remove snake from entities and tree
	delete(g.World.Entities, snake.ID)
	g.World.Tree.Remove(snake.Entity)

	// Remove all segments
	for _, v := range snake.Segments {
		g.World.Tree.Remove(v)
		delete(g.World.Entities, v.ID)
	}

	// Now remove the snake itself.
	delete(g.World.Snakes, snake.ID)

}

// addPlayer will create a snake, add it to the game, and return the successful connection message to the player.
func (g *GameSession) addPlayer(ap AddPlayer) {
	newid := uint32(len(g.World.Entities))
	g.World.Snakes[newid] = NewSnake(newid, ap.Entity.Name)
	g.World.Entities[newid] = g.World.Snakes[newid].Entity
	g.World.Tree.Add(g.World.Snakes[newid].Entity)

	for _, s := range g.World.Snakes[newid].Segments {
		g.World.Entities[s.ID] = s
		g.World.Tree.Add(s)
	}

	g.Clients[ap.Client.ID] = &User{
		Account: nil,
		SnakeID: newid,
		GameID:  g.ID,
		Client:  ap.Client,
	}

	cgr := &messages.GameConnected{
		ID:       g.ID,
		TickID:   g.World.TickID,
		SnakeID:  newid,
		Entities: g.World.EntitiesMsg(),
		Snakes:   g.World.SnakesMsg(),
	}

	outgoing := NewOutgoingMsg(ap.Client, messages.GameConnectedMsgType, cgr)
	ap.Client.ToNetwork <- outgoing
}

func (g *GameSession) setDirection(facing *messages.Vect2, snakeID uint32) {
	// TODO: Allow a maximum turn rate so it can't just snap any direction
	snake := g.World.Snakes[snakeID]
	snake.Facing = physics.Vect2{X: facing.X, Y: facing.Y}
	snake.Facing = physics.NormalizeVect2(snake.Facing, 100)
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
func NewGame(toGameManager chan<- GameMessage, toNetwork chan<- OutgoingMessage) *GameSession {
	seed := uint64(rand.Uint32())
	seed = seed << 32
	seed += uint64(rand.Uint32())
	netchan := make(chan GameMessage, 100)
	g := &GameSession{
		IntoGameManager: toGameManager,
		FromGameManager: make(chan InternalMessage, 100),
		FromNetwork:     netchan,
		ToNetwork:       toNetwork,
		World:           NewWorld(),
		Exit:            make(chan int, 1),
		Clients:         make(map[uint32]*User, 16),
		commandHistory:  make([]GameMessage, 0, 10),
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
		Name:     s.Name,
	}
}

// NewSnake creates a new snake at a random location.
func NewSnake(id uint32, name string) *Snake {
	pos := physics.Vect2{
		X: int32(rand.Intn(10000) - 5000),
		Y: int32(rand.Intn(10000) - 5000),
	}
	snake := &Snake{
		Entity: &Entity{
			ID:       id,
			EType:    ETypeHead,
			Position: pos,
			Facing: physics.Vect2{
				X: 0,
				Y: 100,
			},
			Size: 600,
			Name: name,
		},
		Segments: []*Entity{},
		Speed:    2000,
	}
	for i := 0; i < 50; i++ {
		e := &Entity{
			ID:          id + uint32(i+1),
			EType:       ETypeSegment,
			ContainerID: id,
			Position: physics.Vect2{
				X: pos.X,
				Y: pos.Y - (snake.Size/2)*int32(i+1),
			},
			Facing: snake.Facing,
			Size:   600,
		}
		snake.Segments = append(snake.Segments, e)
	}
	return snake
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

// Intersects calculates if two entities overlap exactly.
// Used as a more fine-grained check after the quadtree check.
func (e *Entity) Intersects(o *Entity) bool {
	// TODO: (R0-R1)^2 <= (x0-x1)^2+(y0-y1)^2 <= (R0+R1)^2
	xdiff := (e.Position.X - o.Position.X)
	ydiff := (e.Position.Y - o.Position.Y)
	centerdist := xdiff*xdiff + ydiff*ydiff

	sizesum := e.Size + o.Size
	// This means either intersects OR contains
	return centerdist <= sizesum*sizesum

	// sizediff := (e.Size - o.Size)
	// return sizediff*sizediff <= centerdist // is the circle contained?
}

// BoundingBox is used for quadtree bounding checks.
func (e *Entity) BoundingBox() quadtree.BoundingBox {
	return quadtree.BoundingBox{
		MinX: e.Position.X - e.Size,
		MaxX: e.Position.X + e.Size,
		MinY: e.Position.Y - e.Size,
		MaxY: e.Position.Y + e.Size,
	}
}

// BoxID is used for quadtree intersection checks.
func (e *Entity) BoxID() uint32 {
	return e.ID
}

// GameMessage is a message from a client to a game.
type GameMessage struct {
	net         messages.Net
	client      *Client
	mtype       messages.MessageType
	currentTick uint32 // Tick when the game processed this mesage.
	clientID    uint32 // ID of client. Cached so you can clear client.
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
