package mtcnn

import (
	"log"
	"net"

	"github.com/gin-gonic/gin/internal/json"
	"github.com/rohith2506/facedetect/models"
)

// Host and Port constants ...
const (
	connectionType = "tcp"
	Host           = "localhost"
	Port           = "3333"
	MaxBufSize     = 2000 // I don't think it will be more than 2k bytes
)

var conn net.Conn

func createNewConnection() {
	if conn == nil {
		conn, _ = net.Dial(connectionType, Host+":"+Port)
	}
}

// DetectFaces ...
func DetectFaces(imagePath string) []models.Detection {
	createNewConnection()
	if conn == nil {
		log.Fatalf("Connection to python wrapper not established well")
	}

	// Fire the request and get the result in terms of bytes
	conn.Write([]byte(imagePath))
	output := make([]byte, MaxBufSize)
	receivedPayloadSize, _ := conn.Read(output)
	output = output[:receivedPayloadSize]

	var results []map[string]interface{}
	var facialLandMarks []models.Detection

	json.Unmarshal(output, &results)

	for _, result := range results {
		face := result["box"].([]int)
		if len(face) == 0 {
			continue
		}
		landmarks := result["keypoints"].(map[string]interface{})
		leftEye := landmarks["left_eye"].([]int)
		rightEye := landmarks["right_eye"].([]int)
		leftMouth := landmarks["mouth_left"].([]int)
		rightMouth := landmarks["mouth_right"].([]int)
		nose := landmarks["nose"].([]int)

		var (
			faceCoord     models.RectCoord
			leftEyeCoord  models.Coord
			rightEyeCoord models.Coord
			MouthCoords   []models.Coord
			Nose          models.Coord
		)

		if len(face) >= 3 {
			faceCoord = models.RectCoord{
				Row:    face[0],
				Col:    face[1],
				Width:  face[2],
				Height: face[3],
			}
		}

		if len(leftEye) >= 2 {
			leftEyeCoord = models.Coord{
				Row: leftEye[0],
				Col: leftEye[1],
			}
		}

		if len(rightEye) >= 2 {
			rightEyeCoord = models.Coord{
				Row: rightEye[0],
				Col: rightEye[1],
			}
		}

		if len(leftMouth) >= 2 {
			MouthCoords = append(MouthCoords, models.Coord{
				Row: leftMouth[0],
				Col: leftMouth[1],
			})
		}

		if len(rightMouth) >= 2 {
			MouthCoords = append(MouthCoords, models.Coord{
				Row: rightMouth[0],
				Col: rightMouth[1],
			})
		}

		if len(nose) >= 2 {
			Nose = models.Coord{
				Row: nose[0],
				Col: nose[1],
			}
		}

		facialLandMarks = append(facialLandMarks, models.Detection{
			FaceCoord: faceCoord,
			LeftEye:   leftEyeCoord,
			RightEye:  rightEyeCoord,
			Mouth:     MouthCoords,
			Nose:      Nose,
		})

	}

	return facialLandMarks
}
