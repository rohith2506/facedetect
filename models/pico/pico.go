package pico

import (
	"image"
	"io/ioutil"
	"log"
	"os"

	pigo "github.com/esimov/pigo/core"
	models "github.com/rohith2506/facedetect/models"
)

var (
	cascade          []byte
	puplocCascade    []byte
	faceClassifier   *pigo.Pigo
	puplocClassifier *pigo.PuplocCascade
	flpcs            map[string][]*pigo.FlpCascade
	err              error
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
	maxImageSize      = 2000
)

var (
	mouthCascade         = []string{"lp93", "lp84", "lp82", "lp81"}
	qThresh      float32 = qThreshold
	perturb              = perturbVal
)

// DetectFaces ....
func DetectFaces(imagePath string) []models.Detection {
	// Decode the image
	reader, err := os.Open(imagePath)
	if err != nil {
		log.Fatalf("Error in reading the image file: %v", err)
	}
	img, _, err := image.Decode(reader)
	if err != nil {
		log.Fatalf("Error in decoding the image: %v", err)
	}
	src := pigo.ImgToNRGBA(img)
	pixels := pigo.RgbToGrayscale(src)
	cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y

	// Find all landmarks ( faces, eyes and mouth )
	facialLandmarks := findFacialLandMarks(pixels, rows, cols)

	return facialLandmarks
}

// load pico models ...
func loadPicoModels() {
	if len(cascade) == 0 {
		cascade, err = ioutil.ReadFile(cascadeFaceFinder)
		if err != nil {
			log.Fatalf("Error reading the cascade file: %v", err)
		}
		p := pigo.NewPigo()
		faceClassifier, err = p.Unpack(cascade)
		if err != nil {
			log.Fatalf("Error unpacking the cascade file: %s", err)
		}
	}

	if len(puplocCascade) == 0 {
		puplocCascade, err := ioutil.ReadFile(cascadePuploc)
		if err != nil {
			log.Fatalf("Error reading the puploc cascade file: %s", err)
		}
		puplocClassifier, err = puplocClassifier.UnpackCascade(puplocCascade)
		if err != nil {
			log.Fatalf("Error unpacking the puploc cascade file: %s", err)
		}
		flpcs, err = puplocClassifier.ReadCascadeDir(cascadeLPS)
		if err != nil {
			log.Fatalf("Error unpacking the facial landmark detection cascades: %s", err)
		}
	}
}

func findFacialLandMarks(pixels []uint8, rows, cols int) []models.Detection {
	// Initialize the parameters
	imgParams := &pigo.ImageParams{
		Pixels: pixels,
		Rows:   rows,
		Cols:   cols,
		Dim:    cols,
	}
	cParams := pigo.CascadeParams{
		MinSize:     minImageSize,
		MaxSize:     maxImageSize,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
		ImageParams: *imgParams,
	}

	// Load pico models
	loadPicoModels()

	// Find the faces
	dets := faceClassifier.RunCascade(cParams, defaultAngle)
	faces := faceClassifier.ClusterDetections(dets, iouThreshold)

	var facialLandmarks []models.Detection

	// Find the remaining landmarks
	for _, face := range faces {
		if face.Q <= qThresh || face.Scale < 50 {
			continue
		}
		var (
			puploc        *pigo.Puploc
			leftEyeCoord  models.Coord
			rightEyeCoord models.Coord
			mouthCoords   []models.Coord
		)

		// face
		faceCoord := models.RectCoord{
			Row:    face.Row,
			Col:    face.Col,
			Width:  face.Scale,
			Height: face.Scale,
		}

		// left eye
		puploc = &pigo.Puploc{
			Row:      face.Row - int(0.075*float32(face.Scale)),
			Col:      face.Col - int(0.175*float32(face.Scale)),
			Scale:    float32(face.Scale) * 0.25,
			Perturbs: perturb,
		}
		leftEye := puplocClassifier.RunDetector(*puploc, *imgParams, 0.0, false)
		if leftEye.Row > 0 && leftEye.Col > 0 {
			leftEyeCoord = models.Coord{
				Row: leftEye.Row,
				Col: leftEye.Col,
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
			rightEyeCoord = models.Coord{
				Row: rightEye.Row,
				Col: rightEye.Col,
			}
		}

		// mouth
		for _, mouth := range mouthCascade {
			for _, flpc := range flpcs[mouth] {
				flp := flpc.FindLandmarkPoints(leftEye, rightEye, *imgParams, perturb, false)
				if flp.Row > 0 && flp.Col > 0 {
				}
				mouthCoords = append(mouthCoords, models.Coord{
					Row: flp.Row,
					Col: flp.Col,
				})
			}
		}

		// Append everything to the output
		facialLandmarks = append(facialLandmarks, models.Detection{
			FaceCoord: faceCoord,
			LeftEye:   leftEyeCoord,
			RightEye:  rightEyeCoord,
			Mouth:     mouthCoords,
		})
	}

	return facialLandmarks
}
