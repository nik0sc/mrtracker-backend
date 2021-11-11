package main

import (
	"context"
	"os"
	"os/signal"

	"go.lepak.sg/mrtracker-backend/server"
)

const (
	envAddr = "ADDR"

	defaultAddr     = "0.0.0.0:8080"
	defaultPromAddr = "127.0.0.1:9100"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	addr, ok := os.LookupEnv(envAddr)
	if !ok {
		addr = defaultAddr
	}

	server.StartHttp(ctx, addr, defaultPromAddr)
}
