package main

import (
	"log"
	"net/http"
)

func main() {
	log.SetFlags(0)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/v1/logs", logsHandler)
	http.HandleFunc("/v2/logs", logsHandler)
	http.ListenAndServe(":8080", nil)
}
