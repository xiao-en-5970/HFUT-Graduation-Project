package main

import (
	"log"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/bootstrap"
)

func main() {
	// Boot the application (initializes all components and starts the server)
	if err := bootstrap.Boot(); err != nil {
		log.Fatalf("Failed to bootstrap application: %v", err)
	}
}
