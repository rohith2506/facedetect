package s3

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	awsZone = "eu-central-1"
	prodEnv = "default"
)

// Connection ...
type Connection struct {
	env  string
	sess *session.Session
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

var connection *Connection

// GetAwsSession ...
func GetAwsSession(environment string) *Connection {
	if connection == nil {
		sess, err := session.NewSession(&aws.Config{
			Region:      aws.String(awsZone),
			Credentials: credentials.NewSharedCredentials("", environment),
		})
		if err != nil {
			log.Fatalf("Error in creating aws session: %v", err)
		}
		connection = &Connection{
			env:  environment,
			sess: sess,
		}
	}
	return connection
}

// UploadFile ...
func (conn *Connection) UploadFile(imagePath string, imageID string, bucket string) error {
	uploader := s3manager.NewUploader(conn.sess)
	file, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(imageID),
		Body:   file,
	})
	if err != nil {
		return err
	}
	return nil
}

// GetImageURL ...
func (conn *Connection) GetImageURL(imageID string, bucket string) (string, error) {
	svc := s3.New(conn.sess)
	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(imageID),
	})
	urlStr, err := req.Presign(15 * time.Minute)
	if err != nil {
		return "", err
	}
	return urlStr, nil
}
