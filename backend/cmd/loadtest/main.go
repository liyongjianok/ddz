package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"ddz/backend/internal/loadtest"
)

func main() {
	cfg := loadtest.Config{}

	flag.StringVar(&cfg.BaseURL, "base-url", "http://127.0.0.1:8080", "backend base url")
	flag.StringVar(&cfg.Mode, "mode", "classic", "room mode")
	flag.IntVar(&cfg.BaseScore, "base-score", 1, "room base score")
	flag.IntVar(&cfg.TotalConnections, "connections", 30, "total websocket connections")
	flag.IntVar(&cfg.Concurrency, "concurrency", 20, "concurrent room workers")
	flag.IntVar(&cfg.ReadyRooms, "ready-rooms", 0, "number of rooms to send one ready action")
	flag.DurationVar(&cfg.HoldDuration, "hold", 10*time.Second, "connection hold duration")
	flag.DurationVar(&cfg.HTTPTimeout, "http-timeout", 10*time.Second, "http request timeout")
	flag.DurationVar(&cfg.ConnectTimeout, "connect-timeout", 10*time.Second, "websocket connect timeout")
	flag.Parse()

	result, err := loadtest.Run(context.Background(), cfg)
	if err != nil {
		printResult(result)
		fmt.Fprintf(os.Stderr, "load smoke failed: %v\n", err)
		os.Exit(1)
	}

	printResult(result)
}

func printResult(result loadtest.Result) {
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal result: %v\n", err)
		return
	}
	fmt.Println(string(output))
}
