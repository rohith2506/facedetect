FROM golang:latest

LABEL maintainer="Rohith Uppala <rohith.uppala369@gmail.com>"

ENV GIN_MODE=release
ENV AWS_SECRET_ACCESS_KEY=<YOUR AWS SECRET ACCESS KEY>
ENV AWS_ACCESS_KEY_ID=<YOUR AWS ACCESS KEY ID>

RUN mkdir -p /tmp/images/out
RUN mkdir -p /app/facedetect

RUN apt-get update && apt-get install -y python3.7 python3-pip redis libsm6 libxext6 libxrender1
RUN pip3 install --upgrade pip
RUN pip3 install tensorflow==2.2.0
RUN pip3 install mtcnn

WORKDIR /app/facedetect

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

EXPOSE 8000

RUN chmod +x /app/facedetect/start.sh
CMD ["/bin/bash", "-c", "./start.sh"]
