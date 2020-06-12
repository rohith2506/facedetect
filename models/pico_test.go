package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDetectPico(t *testing.T) {
	imagePath := "../test_images/elon.jpg"
	result, _ := json.Marshal(DetectPico(imagePath))
	wanted := "{\"bounds\":{\"y\":753,\"x\":1257,\"width\":986,\"height\":986}"
	if err != nil || strings.Index(string(result), wanted) < 0 {
		t.Fail()
	}
}
