package models

import (
	pico "github.com/rohith2506/facedetect/models/pico"
)


// Model Constants 
const (
	PicoModel = 1
	MTCNNModel = 2
)

// RunFaceDetection ....
func RunFaceDetection(imageHash string, imagePath string, modelToRun int) *pico.ImageOutput {
	var result *pico.ImageOutput
	if modelToRun == PicoModel {
		result = pico.DetectFaces(imageHash, imagePath)
	}
	return result
}
