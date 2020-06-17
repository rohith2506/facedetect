package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func main() {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("eu-central-1"),
		Credentials: credentials.NewSharedCredentials("", "default"),
	})
	svc := s3.New(sess)
	result, err := svc.ListBuckets(nil)
	if err != nil {
		exitErrorf("Unable to list buckets, %v", err)
	}
	fmt.Println("Buckets:")
	for _, b := range result.Buckets {
		fmt.Printf("* %s created on %s\n",
			aws.StringValue(b.Name), aws.TimeValue(b.CreationDate))
	}

	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	if err != nil {
		exitErrorf("Unable to list items in bucket %q, %v", bucket, err)
	}

	for _, item := range resp.Contents {
		fmt.Println("Name:         ", *item.Key)
		fmt.Println("Last modified:", *item.LastModified)
		fmt.Println("Size:         ", *item.Size)
		fmt.Println("Storage class:", *item.StorageClass)
		fmt.Println("")
	}

	// upload file to the bucket
	filename := "../test_images/elon.jpg"
	file, err := os.Open(filename)
	if err != nil {
		exitErrorf("Unable to open file %q, %v", err)
	}
	defer file.Close()
	uploader := s3manager.NewUploader(sess)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("elon.jpg"),
		Body:   file,
	})

	if err != nil {
		// Print the error and exit.
		exitErrorf("Unable to upload %q to %q, %v", filename, bucket, err)
	}
	fmt.Printf("Successfully uploaded %q to %q\n", filename, bucket)

	// Download the file
	bucket := "facedetection25"
	item := "elon.jpg"
	file, err := os.Create(item)
	downloader := s3manager.NewDownloader(sess)
	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(item),
		})
	if err != nil {
		exitErrorf("Unable to download item %q, %v", item, err)
	}
	fmt.Println("Downloaded", file.Name(), numBytes, "bytes")

}
