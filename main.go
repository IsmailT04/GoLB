package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

const backendURL string = "http://localhost:8081"

func main() {
	origin, err := url.Parse(backendURL)
	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(origin)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		log.Printf("LB Received request: %s %s", r.Method, r.URL.Path)

		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))

		proxy.ServeHTTP(w, r)
	})

	log.Println("Load Balancer started on :8080 -> Forwarding to :8081")

	// Start the Load Balancer server
	log.Fatal(http.ListenAndServe(":8080", nil))
}
