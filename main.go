package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("LLMGateway listening on %s", addr)
	if err := http.ListenAndServe(addr, NewHandler()); err != nil {
		log.Fatal(err)
	}
}
