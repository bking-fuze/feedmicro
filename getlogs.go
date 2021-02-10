package main

import (
	"io"
	"time"
	"bytes"
	"regexp"
	"fmt"
	"archive/zip"
	"log"
)

const (
	LogLookbackTimeInHours = 3
	MaxDownloadRetries = 20
)

type getLogsOperation struct {
    token      string
    meetingId  int64
    instanceId int64
    beginTime  time.Time
    endTime    time.Time
}

type parsedKey struct {
	key		  string
	timestamp time.Time
	meeting	  string
	instance  string
}
var keyParseRE = regexp.MustCompile(`/Fuze-(\d\d\d\d-\d\d-\d\d-\d\d-\d\d-\d\d)\.zip$`)
func parseKey(key string) *parsedKey {
	m := keyParseRE.FindStringSubmatch(key)
	if m == nil {
		log.Printf("WARN: key %s did not match regex.", key)
		return nil
	}
	timestamp, err := time.Parse("2006-01-02-15-04-05", m[1])
	if err != nil {
		log.Printf("WARN: could not parse timestamp %s", m[1])
		return nil
	}
	return &parsedKey {
		key: key,
		timestamp: timestamp,
	}
}

type state struct {
	op        *getLogsOperation
	prior     string
	gathered  []string
}

func handleKey(key string, state *state) bool {
	pk := parseKey(key)
	if pk == nil {
		/* we skip keys we don't understand */
		return true
	}
	if state.gathered == nil {
		if pk.timestamp.After(state.op.beginTime) {
			if state.prior != "" {
				state.gathered = append(state.gathered, state.prior)
			}
			state.gathered = append(state.gathered, key)
		} else {
			state.prior = key
			return true
		}
	} else {
		state.gathered = append(state.gathered, key)
	}
	return !pk.timestamp.After(state.op.endTime)
}

/* it is tricky to retrieve logs between beginTime and endTime
   because the logs for an event at time T are usually in a file
   that started before T, and interesting logs sometimes wind
   up in subsequent files.

   So, we start listing S3 three hours before beginTime, but skip all
   until the last file prior to beginTime.
  
   We then include every file up to and including
   the first file after endTime.
 */

func getLogKeys(bucket string, op getLogsOperation) ([]string, error) {
	log.Printf("INFO: using time range: %s - %s", op.beginTime.Format(time.RFC3339), op.endTime.Format(time.RFC3339))
	state := state{ op: &op }
	scanTime := op.beginTime.Add(-LogLookbackTimeInHours * time.Hour)
	scanDir := op.token
	startDay := scanTime.Format("/2006/01/02/")
	startFile := scanTime.Format("Fuze-2006-01-02-15-04-05")
	startAfter := scanDir + startDay + startFile
	err := awsList(bucket, scanDir, startAfter, 
		func (key string) bool {
			return handleKey(key, &state)
		})
	if err != nil {
		return nil, err
	}
	return state.gathered, nil
}

func getLogs(w io.Writer, bucket string, keys []string) error {
	errorCount := 0
	for _, key := range keys {
		if err := getSingleLog(w, bucket, key, &errorCount); err != nil {
			return err
		}
	}
	return nil
}

func retryDownload(bucket string, key string, perrorCount *int) (buff []byte, numBytes int64, err error) {
	for {
		buff, numBytes, err = awsDownload(bucket, key)
		if err == nil {
			return
		}
		*perrorCount++
		log.Printf("ERROR: downloading %s: %s (try %d of %d)", key, err, *perrorCount, MaxDownloadRetries)
		if *perrorCount == MaxDownloadRetries {
			log.Printf("ERROR: giving up after %d retries", MaxDownloadRetries)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}


func getSingleLog(w io.Writer, bucket string, key string, perrorCount *int) error {
	buff, numBytes, err := retryDownload(bucket, key, perrorCount)
	if err != nil {
		return err
	}
	log.Printf("INFO: downloaded %s (%d bytes)", key, numBytes)
	zr, err := zip.NewReader(bytes.NewReader(buff), numBytes)
    if err != nil {
		return fmt.Errorf("could not read zip: %s", key, err)
    }
    for _, f := range zr.File {
        fr, err := f.Open()
        if err != nil {
			return fmt.Errorf("could not read zipentry: %s", key, err)
        }
        defer fr.Close()
		_, err = io.Copy(w, fr)
		if err != nil {
			return fmt.Errorf("could not copy zipentry: %s", key, err)
		}
    }
	return nil
}
