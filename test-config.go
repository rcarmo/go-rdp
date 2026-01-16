package main

import (
	"fmt"
	"github.com/kulaginds/rdp-html5/internal/pkg/config"
)

func main() {
	opts := config.LoadOptions{
		Host:              "localhost",
		Port:              "8080",
		LogLevel:          "info",
		ConfigFile:        "",
		SkipTLSValidation: false,
		TLSServerName:     "",
	}
	cfg, err := config.LoadWithOverrides(opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Success: Host=%s, Port=%s\n", cfg.Server.Host, cfg.Server.Port)
}
