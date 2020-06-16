package models

import (
	"encoding/json"
	"log"
	"net"
)

// Host and Port constants ...
const (
	connectionType = "tcp"
	Host           = "localhost"
	Port           = "3333"
	MaxBufSize     = 20000 // I don't think it will be more than 2k bytes
)

var conn net.Conn

func createNewConnection() {
	if conn == nil {
		conn, _ = net.Dial(connectionType, Host+":"+Port)
	}
}

func convertInterface(input []interface{}) []int {
	var output []int
	for i := range input {
		output = append(output, int(input[i].(float64)))
	}
	return output
}

// DetectMTCNN ...
func DetectMTCNN(imagePath string) []Detection {
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
	var facialLandMarks []Detection

	json.Unmarshal(output, &results)

	for _, result := range results {
		face := convertInterface(result["box"].([]interface{}))
		landmarks := result["keypoints"].(map[string]interface{})
		leftEye := convertInterface(landmarks["left_eye"].([]interface{}))
		rightEye := convertInterface(landmarks["right_eye"].([]interface{}))
		leftMouth := convertInterface(landmarks["mouth_left"].([]interface{}))
		rightMouth := convertInterface(landmarks["mouth_right"].([]interface{}))
		nose := convertInterface(landmarks["nose"].([]interface{}))
		var (
			faceCoord     RectCoord
			leftEyeCoord  Coord
			rightEyeCoord Coord
			MouthCoords   []Coord
			Nose          Coord
		)

		if len(face) >= 3 {
			faceCoord = RectCoord{
				Row:    face[0],
				Col:    face[1],
				Width:  face[2],
				Height: face[3],
			}
		}

		if len(leftEye) >= 2 {
			leftEyeCoord = Coord{
				Row: leftEye[0],
				Col: leftEye[1],
			}
		}

		if len(rightEye) >= 2 {
			rightEyeCoord = Coord{
				Row: rightEye[0],
				Col: rightEye[1],
			}
		}

		if len(leftMouth) >= 2 {
			MouthCoords = append(MouthCoords, Coord{
				Row: leftMouth[0],
				Col: leftMouth[1],
			})
		}

		if len(rightMouth) >= 2 {
			MouthCoords = append(MouthCoords, Coord{
				Row: rightMouth[0],
				Col: rightMouth[1],
			})
		}

		if len(nose) >= 2 {
			Nose = Coord{
				Row: nose[0],
				Col: nose[1],
			}
		}

		facialLandMarks = append(facialLandMarks, Detection{
			FaceCoord: faceCoord,
			LeftEye:   leftEyeCoord,
			RightEye:  rightEyeCoord,
			Mouth:     MouthCoords,
			Nose:      Nose,
		})
	}

	return facialLandMarks
}
