package slinkserv

import (
	"fmt"
	"testing"
)

func TestTick(t *testing.T) {
	tgm := make(chan GameMessage, 100)
	ton := make(chan OutgoingMessage, 100)

	g := NewGame(tgm, ton)
	g.addSnake(1, "test")
	g.setDirection(1, 1)

	for i := 0; i < 10; i++ {
		g.World.Tick()
	}

	g.setDirection(0, 1)

	for i := 0; i < 50; i++ {
		g.World.Tick()
	}
	entmsg := g.World.EntitiesMsg()
	for _, ent := range entmsg {
		fmt.Printf("  Ent %d @ (%v,%v)\n", ent.ID, ent.X, ent.Y)
	}
}
