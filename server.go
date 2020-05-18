package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
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
	inputDir = "/tmp/images/in/"
	tempDir  = "/tmp/images/temporary/"

	redisDB = 0
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

/*
When image gets uploaded
*/
func imageUploadHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"invalid input file error": err.Error()})
		return
	}

	// temporary image
	uniqueImageID := utilities.RandStringBytes()
	tempImage := tempDir + uniqueImageID

	// Create a temporary file
	err = createTempFile(file, nil, tempImage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"temp file creation error": err.Error()})
		return
	}

	// get the image hash
	imageHash, err := utilities.GetImageHash(tempImage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"file hash error": err.Error()})
		return
	}

	// Find whether there is an existing image or not
	cacheOutput, err := getExistingImage(imageHash)
	if err != nil {
		log.Panicf("Error in redis get: %v", err)
	}

	// Return from cache
	if cacheOutput != nil {
		c.JSON(http.StatusOK, gin.H{"result": cacheOutput.Landmarks})
	} else {
		// run the algorithm
		output := detector.DetectFaces(imageHash, tempImage)

		// Cache the image
		value, _ := json.Marshal(output)
		err = redisConn.SetKey(imageHash, value)
		if err != nil {
			log.Panicf("Error in redis set: %v", err)
		}
		c.JSON(http.StatusOK, gin.H{"result": output.Landmarks})
	}

	// delete the temporary image
	err = os.Remove(tempImage)
	return
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
		c.JSON(http.StatusBadRequest, gin.H{"invalid_image_extension": "possible extensions are [jpg, jpeg, png]"})
		return
	}

	// get the image from the URL
	response, err := http.Get(rawImageURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"url_fetch_error": err})
		return
	}
	defer response.Body.Close()

	// temporary image
	uniqueImageID := utilities.RandStringBytes()
	tempImage := tempDir + uniqueImageID

	// create the temporary image
	err = createTempFile(nil, response, tempImage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"temp file creation error": err.Error()})
		return
	}

	// get the image hash
	imageHash, err := utilities.GetImageHash(tempImage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"file hash error": err.Error()})
		return
	}

	// Find whether there is an existing image or not
	cacheOutput, err := getExistingImage(imageHash)
	if err != nil {
		log.Panicf("Error in redis get: %v", err)
	}

	// Return from cache
	if cacheOutput != nil {
		c.JSON(http.StatusOK, gin.H{"result": cacheOutput.Landmarks})
	} else {
		// Run the algorithm
		output := detector.DetectFaces(imageHash, tempImage)

		// Cache the image
		value, _ := json.Marshal(output)
		err = redisConn.SetKey(imageHash, value)
		if err != nil {
			log.Panicf("Error in redis set: %v", err)
		}
		c.JSON(http.StatusOK, gin.H{"result": output.Landmarks})
	}

	// Delete the temp file
	err = os.Remove(tempImage)
	return
}
