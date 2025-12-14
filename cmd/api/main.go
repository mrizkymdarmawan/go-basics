package main

import (
	"log"

	"go-basics/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("application failed to start: %v", err)
	}
}
