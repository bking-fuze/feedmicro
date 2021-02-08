package main

import (
//	"io"
//	"fmt"
	"net/http"
)

const (
	MaxGetLogRangeInHours = 14
)

func authenticateGetLogs(w http.ResponseWriter, req *http.Request) bool {
	return true
}

func logsHandlerGet(w http.ResponseWriter, req *http.Request) {
	if !authenticateGetLogs(w, req) {
		return
	}
	q := req.URL.Query()
	rr := rawGetLogsRequest{};
	rr.token = querySimpleItem(q, "token")
	rr.meetingId = querySimpleItem(q, "meeting_id")
	rr.instanceId = querySimpleItem(q, "instance_id")
	rr.beginTime = querySimpleItem(q, "begin_time")
	rr.endTime = querySimpleItem(q, "end_time")
	ro, err := prepareGetLogsOperation(&rr)
	if err != nil {
		httpInternalServerError(w, req)
		return
	}
	if ro == nil {
		httpBadRequest(w, req)
		return
	}
	if ro.endTime.Sub(ro.beginTime).Hours() > MaxGetLogRangeInHours {
		httpBadRequest(w, req)
		return
	}
	getLogs(w, ro)
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
