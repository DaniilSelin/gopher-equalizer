package main

import (
	"fmt"
	"log"
	"net/http"
)

func startStub(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Stub on port %s got request: %s %s", port, r.Method, r.URL.Path)
		fmt.Fprintf(w, "Response from stub on port %s\n", port)
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		log.Printf("Starting stub on port %s...", port)
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Error starting stub on port %s: %v", port, err)
		}
	}()
}

func main() {
	ports := []string{"8081", "8082", "8083"}
	for _, port := range ports {
		startStub(port)
	}

	select {} // блокируемся
}
