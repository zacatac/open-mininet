package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func cleanup() {
	fmt.Println("cleanup")
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		cleanup()
		os.Exit(1)
	}()

	name := flag.String("name", "default", "vm name")
	flag.Parse()
	if *name == "" {
		os.Exit(10)
	}

	for {
		fp, err := os.OpenFile("/tmp/"+*name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}

		fmt.Fprintf(fp, "%s:%d alive\n", *name, os.Getpid())
		fp.Close()

		time.Sleep(5 * time.Second)
	}
}
