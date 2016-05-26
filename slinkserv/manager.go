package slinkserv

import (
	"fmt"
	"log"
	"math"

	"github.com/lologarithm/slink/slinkserv/messages"
)

// GameStatus type used for setting status of a game
type GameStatus byte

// Game statuses
const (
	UnknownStatus GameStatus = 0
	RunningStatus GameStatus = iota
)

// GameManager manages all connected users and games.
type GameManager struct {
	// Player data
	Users      []*User
	Games      map[uint32]*GameSession
	NextGameID uint32 // TODO: this shouldn't just be a number..

	FromGames   chan GameMessage // Manager reads this only, all games created write only
	FromNetwork <-chan GameMessage
	ToNetwork   chan<- OutgoingMessage
	Exit        chan int

	// Temp junk to make this crap work
	Accounts   []*Account
	AccountID  uint32
	AcctByName map[string]*Account
}

// NewGameManager is the constructor for the main game manager.
// This should only be called once on a single server.
func NewGameManager(exit chan int, fromNetwork chan GameMessage, toNetwork chan OutgoingMessage) *GameManager {
	gm := &GameManager{
		Users:       make([]*User, math.MaxUint16),
		Games:       map[uint32]*GameSession{},
		FromGames:   make(chan GameMessage, 100),
		FromNetwork: fromNetwork,
		ToNetwork:   toNetwork,
		Exit:        exit,
		Accounts:    make([]*Account, math.MaxUint16),
		AcctByName:  map[string]*Account{},
	}
	return gm
}

// Run launches the game manager.
func (gm *GameManager) Run() {
	for {
		select {
		case netMsg := <-gm.FromNetwork:
			gm.ProcessNetMsg(netMsg)
		case gMsg := <-gm.FromGames:
			gm.ProcessGameMsg(gMsg)
		case <-gm.Exit:
			fmt.Printf("Manager got exit signal, shutting down all games.\n")
			for _, game := range gm.Games {
				game.Exit <- 1
			}
			fmt.Printf("  Shutdown sent to all games, manager closing now.\n")
			return
		}
	}
}

// ProcessNetMsg is the method by which the game manager can deal with incoming messages from the network.
func (gm *GameManager) ProcessNetMsg(msg GameMessage) {
	switch msg.mtype {
	case messages.DisconnectedMsgType:
		gm.handleDisconnect(msg)
	case messages.ConnectedMsgType:
		gm.handleConnection(msg)
	case messages.CreateAcctMsgType:
		gm.createAccount(msg)
	case messages.LoginMsgType:
		gm.loginUser(msg)
	case messages.JoinGameMsgType:
		gm.joinGame(msg)
		// TODO: make this work
	default:
		// These messages probably go to a game?
		// TODO: Probably have a direct conn to a game from the *Client
	}
}

func (gm *GameManager) joinGame(msg GameMessage) {
	if len(gm.Games) == 0 {
		gm.createGame(msg)
	}
	gameID := uint32(1) // TODO: scale multiple games!

	g := gm.Games[gameID]
	g.FromGameManager <- AddPlayer{
		Entity: &Entity{
			Name: gm.Users[msg.client.ID].Account.Name,
		},
		Client: msg.client,
	}
	gm.Users[msg.client.ID].GameID = gameID

	msg.client.FromGameManager <- ConnectedGame{
		ToGame: g.FromNetwork,
		ID:     gameID,
	}
}

func (gm *GameManager) createGame(msg GameMessage) {
	gm.NextGameID++

	g := NewGame(gm.FromGames, gm.ToNetwork)
	g.ID = gm.NextGameID
	go g.Run()
	log.Printf("Launched new game: %d", g.ID)
	gm.Games[gm.NextGameID] = g
}

func (gm *GameManager) handleConnection(msg GameMessage) {
	// First make sure this is a new connection.
	if gm.Users[msg.client.ID] == nil {
		// log.Printf("New user connected: %d", msg.client.ID)
		gm.Users[msg.client.ID] = &User{
			Client:  msg.client,
			Account: &Account{},
		}
	}
}

func (gm *GameManager) handleDisconnect(msg GameMessage) {
	log.Printf("GM: handling disconnect now: %d", msg.client.ID)
	// message active game that player disconnected.
	gameid := gm.Users[msg.client.ID].GameID
	if gm.Games[gameid] != nil {
		log.Printf("Signalling game %d to remove player %d.", gameid, msg.client.ID)
		gm.Games[gameid].FromGameManager <- RemovePlayer{Client: msg.client}
	}
	// Then clear out the user.
	gm.Users[msg.client.ID] = nil
}

func (gm *GameManager) createAccount(msg GameMessage) {
	netmsg := msg.net.(*messages.CreateAcct)
	ac := &messages.CreateAcctResp{
		AccountID: 0,
		Name:      netmsg.Name,
	}
	// log.Printf("Trying to login: %s", netmsg.Name)
	if _, ok := gm.AcctByName[netmsg.Name]; !ok {
		gm.AccountID++
		gm.Accounts[gm.AccountID] = &Account{
			ID:       gm.AccountID,
			Name:     netmsg.Name,
			Password: netmsg.Password,
		}

		ac.AccountID = gm.AccountID

		gm.AcctByName[netmsg.Name] = gm.Accounts[gm.AccountID]
		gm.Users[msg.client.ID].Account = gm.Accounts[gm.AccountID]
		// log.Printf("logged in: %s", netmsg.Name)
	}

	resp := NewOutgoingMsg(msg.client, messages.CreateAcctRespMsgType, ac)
	gm.ToNetwork <- resp
}

func (gm *GameManager) loginUser(msg GameMessage) {
	tmsg := msg.net.(*messages.Login)
	lr := messages.LoginResp{
		Success: 0,
		Name:    tmsg.Name,
	}
	if acct, ok := gm.AcctByName[tmsg.Name]; ok {
		if acct.Password == tmsg.Password {
			// log.Printf("Logging in account: %s", tmsg.Name)
			lr.AccountID = acct.ID
			gm.Users[msg.client.ID].Account = acct
		}
	}
	resp := NewOutgoingMsg(msg.client, messages.LoginRespMsgType, &lr)
	gm.ToNetwork <- resp
}

// ProcessGameMsg is used to process messages from an individual game to the main server controller.
func (gm *GameManager) ProcessGameMsg(msg GameMessage) {
	switch msg.mtype {
	}
}

// NewOutgoingMsg creates a new message that can be sent to a specific client.
func NewOutgoingMsg(dest *Client, tp messages.MessageType, msg messages.Net) OutgoingMessage {
	frame := messages.Frame{
		MsgType:       tp,
		Seq:           1,
		ContentLength: uint16(msg.Len()),
	}
	resp := OutgoingMessage{
		dest: dest,
		msg: messages.Packet{
			Frame:  frame,
			NetMsg: msg,
		},
	}
	return resp
}
