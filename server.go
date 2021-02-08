package main

import (
	"fmt"
	"net/http"
)

func healthHandler(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
		case "GET":
			fmt.Fprintf(w, "ok\n")
		default:
			http.Error(w, "Bad Request", 400)
	}
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/v1/logs", logsHandler)
	http.HandleFunc("/v2/logs", logsHandler)
	http.ListenAndServe(":8080", nil)
}
