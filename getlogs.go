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

type rawGetLogsRequest struct {
    token      string
    meetingId  string
    instanceId string
    beginTime  string
    endTime    string
}

type preparedGetLogsOperation struct {
    token      string
    meetingId  int64
    instanceId int64
    beginTime  time.Time
    endTime    time.Time
}

func prepareGetLogsOperation(rr *rawGetLogsRequest) (*preparedGetLogsOperation, error) {
	return nil, nil
}

var sess = session.New()
var s3svc = s3.New(sess)

/* it is tricky to retrieve logs between startTime and endTime
   because the logs for an event at time T are usually in a file
   that started before T, and interesting logs sometimes wind
   up in subsequent files.

   So, we start listing S3 three hours before startTime, but skip all
   until the last file prior to startTime.
  
   We then include every file up to and including
   the first file after endTime.
 */

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
	startTime time.Time
	endTime time.Time
	lastKey string
	printing bool
	w	io.Writer
}

func emit(key string, w io.Writer) error {
	buff := &aws.WriteAtBuffer{}
	downloader := s3manager.NewDownloaderWithClient(s3svc)
	numBytes, err := downloader.Download(buff,
		&s3.GetObjectInput{
			Bucket: aws.String("mbk-upload-bucket"),
			Key:    aws.String(key),
		})
	if err != nil {
		return fmt.Errorf("could not download %s: %s", key, err)
	}
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
		/* hack */
		w.Write([]byte("\n"))
    }
	return nil
}

func handleKey(key string, state *state) bool {
	pk := parseKey(key)
	if pk == nil {
		/* skip files we can't handle */
		return true
	}
	if !state.printing {
		if pk.timestamp.After(state.startTime) {
			if state.lastKey != "" {
				emit(state.lastKey, state.w)
			}
			emit(key, state.w)
			state.printing = true
		} else {
			state.lastKey = key
		}
		return true
	}
	emit(key, state.w)
	return !pk.timestamp.After(state.endTime)
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

func stream(startTime time.Time, endTime time.Time, w io.Writer) error {
	state := state{
		startTime: startTime,
		endTime: endTime,
		w: w,
	}
	scanTime := startTime.Add(-3 * time.Hour)
	scanDir := scanTime.Format("date-files/2006/01/02/")
	scanFile := scanTime.Format("Fuze-2006-01-02-15-04-05")
	err := s3svc.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: aws.String("mbk-upload-bucket"),
		Prefix: aws.String("date-files"),
		StartAfter: aws.String(scanDir + scanFile),
	}, func (page *s3.ListObjectsV2Output, lastPage bool) bool {
		return handlePage(page, lastPage, &state)
	})
	if err != nil {
		return err
	}
	return nil
}

func getLogs(w io.Writer, op *preparedGetLogsOperation) {
}
