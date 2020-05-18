package main

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	detector "github.com/rohith2506/facedetect/detector"
	redis "github.com/rohith2506/facedetect/redis"
	utilities "github.com/rohith2506/facedetect/utilities"
)

const (
	tempDir  = "/var/images/tmp/"
	inputDir = "/var/images/in/"
	redisDB  = 0
)

var redisConn *redis.Connection

var (
	availableExtensions = []string{".jpeg", ".jpg", ".png"}
)

func main() {
	router := gin.Default()
	redisConn = redis.CreateConnection(redisDB)
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

func getExistingImage(imageHash string) (*detector.RedisOutput, error) {
	var redisOutput *detector.RedisOutput

	if redisConn == nil {
		redisConn = redis.CreateConnection(redisDB)
	}

	// Get the value from redis
	value, err := redisConn.GetKey(imageHash)
	if err != nil {
		return redisOutput, err
	}

	// There is no existing key present. Just return nil
	if len(value) == 0 {
		return redisOutput, nil
	}

	// Parse the value to custom struct
	if err := json.Unmarshal([]byte(value), &redisOutput); err != nil {
		return redisOutput, err
	}
	return redisOutput, nil
}

// store the input image
func storeUploadedImage(file *multipart.FileHeader, c *gin.Context) (string, error) {
	uniqueImageID := utilities.RandStringBytes()
	imagePath := uniqueImageID + "_" + file.Filename
	imageAbsPath := filepath.Join(inputDir, imagePath)
	if err := c.SaveUploadedFile(file, imageAbsPath); err != nil {
		return "", err
	}
	return imagePath, nil
}

// store the url submitted image
func storeURLImage(response *http.Response, imageExtension string) (string, error) {
	var imagePath string
	uniqueImageID := utilities.RandStringBytes()
	imagePath = uniqueImageID + imageExtension
	file, err := os.Create(inputDir + "/" + imagePath)
	if err != nil {
		return imagePath, err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return imagePath, err
	}
	return imagePath, nil
}

/*
When image gets uploaded
*/
func imageUploadHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// get the image hash
	imageHash, err := utilities.GetImageHash(2, nil, file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"file hash error": err.Error()})
		return
	}

	// Find whether there is an existing image or not
	result, err := getExistingImage(imageHash)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"Redis get key error": err.Error()})
		return
	}

	// Return from cache
	if result != nil {
		c.JSON(http.StatusOK, gin.H{"result": result})
		return
	}

	// Store the image in /var/images/in
	imagePath, err := storeUploadedImage(file, c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"image store error": err})
	}

	detections := detector.DetectFaces(imagePath)

	// Cache the image
	value, _ := json.Marshal(result)
	err = redisConn.SetKey(imageHash, value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"Redis set key error": err.Error()})
	}

	c.JSON(http.StatusOK, gin.H{"facial landmarks": detections})
}

/*
when image gets submitted via URL
*/
func imagePostHandler(c *gin.Context) {
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

	// get the image from the URL
	response, err := http.Get(rawImageURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"url_fetch_error": err})
		return
	}
	defer response.Body.Close()

	// check whether the image exists in cache or not
	// get the image hash
	imageHash, err := utilities.GetImageHash(2, response, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"file hash error": err.Error()})
		return
	}

	// Find whether there is an existing image or not
	result, err := getExistingImage(imageHash)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"Redis get key error": err.Error()})
		return
	}

	// Return from cache
	if result != nil {
		c.JSON(http.StatusOK, gin.H{"result": result})
		return
	}

	// store the image in /var/images/in
	imagePath, err := storeURLImage(response, imageExtension)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"image store error": err})
	}

	result = detector.DetectFaces(imagePath)

	// Cache the image
	value, _ := json.Marshal(result)
	err = redisConn.SetKey(imageHash, value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"Redis set key error": err.Error()})
	}

	c.JSON(http.StatusOK, gin.H{"result": result})
}
