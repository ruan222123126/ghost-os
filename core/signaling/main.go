package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	defaultBindAddr         = ":8090"
	defaultHeartbeatSeconds = 45
)

func main() {
	config, err := loadConfigFromEnv()
	if err != nil {
		fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server, err := NewServer(config)
	if err != nil {
		fatal(err)
	}
	log.Printf("signaling listening addr=%s", config.BindAddr)
	if err := server.Run(ctx, config.BindAddr); err != nil {
		fatal(err)
	}
}

func loadConfigFromEnv() (Config, error) {
	timeoutSeconds, err := positiveIntEnv("GHOST_SIGNALING_HEARTBEAT_SECONDS", defaultHeartbeatSeconds)
	if err != nil {
		return Config{}, err
	}
	return Config{
		BindAddr:         envDefault("GHOST_SIGNALING_BIND_ADDR", defaultBindAddr),
		Token:            strings.TrimSpace(os.Getenv("GHOST_SIGNALING_TOKEN")),
		HeartbeatTimeout: time.Duration(timeoutSeconds) * time.Second,
	}, nil
}

func envDefault(name string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func positiveIntEnv(name string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid %s: must be > 0", name)
	}
	return value, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
