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

	static "github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	models "github.com/rohith2506/facedetect/models"
	redis "github.com/rohith2506/facedetect/redis"
	"github.com/rohith2506/facedetect/s3"
	utilities "github.com/rohith2506/facedetect/utilities"
)

const (
	tempDir     = "/tmp/images/"
	redisDB     = 0
	environment = "default"
	bucket      = "facedetection25"
)

var redisConn *redis.Connection

// RedisOutput ...
type RedisOutput struct {
	Landmarks []models.Detection
	ImageURL  string
}

var (
	availableExtensions = []string{".jpeg", ".jpg", ".png"}
)

// SetupRouter setups the default gin router
func SetupRouter() *gin.Engine {
	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20 // 8 MiB
	router.Use(static.Serve("/", static.LocalFile("./templates", true)))

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
func getExistingImage(imageHash string) (*RedisOutput, error) {
	var output *RedisOutput

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

func handleFaceDetection(tempImage string, c *gin.Context, start time.Time, imageExtension string) {
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
			"landmarks": cacheOutput.Landmarks,
			"image_url": cacheOutput.ImageURL,
			"time_took": elapsed.Milliseconds(),
		})
	} else {
		// Run the algorithm
		outputImageName := imageHash + filepath.Ext(imageExtension)
		landmarks := models.RunFaceDetection(outputImageName, tempImage)

		// get the image from s3
		connection := s3.GetAwsSession(environment)
		imageURL, err := connection.GetImageURL(outputImageName, bucket)
		if err != nil {
			log.Fatalf("Error in retrieving the image from aws s3: %v", err)
		}

		elapsed := time.Since(start)
		c.JSON(http.StatusOK, gin.H{
			"landmarks": landmarks,
			"image_url": imageURL,
			"time_took": elapsed.Milliseconds(),
		})

		// set the value in redis
		redisOutput := RedisOutput{
			Landmarks: landmarks,
			ImageURL:  imageURL,
		}

		redisValue, err := json.Marshal(redisOutput)
		if err != nil {
			log.Fatalf("Error in creating json marshal for redis output: %v", err)
		}
		err = redisConn.SetKey(imageHash, redisValue)
		if err != nil {
			log.Printf("Error in redis set: %v", err)
		}
	}

	// Delete the temp file
	err = os.Remove(tempImage)
	return
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
	_, found := utilities.Find(availableExtensions, imageExtension)
	if !found {
		c.JSON(http.StatusBadRequest, gin.H{"invalid image extension": "possible url extensions are [jpg, jpeg, png]. This limitation will be fixed soon."})
		return
	}

	// Create a temporary file
	tempImage := tempDir + uniqueImageID + imageExtension
	err = createTempFile(file, nil, tempImage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"temporary file creation failed": err.Error()})
		return
	}

	// Handle the face detection
	handleFaceDetection(tempImage, c, start, imageExtension)
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
		c.JSON(http.StatusBadRequest, gin.H{"invalid image extension": "possible extensions are [jpg, jpeg, png]. This limitation will be fixed soon."})
		return
	}

	// get the image from the URL
	response, err := http.Get(rawImageURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"image fetch failed": err})
		return
	}
	defer response.Body.Close()

	// create the temporary image
	uniqueImageID := utilities.RandStringBytes()
	tempImage := tempDir + uniqueImageID + imageExtension
	err = createTempFile(nil, response, tempImage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"temporary file creation failed": err.Error()})
		return
	}

	// Handle the face detection
	handleFaceDetection(tempImage, c, start, imageExtension)
}
