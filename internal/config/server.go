package config

import "os"

type ServerConfig struct {
	Address         string `env:"ADDRESS" flag:"a,default=localhost:8080" usage:"HTTP server address"`
	StoreInterval   int    `env:"STORE_INTERVAL" flag:"i,default=300" usage:"metrics store interval in seconds"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" flag:"f,default=metricsDataDefault" usage:"metrics storage path"`
	Restore         bool   `env:"RESTORE" flag:"r,default=true" usage:"should load metrics data from storage"`
}

func ParseServerConfig() (ServerConfig, error) {
	return ParseConfig[ServerConfig](os.Args[1:])
}
