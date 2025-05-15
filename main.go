package main

import (
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(".")))

	server := http.Server{}
	server.Addr = ":8080"
	server.Handler = mux
	server.ListenAndServe()
}
