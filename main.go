package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var Cfg *Config

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("fatal error occurred:", r)
			os.Exit(1)
		}
	}()

	config, err := NewConfig()
	if err != nil {
		panic(err)
	}

	Cfg = config

	if Cfg.IsHelpRequested {
		Cfg.PrintUsage()
		os.Exit(0)
	}

	runHttpServer()
}

func runHttpServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		req, err := NewRequest(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
		}

		if req != nil {
			req.Print()
		}

		w.WriteHeader(http.StatusAccepted)
	})

	addr := fmt.Sprintf(":%d", Cfg.Port)
	server := &http.Server{Addr: addr, Handler: nil}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	log.Printf("Server is running on %s, ctrl+c to stop", addr)

	<-quit

	log.Println("Shutting down the server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
