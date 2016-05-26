package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/lologarithm/slink/slinkserv"
)

func main() {
	exit := make(chan int, 10)
	complete := make(chan int, 1)
	fmt.Println("Starting Server!")
	// Launch server manager
	s := slinkserv.NewServer(exit)
	go slinkserv.RunServer(s, exit, complete)

	// f, err := os.Create(strconv.FormatInt(time.Now().Unix(), 10) + "_servercpu.prof")
	// if err != nil {
	// 	log.Fatalf("Could not create profile results file: \n%v", err)
	// } else {
	// 	pprof.StartCPUProfile(f)
	// }

	fmt.Println("Server started. Press a ctrl+c to exit.")

	// time.Sleep(60 * time.Second)
	// pprof.StopCPUProfile()
	// fmt.Println("CPU profile completed.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	fmt.Println("Shutting down server, please wait a few seconds for server to timeout.")
	exit <- 1
	exit <- 1 // twice, once for server, once for
	<-complete

	fmt.Println("Goodbye!")
	return
}
