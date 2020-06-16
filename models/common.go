package models

import (
	"errors"
	"image"
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
	outputDir    = "/tmp/images/out/"
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

// Let's create the output file
func createOutputFile(imagePath string) {
	fn, err := os.OpenFile(imagePath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Fatalf("Unable to open output file: %v", err)
	}
	dst = fn
}

// encode the image
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

func drawImages(imagePath string, faces []Detection) {
	reader, err := os.Open(imagePath)
	if err != nil {
		log.Fatalf("Error in reading the image file: %v", err)
	}

	// Decode the image
	img, _, err := image.Decode(reader)
	if err != nil {
		log.Fatalf("Error in decoding the image: %v", err)
	}

	src := pigo.ImgToNRGBA(img)
	cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y

	dc = gg.NewContext(cols, rows)
	dc.DrawImage(src, 0, 0)

	drawFaces(faces)

	if err := encodeImage(dst); err != nil {
		log.Fatalf("Error encoding the output image: %v", err)
	}
	defer fn.Close()
}

func drawFaces(faces []Detection) {
	for _, face := range faces {
		// Draw the face
		dc.DrawRectangle(float64(face.FaceCoord.Row), float64(face.FaceCoord.Col),
			float64(face.FaceCoord.Width), float64(face.FaceCoord.Height))
		dc.SetLineWidth(4.0)
		dc.SetStrokeStyle(gg.NewSolidPattern(color.RGBA{R: 255, G: 0, B: 0, A: 255}))
		dc.Stroke()

		// Set the radius for drawing out points
		radius := math.Min(10, float64(face.FaceCoord.Width/10))

		// left eye
		dc.DrawPoint(float64(face.LeftEye.Row), float64(face.LeftEye.Col), float64(radius))
		dc.SetLineWidth(4.0)
		dc.SetFillStyle(gg.NewSolidPattern(color.RGBA{R: 255, G: 0, B: 0, A: 255}))
		dc.Fill()

		// right eye
		dc.DrawPoint(float64(face.RightEye.Row), float64(face.RightEye.Col), float64(radius))
		dc.SetLineWidth(4.0)
		dc.SetFillStyle(gg.NewSolidPattern(color.RGBA{R: 255, G: 0, B: 0, A: 255}))
		dc.Fill()

		// nose
		dc.DrawPoint(float64(face.Nose.Row), float64(face.Nose.Col), float64(radius))
		dc.SetLineWidth(4.0)
		dc.SetFillStyle(gg.NewSolidPattern(color.RGBA{R: 255, G: 0, B: 0, A: 255}))
		dc.Fill()

		// mouth
		for _, mouth := range face.Mouth {
			dc.DrawPoint(float64(mouth.Row), float64(mouth.Col), float64(radius))
			dc.SetLineWidth(4.0)
			dc.SetFillStyle(gg.NewSolidPattern(color.RGBA{R: 255, G: 0, B: 0, A: 255}))
			dc.Fill()
		}

	}
}

// RunFaceDetection ....
func RunFaceDetection(imageHash string, imagePath string) ([]Detection, string) {
	// Find the facial landmarks
	result := DetectMTCNN(imagePath)

	// create an output image
	outputImageFile := imageHash + filepath.Ext(imagePath)
	createOutputFile(outputDir + outputImageFile)

	drawImages(imagePath, result)
	return result, outputImageFile
}
