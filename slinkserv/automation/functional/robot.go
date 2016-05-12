package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/lologarithm/slink/slinkserv/automation"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	exit := make(chan int, 1)
	mu := automation.NewMockUser()
	connected := automation.Connect(mu)

	if connected {
		automation.CreateAccount(mu, "testuser1", "testuser2")
		go automation.ReadMessages(mu)
		go automation.RunUser(mu, exit)
	} else {
		c <- os.Interrupt
	}

	<-c
	exit <- 1

	time.Sleep(time.Second)
	log.Printf("Goodbye!")

}
