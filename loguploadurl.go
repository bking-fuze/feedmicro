package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"net/http"
)


/* bogus logic copied from meetings-goservices */
func isWardenToken(token string) bool {
	return len(token) > 41 && len(token) < 100
}

func checkToken(token string, req *http.Request) bool {
	if !isWardenToken(token) {
		return false
	} else if authHeaders, ok := req.Header["authorization"]; ok &&
		len(authHeaders[0]) > 6 &&
		strings.ToUpper(authHeaders[0][0:6]) == "BEARER" {
		return token == authHeaders[0][7:]
	} else if fzTokenHeaders, ok := req.Header["fz-token"]; ok {
		return token == fzTokenHeaders[0]
	} else {
		return false
	}
}

type uploadURLRequest struct {
	token           string
	fileName        string
	contentType     string
	deviceId        string
	year            string
	month           string
	day             string
	client          string
	downloadToken   string
	meetingInstance string
}

func normalPattern(uur uploadURLRequest) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", uur.deviceId, uur.year, uur.month, uur.day, uur.fileName)
}

func downloadPattern(uur uploadURLRequest) string {
	return fmt.Sprintf("download/%s/%s/%s/%s/%s", uur.downloadToken, uur.year, uur.month, uur.day, uur.fileName)
}

func unauthenticatedPattern(uur uploadURLRequest) string {
	return fmt.Sprintf("unauthenticated/%s/%s/%s/%s/%s", uur.client, uur.year, uur.month, uur.day, uur.fileName)
}

/* invalid/non-existing meeting IDs are ignored */
func readMeetingDate(ctx context.Context, uur *uploadURLRequest) error {
	id, err := strconv.ParseInt(uur.meetingInstance, 10, 64)
	if err != nil {
		log.Printf("WARN invalid meeting instance: %d", id)
		return nil
	}
	pStartedAt, err := dbMeetingInstanceStartedAt(ctx, id)
	if err != nil {
		return err
	}
	if pStartedAt == nil {
		log.Printf("WARN missing/null meeting instance started_at: %d", id)
		return nil
	}
	t := *pStartedAt
	uur.year = fmt.Sprintf("%04d", t.Year())
	uur.month = fmt.Sprintf("%02d", int(t.Month()))
	uur.day = fmt.Sprintf("%02d", int(t.Day()))
	return nil
}

type structuredResponse struct {
	SignedRequest string  `json:"signed_request"`
	Message       string  `json:"message"`
	Code          float64 `json:"code"`
	ContentType   string  `json:"content_type"`
}

func logUploadURLHandlerGet(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	q := req.URL.Query()
	uur := uploadURLRequest{}
	queryStringItem(q, "token", &uur.token)
	queryStringItem(q, "file_name", &uur.fileName)
	queryStringItem(q, "content_type", &uur.contentType)
	queryStringItem(q, "device_id", &uur.deviceId)
	queryStringItem(q, "year", &uur.year)
	queryStringItem(q, "month", &uur.month)
	queryStringItem(q, "day", &uur.day)
	queryStringItem(q, "client", &uur.client)
	queryStringItem(q, "download_token", &uur.downloadToken)
	queryStringItem(q, "meeting_instance", &uur.meetingInstance)

	var err error
	ok := true
	patternFunc := unauthenticatedPattern

	/* authentication rule and pattern selection */
	if len(uur.token) > 0 {
		patternFunc = normalPattern
		ok = checkToken(uur.token, req)
	} else if len(uur.deviceId) > 0 {
		patternFunc = normalPattern
		ok, err = checkDeviceId(ctx, uur.deviceId)
	} else if len(uur.downloadToken) > 0 {
		patternFunc = downloadPattern
		ok, err = checkDownloadToken(ctx, uur.downloadToken)
	}
	if err != nil {
		httpInternalServerError(w, req)
		return
	}
	if !ok {
		httpForbidden(w, req)
		return
	}

	/* special handling of parameters */
	if uur.client == "wininstaller" {
		if !strings.HasSuffix(uur.fileName, ".wininstaller.zip") {
			basename := strings.TrimSuffix(uur.fileName, ".zip")
			uur.fileName = basename + ".winstaller.zip"
		}
	}
	if len(uur.meetingInstance) > 0 {
		if len(uur.token) > 0 || len(uur.deviceId) > 0 {
			err = readMeetingDate(ctx, &uur)
			if err != nil {
				httpInternalServerError(w, req)
				return
			}
		}
	}
	if len(uur.token) > 0 && uur.deviceId == "ngbrowser" {
		uur.deviceId = uur.token
	}

	s3key := patternFunc(uur)
	signedUrl, err := awsMakeSignedUrl("mbk-upload-bucket", s3key)
	if err != nil {
		httpInternalServerError(w, req)
		return
	}

	body, err := json.Marshal(&structuredResponse{
		SignedRequest: signedUrl,
		Message:       "",
		Code:          200,
		ContentType:   uur.contentType,
	})
	if err != nil {
		httpInternalServerError(w, req)
		return
	}
	_, err = w.Write(body)
	if err != nil {
		httpInternalServerError(w, req)
		return
	}
}
