package main

import (
	"fmt"
	"net/http"
	"encoding/json"
	"strings"
	"time"
	"log"
	"io"
)

const (
	MaxGetLogRangeInHours = 14
	MaxInMemoryMultipartMB = 8
)

func authenticateGetLogs(w http.ResponseWriter, req *http.Request) bool {
	return true
}

func logsHandlerGet(w http.ResponseWriter, req *http.Request) {
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
		mi, err := dbGetMeetingInstanceInfo(req.Context(), glo.instanceId)
		if err != nil {
			log.Printf("ERROR: could not read start/end times for instance %d: %s", glo.instanceId, err)
			httpInternalServerError(w, req)
			return
		}
		if mi == nil {
			log.Printf("WARN: no meeting instance for id %d", glo.instanceId)
			httpBadRequest(w, req)
			return
		}
		glo.beginTime = mi.startedAt
		glo.endTime = mi.endedAt
	}
	zeroTime := time.Time{}
	if glo.endTime == zeroTime ||
	   int(glo.endTime.Sub(glo.beginTime).Hours()) > MaxGetLogRangeInHours {
		log.Printf("ERROR: invalid time range: %s - %s", glo.beginTime, glo.endTime)
		httpBadRequest(w, req)
		return
	}
	keys, err := getLogKeys("mbk-upload-bucket", glo)
	if err != nil {
		log.Printf("ERROR: could not obtain log keys: %s", err)
		httpInternalServerError(w, req)
		return
	}
	w.Header().Add("Trailer", "X-Streaming-Error")
	err = getLogs(w, "mbk-upload-bucket", keys)
	if err != nil {
		log.Printf("ERROR: trouble streaming result: %s", err)
		w.Header().Set("X-Streaming-Error", "true")
	} else {
		log.Printf("INFO: success")
		w.Header().Set("X-Streaming-Error", "false")
	}
}

func authenticatePostLogs(w http.ResponseWriter, req *http.Request) bool {
	return true
}

type logsPostResponse struct {
	URL  string `json:"url"`
	Code int    `json:"code"`
}

func logsHandlerPost(w http.ResponseWriter, req *http.Request) {
	if !authenticatePostLogs(w, req) {
		return
	}
	var r io.Reader
	var token string
	var err error

	/* I believe that in production no one uses multipart;
	   we should clean this up at some point, so I am logging
	   the content-type */
    ct := req.Header.Get("Content-Type")
	log.Printf("INFO: content-type: %s", ct)
	switch {
		case strings.HasPrefix(ct, "application/json"):
			fallthrough
		case strings.HasPrefix(ct, "text/plain"):
			r = req.Body
			token = req.URL.Query().Get("token")
		case strings.HasPrefix(ct, "multipart/"):
			file, _, err := req.FormFile("request")
			if err == http.ErrMissingFile {
				log.Printf("INFO: missing file", ct)
				httpBadRequest(w, req)
				return
			} else if err != nil {
				httpInternalServerError(w, req)
				return
			}
    		defer file.Close()
			r = file
			token = req.FormValue("token")
		default:
			log.Printf("INFO: illegal content-type: %s", ct)
			httpBadRequest(w, req)
			return
	}

	if token == "" {
		log.Printf("INFO: missing token")
		httpBadRequest(w, req)
		return
	}

	url, err := putLog(req.Context(), &putLogHeader{
		Token: token,
		TimeZone: req.URL.Query().Get("tz"),
		Encoding: req.Header.Get("Content-Encoding"),
	}, r)
	if err != nil {
		log.Printf("ERROR: could not process request: %s", err)
		httpInternalServerError(w, req)
		return
	}

	/* I'd prefer not to leak the URL, but that's the existing interface */
	response, err := json.Marshal(&logsPostResponse{ URL: url, Code: 200 })
	if err != nil {
		log.Printf("ERROR: could marshal response: %s", err)
		httpInternalServerError(w, req)
		return
	}
	_, err = w.Write(response)
	if err != nil {
		log.Printf("ERROR: could not write response: %s", err)
	}
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
