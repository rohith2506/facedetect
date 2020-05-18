package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	static "github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	detector "github.com/rohith2506/facedetect/detector"
	redis "github.com/rohith2506/facedetect/redis"
	utilities "github.com/rohith2506/facedetect/utilities"
)

const (
	inputDir = "/tmp/images/in/"
	tempDir  = "/tmp/images/temporary/"

	redisDB = 0
)

var redisConn *redis.Connection

var (
	availableExtensions = []string{".jpeg", ".jpg", ".png"}
)

// SetupRouter setups the default gin router
func SetupRouter() *gin.Engine {
	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20 // 8 MiB
	router.Use(static.Serve("/", static.LocalFile("./templates", true)))
	router.Use(static.Serve("/images", static.LocalFile("/tmp/images/out", true)))

	router.POST("/upload", ImageUploadHandler)
	router.POST("/submit", ImagePostHandler)

	return router
}

// Main function
func main() {
	router := SetupRouter()
	redisConn = redis.CreateConnection(redisDB)
	s := &http.Server{
		Addr:           ":8000",
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	s.ListenAndServe()
}

// Checks whether the image already exists in codebase
func getExistingImage(imageHash string) (*detector.ImageOutput, error) {
	var output *detector.ImageOutput

	if redisConn == nil {
		redisConn = redis.CreateConnection(redisDB)
	}

	// Get the value from redis
	value, err := redisConn.GetKey(imageHash)
	if err != nil {
		return output, err
	}

	// There is no existing key present. Just return nil
	if len(value) == 0 {
		return output, nil
	}

	// Parse the value to custom struct
	if err := json.Unmarshal([]byte(value), &output); err != nil {
		return output, err
	}
	return output, nil
}

// create the temporary image
func createTempFile(multipartFile *multipart.FileHeader, response *http.Response, imagePath string) error {
	if multipartFile == nil && response == nil {
		return errors.New("Both Upload and URL retrieval are empty")
	}
	file, err := os.Create(imagePath)
	defer file.Close()
	if multipartFile == nil {
		_, err = io.Copy(file, response.Body)
	} else {
		temp, _ := multipartFile.Open()
		_, err = io.Copy(file, temp)
	}
	if err != nil {
		return err
	}
	return nil
}

// ImageUploadHandler endpoint is responsible for handling uploaded images
func ImageUploadHandler(c *gin.Context) {
	start := time.Now()
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"invalid input file": err.Error()})
		return
	}

	// temporary image
	uniqueImageID := utilities.RandStringBytes()
	imageExtension := filepath.Ext(file.Filename)
	fmt.Println("image extension: ", imageExtension)
	_, found := utilities.Find(availableExtensions, imageExtension)
	if !found {
		c.JSON(http.StatusBadRequest, gin.H{"invalid image extension": "possible extensions are [jpg, jpeg, png]"})
		return
	}

	tempImage := tempDir + uniqueImageID + imageExtension

	// Create a temporary file
	err = createTempFile(file, nil, tempImage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"temporary file creation failed": err.Error()})
		return
	}

	// get the image hash
	imageHash, err := utilities.GetImageHash(tempImage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"image hash generation failed": err.Error()})
		return
	}

	// Find whether there is an existing image or not
	cacheOutput, err := getExistingImage(imageHash)
	if err != nil {
		log.Printf("Redis error get failed: %v", err)
	}

	// Return from cache
	if cacheOutput != nil {
		elapsed := time.Since(start)

		c.JSON(http.StatusOK, gin.H{
			"landmarks":   cacheOutput.Landmarks,
			"output_file": cacheOutput.ImagePath,
			"time_took":   elapsed.Milliseconds(),
		})
	} else {
		// run the algorithm
		output := detector.DetectFaces(imageHash, tempImage)

		// Cache the image
		value, _ := json.Marshal(output)
		err = redisConn.SetKey(imageHash, value)
		if err != nil {
			log.Printf("Error in redis set: %v", err)
		}
		elapsed := time.Since(start)
		c.JSON(http.StatusOK, gin.H{
			"landmarks":   output.Landmarks,
			"output_file": output.ImagePath,
			"time_took":   elapsed.Milliseconds(),
		})
	}

	// delete the temporary image
	err = os.Remove(tempImage)
	return
}

// ImagePostHandler endpoint is responsible for handling URL images
func ImagePostHandler(c *gin.Context) {
	start := time.Now()
	rawImageURL := c.PostForm("image_url")

	imageURL, err := url.ParseRequestURI(rawImageURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"invalid url": err})
		return
	}

	// Look for file extension (whether this is png, jpg, jpeg)
	imageExtension := filepath.Ext(imageURL.Path)
	_, found := utilities.Find(availableExtensions, imageExtension)
	if !found {
		c.JSON(http.StatusBadRequest, gin.H{"invalid image extension": "possible extensions are [jpg, jpeg, png]"})
		return
	}

	// get the image from the URL
	response, err := http.Get(rawImageURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"image fetch failed": err})
		return
	}
	defer response.Body.Close()

	// temporary image
	uniqueImageID := utilities.RandStringBytes()
	tempImage := tempDir + uniqueImageID + imageExtension

	// create the temporary image
	err = createTempFile(nil, response, tempImage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"temporary file creation failed": err.Error()})
		return
	}

	// get the image hash
	imageHash, err := utilities.GetImageHash(tempImage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"image hash generation failed": err.Error()})
		return
	}

	// Find whether there is an existing image or not
	cacheOutput, err := getExistingImage(imageHash)
	if err != nil {
		log.Printf("Redis get failed: %v", err)
	}

	// Return from cache
	if cacheOutput != nil {
		elapsed := time.Since(start)
		c.JSON(http.StatusOK, gin.H{
			"landmarks":   cacheOutput.Landmarks,
			"output_file": cacheOutput.ImagePath,
			"time_took":   elapsed.Milliseconds(),
		})
	} else {
		// Run the algorithm
		output := detector.DetectFaces(imageHash, tempImage)

		// Cache the image
		value, _ := json.Marshal(output)
		err = redisConn.SetKey(imageHash, value)
		if err != nil {
			log.Printf("Error in redis set: %v", err)
		}
		elapsed := time.Since(start)
		c.JSON(http.StatusOK, gin.H{
			"landmarks":   output.Landmarks,
			"output_file": output.ImagePath,
			"time_took":   elapsed.Milliseconds(),
		})
	}

	// Delete the temp file
	err = os.Remove(tempImage)
	return
}
