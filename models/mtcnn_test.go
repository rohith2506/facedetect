package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDetectMTCNN(t *testing.T) {
	imagePath := "../test_images/elon.jpg"
	result, _ := json.Marshal(DetectMTCNN(imagePath))
	wanted := "{\"y\":909,\"x\":298,\"width\":705,\"height\":987}"
	if err != nil || strings.Index(string(result), wanted) < 0 {
		t.Fail()
	}
}
