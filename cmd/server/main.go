package main

import (
	"log"

	"github.com/MeleshinDA-1/metrics-collector/internal/config"
	"github.com/MeleshinDA-1/metrics-collector/internal/server"
)

func main() {
	serverConfig, err := config.ParseServerConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err := server.Run(serverConfig.Address); err != nil {
		log.Fatal(err)
	}
}
