package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"go.lepak.sg/mrtracker-backend/server"
)

const (
	envHost = "HOST"
	envPort = "PORT"

	defaultHost     = "0.0.0.0"
	defaultPort     = "8080"
	defaultPrivAddr = "0.0.0.0:9100" // TODO: restrict to prometheus bridge network only?
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	host, ok := os.LookupEnv(envHost)
	if !ok {
		host = defaultHost
	}

	port, ok := os.LookupEnv(envPort)
	if !ok {
		port = defaultPort
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	server.StartHttp(ctx, addr, defaultPrivAddr)
}
