package slinkserv

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/lologarithm/slink/slinkserv/messages"
	"github.com/lologarithm/survival/physics"
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
	prevWorlds     [historySize]*GameWorld // Last 1 second of game states. Each state is 10 ticks(50 ticks/sec)
	prevHead       uint32
	commandHistory []GameMessage // Last 1 seconds of commands
}

const historySize = 5
const histTime = (50 / historySize)

func (g *GameSession) replayHistory(ticks uint32) {
	// log.Printf("Replaying %d ticks.", ticks)
	for i := uint32(0); i < ticks; i++ {
		// 1. Apply and commands from this tick
		for _, msg := range g.commandHistory {
			if msg.currentTick == g.World.CurrentTickID {
				g.applyCommand(msg, false)
			}
		}
		// 2. call tick
		collisions := g.World.Tick()
		if g.World.CurrentTickID > g.World.RealTickID {
			g.World.RealTickID = g.World.CurrentTickID
		}
		for _, col := range collisions {
			switch col.Entity.EType {
			case ETypeFood:
				delete(g.World.Entities, col.Entity.ID)
				found := g.World.Tree.Remove(col.Entity)
				if !found {
					log.Printf("Failed to remove food %d!?", col.Entity.ID)
					panic("error removing snake from world.")
				}
				col.Snake.Size += (col.Entity.Size / 2)
				numsegs := int32(len(col.Snake.Segments))
				for (col.Snake.Size / 30) > numsegs {
					g.World.MaxID++
					e := &Entity{
						ID:       g.World.MaxID,
						EType:    ETypeSegment,
						Position: col.Snake.Segments[numsegs-1].Position,
						Facing:   col.Snake.Segments[numsegs-1].Facing,
						Size:     col.Snake.Size,
					}
					col.Snake.Segments = append(col.Snake.Segments, e)
					g.World.Entities[g.World.MaxID] = e
					g.World.Tree.Add(e)
					numsegs = int32(len(col.Snake.Segments))
				}
				for _, seg := range col.Snake.Segments {
					seg.Size = col.Snake.Size
				}
				fmt.Printf("Snake %d ate a food, size is now: %d\n", col.Snake.ID, col.Snake.Size)
				g.sendEat(col.Snake, col.Entity)
			case ETypeSegment:
				// Snake ded
				g.sendDied(col.Snake.ID)
				g.removeSnake(col.Snake)
				spawns := make([]*messages.UpdateEntity, len(col.Snake.Segments))
				// 2. convert body to food.
				for i, seg := range col.Snake.Segments {
					g.World.MaxID++
					g.addEntity(g.World.MaxID, ETypeFood, seg.Position, (seg.Size / 3))
					entMsg := g.World.Entities[g.World.MaxID].toMsg()
					spawns[i] = &messages.UpdateEntity{Ent: entMsg}
				}
				g.sendSpawns(spawns)
			case ETypeHead:
				// TODO: Bigger snake wins!
			}
		}

		// 3.
		if g.World.CurrentTickID%histTime == 0 {
			g.createHistoryPoint()
		}
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
	case messages.EntityMsgType:
		tm := msg.net.(*messages.Entity)
		g.addEntity(tm.ID, tm.EType, physics.Vect2{X: tm.X, Y: tm.Y}, tm.Size)
	default:
		fmt.Printf("game.go:Run(): UNKNOWN MESSAGE TYPE: %T\n", msg.net)
	}
}

func (g *GameSession) createHistoryPoint() {
	g.prevHead++
	if g.prevHead == historySize {
		g.prevHead = 0 // ring buffer!
	}
	g.prevWorlds[g.prevHead] = g.World.Clone()
}

func (g *GameSession) resetToHistory(tick uint32) {
	// log.Printf("    Resetting from: %d to %d", g.World.CurrentTickID, tick)

	// Reset world back a bit.
	if tick < g.World.CurrentTickID {
		prevWorldIdx := g.prevHead
		for i := 0; i < historySize; i++ {
			if prevWorldIdx == 0 {
				prevWorldIdx = historySize - 1
			} else {
				prevWorldIdx--
			}
			if g.prevWorlds[prevWorldIdx].CurrentTickID < tick {
				break
			}
		}

		realtick := g.World.RealTickID
		maxid := g.World.MaxID
		g.World = g.prevWorlds[prevWorldIdx].Clone()
		g.World.RealTickID = realtick // Make sure to preserve what tick it really is.
		g.World.MaxID = maxid
		g.prevHead = prevWorldIdx
		// log.Printf("    World is now at tick: %d", g.World.CurrentTickID)
	}
}

type Collision struct {
	Snake  *Snake
	Entity *Entity
}

