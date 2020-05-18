package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/go-playground/assert/v2"
)

func performURLRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	params := url.Values{}
	params.Add("image_url", "https://i.dailymail.co.uk/1s/2020/01/03/11/22946918-7848133-image-a-66_1578049904813.jpg")
	req, _ := http.NewRequest(method, path, strings.NewReader(params.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(params.Encode())))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestImagePostHandler(t *testing.T) {
	router := SetupRouter()
	w := performURLRequest(router, "POST", "/submit")
	assert.Equal(t, http.StatusOK, w.Code)

	response, err := ioutil.ReadAll(w.Body)
	wanted := "\"bounds\":{\"y\":83,\"x\":199,\"length\":87}"
	if err != nil || strings.Index(string(response), wanted) < 0 {
		t.Fail()
	}
}

func performUploadRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	w, _ := mw.CreateFormFile("file", "elon.jpg")
	reader, _ := ioutil.ReadFile("test_images/elon.jpg")
	_, err := w.Write(reader)
	if err != nil {
		log.Fatalf("error in creating multipart form: %v", w)
	}
	mw.Close()
	req, _ := http.NewRequest(method, path, buf)
	req.Header.Add("Content-Type", mw.FormDataContentType())
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req)
	return w2
}

func TestImageUploadHandler(t *testing.T) {
	router := SetupRouter()
	w := performUploadRequest(router, "POST", "/upload")
	assert.Equal(t, http.StatusOK, w.Code)

	response, err := ioutil.ReadAll(w.Body)
	wanted := "{\"bounds\":{\"y\":753,\"x\":1257,\"length\":986}"
	if err != nil || strings.Index(string(response), wanted) < 0 {
		t.Fail()
	}
}
