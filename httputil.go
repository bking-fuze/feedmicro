package main

import (
	"net/url"
	"net/http"
	"time"
	"strconv"
)

func queryStringItem(values url.Values, name string, pstr *string) {
	if _, ok := values[name]; ok {
		*pstr = values[name][0]
	}
}

func queryRFC3339Item(values url.Values, name string, ptime *time.Time, perr *error) {
	if *perr != nil {
		return
	}
	var value string
	queryStringItem(values, name, &value)
	if value == "" {
		return
	}
	*ptime, *perr = time.Parse(time.RFC3339, value)
}

func queryInt64Item(values url.Values, name string, pint *int64, perr *error) {
	if *perr != nil {
		return
	}
	var value string
	queryStringItem(values, name, &value)
	if value == "" {
		return
	}
	*pint, *perr = strconv.ParseInt(value, 10, 64)
}

func httpBadRequest(w http.ResponseWriter) {
	http.Error(w, "Bad Request", 400)
}

func httpInternalServerError(w http.ResponseWriter) {
	http.Error(w, "Internal Server Error", 500)
}

func httpForbidden(w http.ResponseWriter) {
	http.Error(w, "Forbidden", 403)
}
