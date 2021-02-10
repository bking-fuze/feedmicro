package main

import (
	"io"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var sess = session.New()
var s3svc = s3.New(sess)
var s3downloader = s3manager.NewDownloaderWithClient(s3svc)
var s3uploader = s3manager.NewUploader(sess)

type awsKeyFunc func (key string) bool

func awsList(bucket string, prefix string, startAfter string, onKey awsKeyFunc) error {
	return s3svc.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
		StartAfter: aws.String(startAfter),
	}, func (page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, item := range page.Contents {
			if !onKey(*item.Key) {
				return false
			}
		}
		return true
	})
}

func awsDownload(bucket string, key string) ([]byte, int64, error) {
	buff := &aws.WriteAtBuffer{}
	numBytes, err := s3downloader.Download(buff,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
	if err != nil {
		return nil, 0, err
	}
	return buff.Bytes(), numBytes, nil
}

func awsUpload(bucket string, key string, r io.Reader) (string, error) {
	result, err := s3uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key: aws.String(key),
		Body: r,
	})
	if err != nil {
		return "", err
	}
	return result.Location, nil
}
