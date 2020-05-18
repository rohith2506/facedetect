package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/go-playground/assert/v2"
)

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
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
	w := performRequest(router, "POST", "/submit")
	assert.Equal(t, http.StatusOK, w.Code)

	response, err := ioutil.ReadAll(w.Body)
	wanted := "\"face\":{\"y\":83,\"x\":199,\"length\":87}"
	if err != nil || strings.Index(string(response), wanted) < 0 {
		t.Fail()
	}
}
