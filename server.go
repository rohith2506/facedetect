package main

import (
	"encoding/json"
	"errors"
	image "image/jpeg"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	pigo "github.com/esimov/pigo/core"
	"github.com/gin-gonic/gin"
	detector "github.com/rohith2506/facedetect/detector"
	utilities "github.com/rohith2506/facedetect/utilities"
)

const (
	imageBaseDir = "/tmp"
	temporaryDir = "/temporary"
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

func storeTemporaryImage(imagePath string) ([]uint8, error) {
	reader, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	img, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}
	src := pigo.ImgToNRGBA(img)
	pixels := pigo.RgbToGrayscale(src)
	return pixels, nil
}

func checkExistingImage(imagePath string) []Detection {
	pixels, err := storeTemporaryImage(imagePath)
	if err != nil {
		return nil
	}
	conn := redis.CreateConnection(0)
	pixelBytes, _ := json.Marshal(pixels)
	value, err := conn.GetKey(string(pixelBytes))
	if len(value) != 0 {
		var dets []Detection
		if err := json.Unmarshal([]byte(value), &dets); err != nil {
			return nil
		}
		return dets
	}
	return nil
}

func imageUploadHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	imageID := utilities.RandStringBytes()
	imageFileName := imageID + "_" + file.Filename
	filename := filepath.Join(imageBaseDir, imageFileName)
	if err := c.SaveUploadedFile(file, filename); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	imagePath := imageBaseDir + "/" + temporaryDir + "/" + imageFileName

	detections := detector.DetectFaces(imagePath)

	c.JSON(http.StatusOK, gin.H{"facial landmarks": detections})
}

func imagePostHandler(c *gin.Context) {
	availableExtensions := []string{".jpeg", ".jpg", ".png"}

	rawImageURL := c.PostForm("image_url")
	imageURL, err := url.ParseRequestURI(rawImageURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"invalid_url_error": err})
		return
	}

	imageExtension := filepath.Ext(imageURL.Path)
	_, found := utilities.Find(availableExtensions, imageExtension)
	if !found {
		c.JSON(http.StatusBadRequest, gin.H{"invalid_image_extension": errors.New("possible extensions are [jpg, jpeg, png]")})
		return
	}

	response, err := http.Get(rawImageURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	defer response.Body.Close()

	imageID := utilities.RandStringBytes()
	imagePath := imageBaseDir + "/" + imageID + imageExtension
	file, err := os.Create(imagePath)
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

	detections := detector.DetectFaces(imagePath)
	c.JSON(http.StatusOK, gin.H{"result": detections})
}
