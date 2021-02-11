package main

import (
	"net/http"
)

func reportPost(crash bool, v2 bool, req *http.Request) func(http.ResponseWriter) {
	return httpOk
}
