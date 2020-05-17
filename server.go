package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	detector "github.com/rohith2506/facedetect/detector"
)

func main() {
	router := gin.Default()
	s := &http.Server{
		Addr:           ":8000",
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	router.MaxMultipartMemory = 8 << 20 // 8 MiB
	router.Static("/", "./templates")
	router.POST("/upload", imageUploadHandler)
	router.POST("/submit", imagePostHandler)

	s.ListenAndServe()
}

func imageUploadHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filename := filepath.Join("/tmp", file.Filename)
	if err := c.SaveUploadedFile(file, filename); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	detections := detector.DetectFaces("/tmp/" + file.Filename)

	c.JSON(http.StatusOK, gin.H{"facial landmarks": detections})
}

func imagePostHandler(c *gin.Context) {
	imageURL := c.PostForm("image_url")
	fmt.Println("image URL: ", imageURL)

	if len(imageURL) != 0 {
		response, err := http.Get(imageURL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			return
		}
		defer response.Body.Close()

		file, err := os.Create("/tmp/dummy.jpg")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			return
		}
		defer file.Close()

		_, err = io.Copy(file, response.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			return
		}

		detections := detector.DetectFaces("/tmp/dummy.jpg")
		c.JSON(http.StatusOK, gin.H{"result": detections})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty url"})
	}
	return
}
