package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/lologarithm/slink/slinkserv"
)

func main() {
	exit := make(chan int, 1)

	fmt.Println("Starting Server!")
	// Launch server manager
	s := slinkserv.NewServer(exit)
	go slinkserv.RunServer(s, exit)

	f, err := os.Create(strconv.FormatInt(time.Now().Unix(), 10) + "_servercpu.prof")
	if err != nil {
		log.Fatalf("Could not create profile results file: \n%v", err)
	} else {
		pprof.StartCPUProfile(f)
	}

	fmt.Println("Server started. Press a ctrl+c to exit.")

	time.Sleep(60 * time.Second)
	pprof.StopCPUProfile()
	fmt.Println("CPU profile completed.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	fmt.Println("Goodbye!")
	exit <- 1
	return
}
