package models

import (
	"errors"
	"fmt"
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
	"github.com/rohith2506/facedetect/models/mtcnn"
	pico "github.com/rohith2506/facedetect/models/pico"
)

var (
	dc               *gg.Context
	cascade          []byte
	puplocCascade    []byte
	faceClassifier   *pigo.Pigo
	puplocClassifier *pigo.PuplocCascade
	flpcs            map[string][]*pigo.FlpCascade
	imgParams        *pigo.ImageParams
	err              error
	dst              io.Writer
	fn               *os.File
)

const (
	cascadeFaceFinder = "cascade/facefinder"
	cascadePuploc     = "cascade/puploc"
	cascadeLPS        = "cascade/lps"
	defaultAngle      = 0.0
	iouThreshold      = 0.2
	perturbVal        = 63
	qThreshold        = 5.0
	minImageSize      = 20
	maxImageSize      = 4000
	adjustedRows      = 400
	adjustedCols      = 300
)

var (
	mouthCascade         = []string{"lp93", "lp84", "lp82", "lp81"}
	qThresh      float32 = qThreshold
	perturb              = perturbVal
)

// Model Constants
const (
	PicoModel  = 1
	MTCNNModel = 2
)

// Image Output
type Coord struct {
	Row int `json:"x,omitempty"`
	Col int `json:"y,omitempty"`
}

type RectCoord struct {
	Row    int `json:"y,omitempty"`
	Col    int `json:"x,omitempty"`
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
}

type Detection struct {
	FaceCoord RectCoord `json:"bounds,omitempty"`
	LeftEye   Coord     `json:"left_eye,omitempty"`
	RightEye  Coord     `json:"right_eye,omitempty"`
	Mouth     []Coord   `json:"mouth,omitempty"`
	Nose      Coord     `json:"nose,omitempty"`
}

type ImageOutput struct {
	Landmarks []Detection `json:"result,omitempty"`
	ImagePath string      `json:"image_path,omitempty"`
}

func drawFaces(faces []pigo.Detection) ([]Detection, error) {
	var (
		qThresh float32 = qThreshold
		perturb         = perturbVal
	)

	var (
		detections []Detection
		puploc     *pigo.Puploc
	)

	for _, face := range faces {
		if face.Q <= qThresh {
			continue
		}
		var (
			leftEyeCoord  coord
			rightEyeCoord coord
			mouthCoords   []coord
		)

		// We are actually drawing a square
		dc.DrawRectangle(
			float64(face.Col-face.Scale/2),
			float64(face.Row-face.Scale/2),
			float64(face.Scale),
			float64(face.Scale),
		)
		faceCoord := &rectCoord{
			Row:    face.Row,
			Col:    face.Col,
			Length: face.Scale,
		}

		dc.SetLineWidth(2.0)
		dc.SetStrokeStyle(gg.NewSolidPattern(color.RGBA{R: 255, G: 0, B: 0, A: 255}))
		dc.Stroke()

		if face.Scale > 50 {
			// left eye
			puploc = &pigo.Puploc{
				Row:      face.Row - int(0.075*float32(face.Scale)),
				Col:      face.Col - int(0.175*float32(face.Scale)),
				Scale:    float32(face.Scale) * 0.25,
				Perturbs: perturb,
			}
			leftEye := puplocClassifier.RunDetector(*puploc, *imgParams, 0.0, false)
			if leftEye.Row > 0 && leftEye.Col > 0 {
				drawDetections(dc,
					float64(leftEye.Col),
					float64(leftEye.Row),
					float64(leftEye.Scale),
					color.RGBA{R: 255, G: 0, B: 0, A: 255},
					true,
				)
				leftEyeCoord = coord{
					Row:   leftEye.Row,
					Col:   leftEye.Col,
					Scale: int(leftEye.Scale),
				}
			}

			// right eye
			puploc = &pigo.Puploc{
				Row:      face.Row - int(0.075*float32(face.Scale)),
				Col:      face.Col + int(0.185*float32(face.Scale)),
				Scale:    float32(face.Scale) * 0.25,
				Perturbs: perturb,
			}
			rightEye := puplocClassifier.RunDetector(*puploc, *imgParams, 0.0, false)
			if rightEye.Row > 0 && rightEye.Col > 0 {
				drawDetections(dc,
					float64(rightEye.Col),
					float64(rightEye.Row),
					float64(rightEye.Scale),
					color.RGBA{R: 255, G: 0, B: 0, A: 255},
					true,
				)
				rightEyeCoord = coord{
					Row:   rightEye.Row,
					Col:   rightEye.Col,
					Scale: int(rightEye.Scale),
				}
			}

			// mouth
			for _, mouth := range mouthCascade {
				for _, flpc := range flpcs[mouth] {
					flp := flpc.FindLandmarkPoints(leftEye, rightEye, *imgParams, perturb, false)
					if flp.Row > 0 && flp.Col > 0 {
						drawDetections(dc,
							float64(flp.Col),
							float64(flp.Row),
							float64(flp.Scale*0.5),
							color.RGBA{R: 0, G: 0, B: 255, A: 255},
							false,
						)
					}
					mouthCoords = append(mouthCoords, coord{
						Row:   flp.Row,
						Col:   flp.Col,
						Scale: int(flp.Scale),
					})
				}
			}
		}

		detections = append(detections, Detection{
			FacePoints: *faceCoord,
			LeftEye:    leftEyeCoord,
			RightEye:   rightEyeCoord,
			Mouth:      mouthCoords,
		})
	}
	return detections, nil
}

// drawDetections is a helper function to draw the detection marks
func drawDetections(ctx *gg.Context, x, y, r float64, c color.RGBA, markDet bool) {
	ctx.DrawArc(x, y, r*0.15, 0, 2*math.Pi)
	ctx.SetFillStyle(gg.NewSolidPattern(c))
	ctx.Fill()

	if markDet {
		ctx.DrawRectangle(x-(r*1.5), y-(r*1.5), r*3, r*3)
		ctx.SetLineWidth(2.0)
		ctx.SetStrokeStyle(gg.NewSolidPattern(color.RGBA{R: 255, G: 255, B: 0, A: 255}))
		ctx.Stroke()
	}
}

// create output file
func createOutputFile(imagePath string) {
	fmt.Println("output file name: ", imagePath)
	fn, err = os.OpenFile(outputDir+imagePath, os.O_CREATE|os.O_WRONLY, 0755)
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

// RunFaceDetection ....
func RunFaceDetection(imageHash string, imagePath string, modelToRun int) *ImageOutput {
	// Find the facial landmarks
	var result []Detection
	if modelToRun == PicoModel {
		result = pico.DetectFaces(imagePath)
	} else {
		result = mtcnn.DetectFaces(imagePath)
	}

	// Create the temporary image ( use go routine to do it asynchronously )
	if err := encodeImage(dst); err != nil {
		log.Fatalf("Error encoding the output image: %v", err)
	}
	defer fn.Close()
	return result
}
