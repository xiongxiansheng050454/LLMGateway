package main

import (
	"log"
	"net/http"
	"os"

	"LLMGateway/internal/server"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("LLMGateway listening on %s", addr)
	if err := http.ListenAndServe(addr, server.NewHandler("dashboard")); err != nil {
		log.Fatal(err)
	}
}
