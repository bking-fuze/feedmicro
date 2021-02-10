package main

import (
	"log"
	"net/http"
)

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
	http.ListenAndServe(":8080", nil)
}
