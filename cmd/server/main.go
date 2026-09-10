package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/MeleshinDA-1/metrics-collector/internal/config"
	"github.com/MeleshinDA-1/metrics-collector/internal/server"
)

func main() {
	serverConfig, err := config.ParseServerConfig()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, serverConfig); err != nil {
		log.Fatal(err)
	}
}
