package main

import (
	"log"
	"net/http"

	"github.com/brianmaksy/chatserver/internal/handlers"
)

func main() {
	mux := routes()

	log.Println("Starting channel listener")
	go handlers.ListenToWsChannel()

	log.Println("starting web server on port 8070")

	_ = http.ListenAndServe(":8070", mux)
}
