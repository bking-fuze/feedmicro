package main

import (
	"net/http"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"context"
	"bytes"
	"crypto/rand"
	"strings"
	"io"
)

const (
	fileHeader = "log 1"
	newline = "\n"
)

type putLogHeader struct {
	Token     string `json:"token"`
	TimeZone  string `json:"tz"`
	Encoding  string `json:"encoding"`
}

func randomKey() (string, error) {
    randId := make([]byte, 8)
    n, err := io.ReadFull(rand.Reader, randId)
    if n != len(randId) || err != nil {
        return "", err
    }
    return fmt.Sprintf("/inbound/%s", hex.EncodeToString(randId)), nil
}

func putLog(context context.Context, header *putLogHeader, r io.Reader) (string, error) {
	hbytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	preamble := fmt.Sprintf("log 1\n%d\n", len(hbytes) + 1)
	mr := io.MultiReader(strings.NewReader(preamble),
	                     bytes.NewReader(hbytes),
						 strings.NewReader(newline),
						 r)
	key, err := randomKey()
	if err != nil {
		return "", err
	}
	url, err := awsUpload("mbk-upload-bucket", key, mr)
	if err != nil {
		return "", err
	}
	return url, nil
}

const (
	MaxInMemoryMultipartMB = 8
)

type logsPostResponse struct {
	URL  string `json:"url"`
	Code int    `json:"code"`
}

func logsPost(req *http.Request) func(http.ResponseWriter) {
    token := req.Header.Get("FZ-Devicetoken")
	if token == "" {
		return httpForbidden
	}
	ok, err := checkDeviceId(req.Context(), token)
	if err != nil {
		return httpInternalServerError
	} else if !ok {
		return httpForbidden
	}

	/* I believe that in production no one uses multipart;
	   we should clean this up at some point, so I am logging
	   the content-type */
    ct := req.Header.Get("Content-Type")
	log.Printf("INFO: content-type: %s", ct)
	var r io.Reader
	switch {
		case strings.HasPrefix(ct, "application/json"):
			fallthrough
		case strings.HasPrefix(ct, "text/plain"):
			r = req.Body
		case strings.HasPrefix(ct, "multipart/"):
			file, _, err := req.FormFile("request")
			if err == http.ErrMissingFile {
				log.Printf("INFO: missing file", ct)
				return httpBadRequest
			} else if err != nil {
				return httpInternalServerError
			}
    		defer file.Close()
			r = file
		default:
			log.Printf("INFO: illegal content-type: %s", ct)
			return httpBadRequest
	}

	url, err := putLog(req.Context(), &putLogHeader{
		Token: token,
		TimeZone: req.URL.Query().Get("tz"),
		Encoding: req.Header.Get("Content-Encoding"),
	}, r)
	if err != nil {
		log.Printf("ERROR: could not process request: %s", err)
		return httpInternalServerError
	}

	/* I'd prefer not to return the URL, but that's the existing interface */
	return jsonResponse(&logsPostResponse{ URL: url, Code: 200 })
}
