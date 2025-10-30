package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"simpleWT/backend"

	"github.com/go-faker/faker/v4"
)

func main() {

	// Simple Client flag
	cPtr := flag.Int("c", 1, "clients")
	flag.Parse()

	var clients []*backend.Client
	if *cPtr > 0 {
		for i := range *cPtr {
			go func() {
				c := connectClient(i)
				if c != nil {
					clients = append(clients, c)
				}
			}()
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	for _, client := range clients {
		client.Close()
	}
}

func connectClient(n int) *backend.Client {
	sran := rand.IntN(5)
	time.Sleep(time.Duration(sran) * time.Second)
	name := fmt.Sprintf("%s-%d", faker.Name(), n)
	log.Printf("Client: Connecting as: %s\n", name)
	c, err := backend.ClientConnect(backend.ClientConnection{Name: name})
	if err == nil {
		return c
	}
	return nil
}
