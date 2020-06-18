package s3

import (
	"strings"
	"testing"
)

func TestGetImageURL(t *testing.T) {
	connection := GetAwsSession("default")
	wanted := "https://facedetection25.s3.eu-central-1.amazonaws.com/elon.jpg"
	expected, err := connection.GetImageURL("elon.jpg", "facedetection25")
	if err != nil {
		t.Fatalf("error: %v", err.Error())
	}
	if strings.Contains(wanted, expected) {
		t.Fail()
	}
}
