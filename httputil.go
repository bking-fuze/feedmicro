package main

import (
	"net/url"
	"net/http"
)

func querySimpleItem(values url.Values, name string) string {
    var value string
    if _, ok := values[name]; ok {
        value = values[name][0]
    }
    return value
}

func httpBadRequest(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "Bad Request", 400)
}

func httpInternalServerError(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "Internal Server Error", 500)
}
