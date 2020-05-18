package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/assert/v2"
)

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	params := map[string]interface{}{"image_url": "https://i.dailymail.co.uk/1s/2020/01/03/11/22946918-7848133-image-a-66_1578049904813.jpg"}
	jsonBytes, _ := json.Marshal(params)
	contentBuffer := bytes.NewBuffer(jsonBytes)
	req, _ := http.NewRequest(method, path, contentBuffer)
	req.Header.Add("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestImagePostHandler(t *testing.T) {
	router := SetupRouter()
	w := performRequest(router, "POST", "/submit")
	assert.Equal(t, http.StatusOK, w.Code)
}