// Run starts the game!
func (g *GameSession) Run() {
	g.StartTime = time.Now()
	waitms := int64(float64(time.Millisecond) * g.World.TickLength)
	nextTick := time.Now().UTC().UnixNano() + waitms
	ticktimes := [50]int64{}
	ttidx := 0

	for {
		// Now create a clone of the world to add to the historical world data
		if g.World.RealTickID%histTime == 0 {
			g.createHistoryPoint()
			// Now scan through command history looking for old commands that are
			// before the oldest stored tick state.
			newHist := make([]GameMessage, len(g.commandHistory))
			hidx := 0
			for _, m := range g.commandHistory {
				if m.currentTick >= g.World.RealTickID-50 {
					newHist[hidx] = m
					hidx++
				}
			}
			g.commandHistory = newHist[:hidx]
		}

		// figure out timeout before each waiting loop.
		for {
			if nextTick-time.Now().UTC().UnixNano() <= 0 {
				nextTick += waitms // Setup the tick after this one before processing.
				break
			}
			// If we didn't timeout, try a non-blocking select here.
			select {
			case msg := <-g.FromNetwork:
				msg.currentTick = g.World.RealTickID
				if setmsg, ok := msg.net.(*messages.TurnSnake); ok {
					if g.World.RealTickID-setmsg.TickID > 50 {
						break // Ignore messages that are too old
					}
					msg.currentTick = setmsg.TickID // Turn messages act like they were received in the past.
					client := g.Clients[msg.clientID]
					if client == nil {
						break // Client is no longer connected.
					}
					setmsg.ID = client.SnakeID

					// log.Printf("Handling snake %d turning (%d) at tick: %d", setmsg.ID, setmsg.Direction, setmsg.TickID)
					if g.World.CurrentTickID >= setmsg.TickID {
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
					if g.World.RealTickID == g.World.CurrentTickID {
						snake := g.World.Snakes[user.SnakeID]
						if snake == nil {
							log.Printf("Removing player that doesn't have a snake in game!?")
							// wtf?
							break
						}
						g.removeSnake(snake)
					}
					removecmd := GameMessage{
						clientID:    user.SnakeID,
						mtype:       messages.DisconnectedMsgType,
						currentTick: g.World.RealTickID - 1,
					}
					g.commandHistory = append(g.commandHistory, removecmd)
				}
			case <-g.Exit:
				fmt.Printf("EXITING: Game %d.\n", g.ID)
				return
			default:
				time.Sleep(time.Microsecond)
				// Don't spinlock cause we don't wanna waste stuff.
			}
		}

		expectedTick := (time.Now().UnixNano() - g.StartTime.UnixNano()) / int64(g.World.TickLength*float64(time.Millisecond))
		st := time.Now()
		for g.World.CurrentTickID < uint32(expectedTick) {
			g.replayHistory(1) // Replay one tick at a time
			// fmt.Printf("   Running Tick: %d", g.World.CurrentTickID)
			// Advance 'real' state if the current state has caught up.
			if g.World.CurrentTickID == g.World.RealTickID {
				if g.World.RealTickID%50 == 0 {
					numspawn := 100
					spawns := make([]*messages.UpdateEntity, numspawn)
					// Spawn food if players are around
					if len(g.Clients) > 0 || len(g.World.Entities) < 100000 {
						for i := 0; i < numspawn; i++ {
							g.World.MaxID++
							x := rand.Intn((MapInternalSize)*2) - (MapInternalSize)
							y := rand.Intn((MapInternalSize)*2) - (MapInternalSize)

							g.addEntity(g.World.MaxID, ETypeFood, physics.Vect2{X: int32(x), Y: int32(y)}, int32(rand.Intn(50)+50))
							entMsg := g.World.Entities[g.World.MaxID].toMsg()
							g.commandHistory = append(g.commandHistory, GameMessage{
								net:         entMsg,
								mtype:       messages.EntityMsgType,
								currentTick: g.World.CurrentTickID - 1,
							})
							spawns[i] = &messages.UpdateEntity{Ent: entMsg}
						}
						// fmt.Printf("Total entities: %d\n", len(g.World.Entities))
					}
					if spawns[0] != nil { // Only send spawn messages if we are spawning.
						g.sendSpawns(spawns)
					}
				}
				// fmt.Printf("  RealTick: %d", g.World.RealTickID)
				if g.World.RealTickID%250 == 0 { // every 5 seconds
					g.SendMasterFrame()
				}
			}
		}

		ttidx++
		if ttidx == 50 {
			ttidx = 0
		}
		ticktimes[ttidx] = time.Now().Sub(st).Nanoseconds() / int64(time.Microsecond)

		if g.World.RealTickID%50 == 0 { // every second
			var tt int64
			for _, v := range ticktimes {
				tt += v
			}
			fmt.Printf("  AVG TICK: %.1fμs. Num Players: %d\n", float64(tt)/50.0, len(g.Clients))
		}
	}
}

func (g *GameSession) removeSnake(snake *Snake) {
	log.Printf("Removing snake id: %d", snake.ID)
	if g.World.Entities[snake.ID] == nil {
		log.Printf("Snake isn't in world entities.")
	}
	// Remove snake from entities and tree
	delete(g.World.Entities, snake.ID)
	found := g.World.Tree.Remove(snake.Entity)
	if !found {
		log.Printf("Failed to remove snake %d!?", snake.ID)
		panic("error removing snake from world.")
	}

	// Remove all segments
	for _, v := range snake.Segments {
		found := g.World.Tree.Remove(v)
		if !found {
			log.Printf("Failed to remove snake seg %d!?", v.ID)
			panic("error removing")
		}
		delete(g.World.Entities, v.ID)
	}

	// Now remove the snake itself.
	delete(g.World.Snakes, snake.ID)
}

func (g *GameSession) addEntity(ID uint32, etype uint16, loc physics.Vect2, size int32) {
	g.World.Entities[ID] = &Entity{
		ID:       ID,
		EType:    etype,
		Size:     size,
		Position: loc,
	}
	g.World.Tree.Add(g.World.Entities[ID])
}

// addPlayer will create a snake, add it to the game, and return the successful connection message to the player.
func (g *GameSession) addPlayer(ap AddPlayer) {
	newid := (g.World.MaxID + 1)
	snake := NewSnake(newid, ap.Entity.Name)
	g.World.MaxID += 1 + uint32(len(snake.Segments))
	// log.Printf("Adding new player to game %d: %s", g.ID, ap.Entity.Name)
	if g.World.RealTickID == g.World.CurrentTickID {
		g.addSnake(newid, ap.Entity.Name)
	} else {
		log.Printf("Added player but game isn't live(%d) yet, not adding snake until tick (%d).", g.World.CurrentTickID, g.World.RealTickID)
	}
	g.Clients[ap.Client.ID] = &User{
		Account: nil,
		SnakeID: newid,
		GameID:  g.ID,
		Client:  ap.Client,
	}

	cgr := &messages.GameConnected{
		ID:       g.ID,
		TickID:   g.World.RealTickID,
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
		currentTick: g.World.RealTickID - 1,
	}
	g.commandHistory = append(g.commandHistory, addcmd)
}

func (g *GameSession) addSnake(newid uint32, name string) {
	log.Printf("Adding snake after tick: %d", newid)
	if g.World.Snakes[newid] != nil {
		log.Printf("Snake already exists!?!?: %d", newid)
		panic("tried to add snake twice.")
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
	msg.data = msg.msg.Pack()
	for _, c := range g.Clients {
		msg.dest = c.Client
		g.ToNetwork <- msg
	}
}
func (g *GameSession) sendDied(snakeID uint32) {
	removeSnake := &messages.SnakeDied{
		ID: snakeID,
	}
	frame := messages.Frame{
		MsgType:       messages.SnakeDiedMsgType,
		Seq:           1,
		ContentLength: uint16(removeSnake.Len()),
	}
	g.sendToAll(OutgoingMessage{
		msg: messages.Packet{
			Frame:  frame,
			NetMsg: removeSnake,
		},
	})
}
func (g *GameSession) sendEat(snake *Snake, food *Entity) {
	msg := NewOutgoingMsg(nil, messages.RemoveEntityMsgType, &messages.RemoveEntity{
		Ent: food.toMsg(),
	})
	g.sendToAll(msg)
	msg = NewOutgoingMsg(nil, messages.UpdateEntityMsgType, &messages.UpdateEntity{
		Ent: snake.Entity.toMsg(),
	})
	g.sendToAll(msg)

}

// sendSpawns
func (g *GameSession) sendSpawns(spawns []*messages.UpdateEntity) {
	for _, s := range spawns {
		frame := messages.Frame{
			MsgType:       messages.UpdateEntityMsgType,
			Seq:           1,
			ContentLength: uint16(s.Len()),
		}
		g.sendToAll(OutgoingMessage{
			msg: messages.Packet{
				Frame:  frame,
				NetMsg: s,
			},
		})
	}
}

// SendMasterFrame will create a 'master' state of all things and send to each client.
func (g *GameSession) SendMasterFrame() {
	mf := &messages.GameMasterFrame{
		ID:       g.ID,
		Entities: g.World.EntitiesMsg(),
		Snakes:   g.World.SnakesMsg(),
		Tick:     g.World.RealTickID,
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

// GameMessage is a message from a client to a game.
type GameMessage struct {
	net         messages.Net
	client      *Client
	mtype       messages.MessageType
	currentTick uint32 // Tick when the game processed this mesage.
	clientID    uint32 // ID of client. Cached so you can clear client.
	clientName  string // Name of client so you can clear client.
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
