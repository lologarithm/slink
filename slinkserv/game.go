package slinkserv

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/lologarithm/slink/slinkserv/messages"
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
	World     *GameWorld // Current world state
	StartTime time.Time

	// Historical state
	prevWorlds     [50]*GameWorld // Last 1 second of game states
	prevHead       uint32
	commandHistory []GameMessage // Last 1 second of commands
}

func (g *GameSession) replayHistory(ticks uint32) {
	// log.Printf("Replaying %d ticks.", ticks)
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
	// log.Printf("   Replayed back to tick: %d", g.World.TickID)
}

func (g *GameSession) applyCommand(msg GameMessage, echo bool) {
	switch msg.mtype {
	case messages.TurnSnakeMsgType:
		dirmsg := msg.net.(*messages.TurnSnake)
		g.setDirection(dirmsg.Direction, dirmsg.ID)
	case messages.DisconnectedMsgType:
		s, ok := g.World.Snakes[msg.clientID]
		if !ok {
			break
		}
		g.removeSnake(s)
	case messages.JoinGameMsgType:
		g.addSnake(msg.clientID, msg.clientName)
	default:
		fmt.Printf("game.go:Run(): UNKNOWN MESSAGE TYPE: %T\n", msg.net)
	}
}

func (g *GameSession) createHistoryPoint() {
	g.prevHead++
	if g.prevHead == 50 {
		g.prevHead = 0 // ring buffer!
	}
	g.prevWorlds[g.prevHead] = g.World.Clone()
}

func (g *GameSession) resetToHistory(tick uint32) uint32 {
	// lat := atomic.LoadInt64(&u.Client.latency) / 2
	// ticklag := uint32(float64(lat) / g.World.TickLength)
	// log.Printf("    Resetting from: %d to %d", g.World.TickID, tick)
	if tick > g.World.TickID {

	}
	ticklag := g.World.TickID - tick
	if ticklag > 50 {
		ticklag = 50
	} else if ticklag < 0 {
		ticklag = 0
	}
	// Reset world back a bit.
	if ticklag > 0 {
		if g.World.TickID < ticklag {
			ticklag = g.World.TickID
		}
		prevWorldIdx := g.prevHead - ticklag
		if ticklag > g.prevHead {
			prevWorldIdx = 50 - (ticklag - g.prevHead)
		}
		g.World = g.prevWorlds[prevWorldIdx].Clone()
		g.prevHead = g.prevHead
		// log.Printf("    World is now at tick: %d", g.World.TickID)
	}
	return ticklag
}

type Collision struct {
	Entity1 *Entity
	Entity2 *Entity
}

// Run starts the game!
func (g *GameSession) Run() {
	g.StartTime = time.Now()
	waiting := true
	waitms := int64(float64(time.Millisecond) * g.World.TickLength)
	nextTick := time.Now().UTC().UnixNano() + waitms
	ticktimes := [50]int64{}
	ttidx := 0

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
		tochan := time.After(timeout)
		for waiting {
			select {
			case <-tochan:
				waiting = false
				nextTick += waitms // Set next tick exactly
				break
			default:
				select {
				case msg := <-g.FromNetwork:
					msg.currentTick = g.World.TickID

					if setmsg, ok := msg.net.(*messages.TurnSnake); ok {
						// st := time.Now()
						log.Printf(" Client %d, setting direction: %v @ tick: %d", msg.clientID, setmsg.Direction, setmsg.TickID)
						// First check if this message is out of date!
						isOld := false
						for _, m := range g.commandHistory {
							if m.clientID == msg.clientID && m.mtype == msg.mtype {
								if m.currentTick >= setmsg.TickID {
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
						setmsg.ID = client.SnakeID

						if g.World.TickID >= setmsg.TickID {
							g.resetToHistory(setmsg.TickID - 1)
						}

						frame := messages.Frame{
							// Seq Doesn't matter because its set by server when its sent.
							MsgType:       messages.TurnSnakeMsgType,
							ContentLength: uint16(setmsg.Len()),
						}
						g.sendToAll(OutgoingMessage{
							msg: messages.Packet{
								Frame:  frame,
								NetMsg: setmsg,
							},
						})

						// log.Printf(" Applying turn took: %dus", time.Now().Sub(st).Nanoseconds()/int64(time.Microsecond))
					}

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
						if snake == nil {
							// wtf?
							break
						}
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
				default:
					time.Sleep(time.Microsecond)
					// Don't spinlock cause we don't wanna waste stuff.
					break
				}
			}
		}

		expectedTick := (time.Now().UnixNano() - g.StartTime.UnixNano()) / int64(g.World.TickLength*float64(time.Millisecond))
		for g.World.TickID < uint32(expectedTick) {
			st := time.Now()
			g.replayHistory(1)
			if g.World.TickID%250 == 0 { // every 5 seconds
				g.SendMasterFrame()
			}
			ttidx++
			if ttidx == 50 {
				ttidx = 0
			}
			ticktimes[ttidx] = time.Now().Sub(st).Nanoseconds() / int64(time.Microsecond)
			if g.World.TickID%50 == 0 { // every second
				var tt int64
				for _, v := range ticktimes {
					tt += v
				}
				fmt.Printf("  AVG TICK: %.1fμs\n", float64(tt)/50.0)
			}
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
	log.Printf("Adding new player to game %d: %s", g.ID, ap.Entity.Name)
	g.addSnake(newid, ap.Entity.Name)
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

	addcmd := GameMessage{
		clientID:    newid,
		clientName:  ap.Entity.Name,
		mtype:       messages.JoinGameMsgType,
		currentTick: g.World.TickID,
	}
	g.commandHistory = append(g.commandHistory, addcmd)

}

func (g *GameSession) addSnake(newid uint32, name string) {
	if g.World.Snakes[newid] != nil {
		return
	}
	g.World.Snakes[newid] = NewSnake(newid, name)
	g.World.Entities[newid] = g.World.Snakes[newid].Entity
	g.World.Tree.Add(g.World.Snakes[newid].Entity)

	for _, s := range g.World.Snakes[newid].Segments {
		g.World.Entities[s.ID] = s
		g.World.Tree.Add(s)
	}
}

func (g *GameSession) setDirection(facing int16, snakeID uint32) {
	snake := g.World.Snakes[snakeID]
	if snake == nil {
		return
	}
	g.World.Snakes[snakeID].Turning = facing
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

	// for _, ent := range mf.Entities {
	// 	for _, s := range mf.Snakes {
	// 		if ent.ID == s.ID {
	// 			log.Printf("  Master snake: %d, facing: %d,%d", s.ID, ent.Facing.X, ent.Facing.Y)
	// 		}
	// 	}
	// }

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

// GameMessage is a message from a client to a game.
type GameMessage struct {
	net         messages.Net
	client      *Client
	mtype       messages.MessageType
	currentTick uint32 // Tick when the game processed this mesage.

	clientID   uint32 // ID of client. Cached so you can clear client.
	clientName string // Name of client so you can clear client.
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
