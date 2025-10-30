package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"simpleWT/backend"
)

func main() {

	// Starts a server

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
		if err1 != nil {
			log.Printf("Error starting server: %s", err1)
		}
	}()

	wt.Start()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		log.Printf("Error shutting down server: %s\n", err)
	}
	wt.Stop()

	log.Println("Shutting down server")
}
