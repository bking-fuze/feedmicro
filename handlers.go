package main

import (
	"fmt"
	"net/http"
	"time"
	"log"
)

const (
	MaxGetLogRangeInHours = 14
)

func authenticateGetLogs(w http.ResponseWriter, req *http.Request) bool {
	return true
}

func logsHandlerGet(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Trailer", "X-Streaming-Error")

	if !authenticateGetLogs(w, req) {
		return
	}
	var err error
	q := req.URL.Query()
	glo := getLogsOperation{};
	queryStringItem(q, "token", &glo.token)
	queryInt64Item(q, "meeting_id", &glo.meetingId, &err)
	queryInt64Item(q, "instance_id", &glo.instanceId, &err)
	queryRFC3339Item(q, "begin_time", &glo.beginTime, &err)
	queryRFC3339Item(q, "end_time", &glo.endTime, &err)
	if err != nil {
		log.Printf("ERROR: malformed query: %s", err)
		httpBadRequest(w, req)
		return
	}
	if glo.token == "" {
		log.Printf("ERROR: missing token")
		httpBadRequest(w, req)
		return
	}
	if glo.meetingId != 0 || glo.instanceId != 0 {
		/* normally we would replace begin/end with the values from the meeting */
		httpInternalServerError(w, req)
		return
	}
	log.Printf("INFO: glo: %+v", glo)
	zeroTime := time.Time{}
	if glo.endTime == zeroTime ||
	   int(glo.endTime.Sub(glo.beginTime).Hours()) > MaxGetLogRangeInHours {
		log.Printf("ERROR: invalid time range: %s - %s", glo.beginTime, glo.endTime)
		httpBadRequest(w, req)
	}
	keys, err := getLogKeys(w, "mbk-upload-bucket", glo)
	if err != nil {
		log.Printf("ERROR: could not obtain log keys: %s", err)
		httpInternalServerError(w, req)
	}
	err = getLogs(w, "mbk-upload-bucket", keys)
	if err != nil {
		log.Printf("ERROR: trouble streaming result: %s", err)
		w.Header().Set("X-Streaming-Error", "true")
	} else {
		log.Printf("INFO: success")
		w.Header().Set("X-Streaming-Error", "false")
	}
}

func logsHandlerPost(w http.ResponseWriter, req *http.Request) {
	httpInternalServerError(w, req)
}

func logsHandler(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
		case "GET":
			logsHandlerGet(w, req)
		case "POST":
			logsHandlerPost(w, req)
		default:
			httpBadRequest(w, req)
	}
}

func healthHandler(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
		case "GET":
			fmt.Fprintf(w, "ok\n")
		default:
			httpBadRequest(w, req)
	}
}
