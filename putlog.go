package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
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
