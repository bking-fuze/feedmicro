package main

import (
	"context"
	"log"
	"time"
)

func checkDeviceId(ctx context.Context, deviceId string) (ok bool, err error) {
	ok, err = dbDeviceExists(ctx, deviceId)
	if err != nil {
		log.Printf("ERROR checking device id: %s", err)
		return
	}
	if !ok {
		log.Printf("WARN no match for device id: %s", deviceId)
	}
	return
}

func checkDownloadToken(ctx context.Context, downloadToken string) (bool, error) {
	pCreatedAt, err := dbDownloadTokenCreatedAt(ctx, downloadToken)
	if err != nil {
		log.Printf("ERROR checking download token: %s", err)
		return false, err
	}
	if pCreatedAt == nil {
		log.Printf("WARN missing/null launch token created_at: %s", downloadToken)
		return false, nil
	}
	expiry := pCreatedAt.Add(time.Hour * 24)
	if time.Now().After(expiry) {
		log.Printf("WARN launch token has expired: %s", downloadToken)
		return false, nil
	}
	return true, nil
}

/*
func checkDeviceAndSession(ctx context.Context, downloadToken string) (string, bool, error) {
	return "", false, nil
}
 */
