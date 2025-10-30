package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-faker/faker/v4"

	"simpleWT/backend"
)

func main() {

	// Combines main.go and client.go
	// Starts a server and can make clients.

	cPtr := flag.Int("c", 0, "clients")
	flag.Parse()

	mux := http.NewServeMux()

	wt := backend.NewWebTransportServer()
	chain := backend.Chain{backend.WithCORS}
	mux.Handle("/login", chain.ThenFunc(wt.HandleLogin))

	// Http server
	server := &http.Server{
		Addr:    ":8770",
		Handler: mux,
	}

	go func() {
		err1 := server.ListenAndServe()
		if !errors.Is(err1, http.ErrServerClosed) {
			log.Printf("Error starting server: %s", err1)
		}
	}()

	wt.Start()

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		log.Printf("Error shutting down server: %s\n", err)
	}

	for _, client := range clients {
		client.Close()
	}

	wt.Stop()

	log.Println("Shutting down server")
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
