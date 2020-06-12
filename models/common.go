package models

import (
	"errors"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"

	pigo "github.com/esimov/pigo/core"
	"github.com/fogleman/gg"
	"github.com/nfnt/resize"
)

var (
	dc  *gg.Context
	dst io.Writer
	fn  *os.File
)

// Model Constants
const (
	PicoModel    = 1
	MTCNNModel   = 2
	outputDir    = "/tmp/images/out"
	adjustedCols = 300
	adjustedRows = 400
)

// Coord ...
type Coord struct {
	Row int `json:"x,omitempty"`
	Col int `json:"y,omitempty"`
}

// RectCoord ...
type RectCoord struct {
	Row    int `json:"y,omitempty"`
	Col    int `json:"x,omitempty"`
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
}

// Detection ...
type Detection struct {
	FaceCoord RectCoord `json:"bounds,omitempty"`
	LeftEye   Coord     `json:"left_eye,omitempty"`
	RightEye  Coord     `json:"right_eye,omitempty"`
	Mouth     []Coord   `json:"mouth,omitempty"`
	Nose      Coord     `json:"nose,omitempty"`
}

func createOutputFile(imageHash string) {
	imageExtension := filepath.Ext(imagePath)
	outputImagePath := imageHash + imageExtension
	createOutputFile(outputImagePath)
	fn, err = os.OpenFile(outputDir+outputImagePath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Fatalf("Unable to open output file: %v", err)
	}
	dst = fn
}

func encodeImage(dst io.Writer) error {
	var err error
	img := dc.Image()
	newImage := resize.Resize(adjustedRows, adjustedCols, img, resize.Lanczos3)

	switch dst.(type) {
	case *os.File:
		ext := filepath.Ext(dst.(*os.File).Name())
		switch ext {
		case "", ".jpg", ".jpeg":
			err = jpeg.Encode(dst, newImage, &jpeg.Options{Quality: 100})
		case ".png":
			err = png.Encode(dst, newImage)
		default:
			err = errors.New("unsupported image format")
		}
	default:
		err = jpeg.Encode(dst, newImage, &jpeg.Options{Quality: 100})
	}
	return err
}

func drawImages(imageHash string, faces []Detection) {
	createOutputFile(imageHash)

	dc = gg.NewContext(cols, rows)
	dc.DrawImage(src, 0, 0)

	landmarks, err := drawFaces(faces)

	if err := encodeImage(dst); err != nil {
		log.Fatalf("Error encoding the output image: %v", err)
	}
	defer fn.Close()
}

func drawFaces(faces []Detection) {
	for _, face := range faces {
		// Draw the face
		dc.DrawRectangle(
			float64(face.FaceCoord.Col-face.FaceCoord.Width/2),
			float64(face.FaceCoord.Row-face.FaceCoord.Height/2),
			float64(face.FaceCoord.Width),
			float64(face.FaceCoord.Height),
		)
		dc.SetLineWidth(2.0)
		dc.SetStrokeStyle(gg.NewSolidPattern(color.RGBA{R: 255, G: 0, B: 0, A: 255}))
		dc.Stroke()
		// left eye		
		drawDetections(dc, float64(face.), float64(leftEye.Row), float64(leftEye.Scale), color.RGBA{R: 255, G: 0, B: 0, A: 255}, true)
		// right eye		
		drawDetections(dc, float64(rig.Col), float64(leftEye.Row), float64(leftEye.Scale), color.RGBA{R: 255, G: 0, B: 0, A: 255}, true)
	}
}

// RunFaceDetection ....
func RunFaceDetection(imageHash string, imagePath string, modelToRun int) []Detection {
	// Find the facial landmarks
	var result []Detection
	if modelToRun == PicoModel {
		result = DetectPico(imagePath)
	} else {
		result = DetectMTCNN(imagePath)
	}

	go drawImages(imageHash)

	return result
}
