package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

// Run this multiple times on different ports: go run deneme.go -port=8081
func main() {
	port := flag.String("port", "8081", "server port")
	flag.Parse()

	log.Printf("Backend server starting on port %s", *port)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from backend server on port %s\n", *port)
	})

	log.Printf("Backend server listening on :%s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
