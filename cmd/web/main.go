// Filename: cmd/web/main.go

package main

import (
	"log"
	"net/http"
)

func handlerWS(w http.ResponseWriter, r *http.Request){
    w.Write([]byte("WebSockets!\n"))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handlerWS)
	log.Print("Starting server on :4000")
	err := http.ListenAndServe(":4000", mux)
	log.Fatal(err)
}
