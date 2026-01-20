package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

// Run this multiple times on different ports: go run main.go -port=8081
func main() {
	port := flag.String("port", "8081", "server port")
	flag.Parse()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from backend server on port %s\n", *port)
	})
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
