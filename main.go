package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"image/jpeg"
	image "image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"

	pigo "github.com/esimov/pigo/core"
	"github.com/fogleman/gg"
	//    "reflect"
)

type coord struct {
	Row   int `json:"x,omitempty"`
	Col   int `json:"y,omitempty"`
	Scale int `json:"size,omitempty"`
}

type rectCoord struct {
	Row    int `json:"y,omitempty"`
	Col    int `json:"x,omitempty"`
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty`
}

// detection holds the detection points of the various detection types
type detection struct {
	FacePoints     rectCoord `json:"face,omitempty"`
	EyePoints      []coord   `json:"eyes,omitempty"`
	LandmarkPoints []coord   `json:"landmark_points,omitempty"`
}

var (
	dc               *gg.Context
	cascade          []byte
	puplocCascade    []byte
	faceClassifier   *pigo.Pigo
	puplocClassifier *pigo.PuplocCascade
	flpcs            map[string][]*pigo.FlpCascade
	imgParams        *pigo.ImageParams
	err              error
)

var (
	eyeCascades  = []string{"lp46", "lp44", "lp42", "lp38", "lp312"}
	mouthCascade = []string{"lp93", "lp84", "lp82", "lp81"}
)

func main() {
	var userInput string
	fmt.Scanln(&userInput)

	reader, err := os.Open(userInput)
	if err != nil {
		log.Fatalf("Error in reading the image gile: %v", err)
	}

	img, err := image.Decode(reader)
	if err != nil {
		log.Fatalf("Error in decoding the image: %v", err)
	}

	var dst io.Writer
	fn, err := os.OpenFile("out.jpg", os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Fatalf("Unable to open output file: %v", err)
	}
	defer fn.Close()
	dst = fn

	src := pigo.ImgToNRGBA(img)

	pixels := pigo.RgbToGrayscale(src)
	cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y

	dc = gg.NewContext(cols, rows)
	dc.DrawImage(src, 0, 0)

	faces := findFaces(pixels, rows, cols)

	dets, err := drawFaces(faces)
	if err != nil {
		log.Fatalf("Error in drawing faces: %v", err)
	}
	bytes, _ := json.Marshal(dets)
	fmt.Println("json output: ", string(bytes))

	if err := encodeImage(dst); err != nil {
		log.Fatalf("Error encoding the output image: %v", err)
	}
}

func drawFaces(faces []pigo.Detection) ([]detection, error) {
	var (
		qThresh float32 = 5.0
		perturb         = 63
	)

	var (
		detections     []detection
		eyesCoords     []coord
		landmarkCoords []coord
		puploc         *pigo.Puploc
	)

	for _, face := range faces {
		if face.Q > qThresh {
			dc.DrawRectangle(
				float64(face.Col-face.Scale/2),
				float64(face.Row-face.Scale/2),
				float64(face.Scale),
				float64(face.Scale),
			)
			faceCoord := &rectCoord{
				Row:    face.Row,
				Col:    face.Col,
				Width:  face.Scale,
				Height: face.Scale,
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
					eyesCoords = append(eyesCoords, coord{
						Row:   leftEye.Row,
						Col:   leftEye.Col,
						Scale: int(leftEye.Scale),
					})
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
				}
				eyesCoords = append(eyesCoords, coord{
					Row:   rightEye.Row,
					Col:   rightEye.Col,
					Scale: int(rightEye.Scale),
				})

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
						landmarkCoords = append(landmarkCoords, coord{
							Row:   flp.Row,
							Col:   flp.Col,
							Scale: int(flp.Scale),
						})
					}
				}
			}
			detections = append(detections, detection{
				FacePoints:     *faceCoord,
				EyePoints:      eyesCoords,
				LandmarkPoints: landmarkCoords,
			})
		}
	}
	return detections, nil
}

func encodeImage(dst io.Writer) error {
	var err error
	img := dc.Image()

	switch dst.(type) {
	case *os.File:
		ext := filepath.Ext(dst.(*os.File).Name())
		switch ext {
		case "", ".jpg", ".jpeg":
			err = jpeg.Encode(dst, img, &jpeg.Options{Quality: 100})
		case ".png":
			err = png.Encode(dst, img)
		default:
			err = errors.New("unsupported image format")
		}
	default:
		err = jpeg.Encode(dst, img, &jpeg.Options{Quality: 100})
	}
	return err
}

// clusterDetection runs Pigo face detector core methods
// and returns a cluster with the detected faces coordinates.
func findFaces(pixels []uint8, rows, cols int) []pigo.Detection {
	imgParams = &pigo.ImageParams{
		Pixels: pixels,
		Rows:   rows,
		Cols:   cols,
		Dim:    cols,
	}
	cParams := pigo.CascadeParams{
		MinSize:     20,
		MaxSize:     4000,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
		ImageParams: *imgParams,
	}

	// Ensure that the face detection classifier is loaded only once.
	if len(cascade) == 0 {
		cascade, err = ioutil.ReadFile("/Users/ruppala/go/src/github.com/esimov/pigo/cascade/facefinder")
		if err != nil {
			log.Fatalf("Error reading the cascade file: %v", err)
		}
		p := pigo.NewPigo()

		// Unpack the binary file. This will return the number of cascade trees,
		// the tree depth, the threshold and the prediction from tree's leaf nodes.
		faceClassifier, err = p.Unpack(cascade)
		if err != nil {
			log.Fatalf("Error unpacking the cascade file: %s", err)
		}
	}

	// Ensure that we load the pupil localization cascade only once
	if len(puplocCascade) == 0 {
		puplocCascade, err := ioutil.ReadFile("/Users/ruppala/go/src/github.com/esimov/pigo/cascade/puploc")
		if err != nil {
			log.Fatalf("Error reading the puploc cascade file: %s", err)
		}
		puplocClassifier, err = puplocClassifier.UnpackCascade(puplocCascade)
		if err != nil {
			log.Fatalf("Error unpacking the puploc cascade file: %s", err)
		}

		flpcs, err = puplocClassifier.ReadCascadeDir("/Users/ruppala/go/src/github.com/esimov/pigo/cascade/lps")
		if err != nil {
			log.Fatalf("Error unpacking the facial landmark detection cascades: %s", err)
		}
	}

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := faceClassifier.RunCascade(cParams, 0.0)

	// Calculate the intersection over union (IoU) of two clusters.
	dets = faceClassifier.ClusterDetections(dets, 0.2)

	return dets
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
