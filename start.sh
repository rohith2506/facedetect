#!/bin/bash
set -x

go build -o ./out/facedetect .

nohup redis-server & 
nohup python3 models/server.py &


./out/facedetect