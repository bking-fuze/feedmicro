package main

import (
	"io"
	"bytes"
	"time"
	"regexp"
	"fmt"
	"archive/zip"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"log"
)

const (
	LogLookbackTimeInHours = 3
)

type getLogsOperation struct {
    token      string
    meetingId  int64
    instanceId int64
    beginTime  time.Time
    endTime    time.Time
}

var sess = session.New()
var s3svc = s3.New(sess)

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
		}
		return true
	}
	state.gathered = append(state.gathered, key)
	return !pk.timestamp.After(state.op.endTime)
}

func handlePage(page *s3.ListObjectsV2Output, lastPage bool, state *state) bool {
	if *page.KeyCount == 0 {
		return true
	}
	for _, item := range page.Contents {
		if !handleKey(*item.Key, state) {
			return false
		}
	}
	return true
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

func getLogs(w io.Writer, bucket string, op getLogsOperation) error {
	state := state{ op: &op }
	scanTime := op.beginTime.Add(-LogLookbackTimeInHours * time.Hour)
	scanDir := op.token
	scanDay := scanTime.Format("/2006/01/02/")
	scanFile := scanTime.Format("Fuze-2006-01-02-15-04-05")
	err := s3svc.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(scanDir),
		StartAfter: aws.String(scanDir + scanDay + scanFile),
	}, func (page *s3.ListObjectsV2Output, lastPage bool) bool {
		return handlePage(page, lastPage, &state)
	})
	if err != nil {
		return err
	}
	for _, key := range state.gathered {
		err = getSingleLog(w, bucket, key)
		if err != nil {
			return err
		}
	}
	return nil
}

func getSingleLog(w io.Writer, bucket string, key string) error {
	buff := &aws.WriteAtBuffer{}
	downloader := s3manager.NewDownloaderWithClient(s3svc)
	numBytes, err := downloader.Download(buff,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
	if err != nil {
		return fmt.Errorf("could not download %s: %s", key, err)
	}
	log.Printf("INFO: downloaded %s (%d bytes)", key, numBytes)
	zr, err := zip.NewReader(bytes.NewReader(buff.Bytes()), numBytes)
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