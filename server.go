package main

import (
	"log"
	"fmt"
	"net/http"
)

func logsHandler(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
		case "GET":
			logsGet(req)(w)
		case "POST":
			logsPost(req)(w)
		default:
			httpBadRequest(w)
	}
}

func healthHandler(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
		case "GET":
			fmt.Fprintf(w, "ok\n")
		default:
			httpBadRequest(w)
	}
}

func logUploadURLHandler(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
		case "GET":
			logUploadURLGet(req)(w)
		default:
			httpBadRequest(w)
	}
}

func main() {
	log.SetFlags(0)
	err := dbOpen("admin:zCIrMi3TnJ1BOHYoiR05@tcp(database-1.cluster-cwntao8rxnbn.us-east-2.rds.amazonaws.com:3306)/testdb")
	if err != nil {
		log.Fatal(err)
	}
	defer dbClose()
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/v1/logs", logsHandler)
	http.HandleFunc("/v2/logs", logsHandler)
	http.HandleFunc("/v1/log_upload_url", logUploadURLHandler)
	http.ListenAndServe(":8080", nil)
}
