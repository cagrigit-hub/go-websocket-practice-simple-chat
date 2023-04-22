package main

import (
	"log"
	"net/http"
	"ws/internal/handlers"
)

func main() {
	mux := routes()
	log.Println("Starting channel listener")
	go handlers.ListenToWsChannel()
	log.Println("Starting server on :8080")

	_ = http.ListenAndServe(":8080", mux)
}
