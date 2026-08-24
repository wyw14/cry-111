package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/wyw14/cry-111/internal/api"
)

func main() {
	address := flag.String("addr", "127.0.0.1:21211", "listen address")
	dataDir := flag.String("data", filepath.Join(os.TempDir(), "signalroute-data"), "persistent data directory")
	flag.Parse()
	application, err := api.NewApplication(*dataDir)
	if err != nil {
		log.Fatal(err)
	}
	server := api.NewServer(application)
	httpServer := &http.Server{Addr: *address, Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	errorsChannel := make(chan error, 1)
	go func() {
		errorsChannel <- httpServer.ListenAndServe()
	}()
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	select {
	case received := <-signalChannel:
		log.Printf("received %s", received)
	case serverError := <-errorsChannel:
		if !errors.Is(serverError, http.ErrServerClosed) {
			log.Fatal(serverError)
		}
	}
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownContext); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
