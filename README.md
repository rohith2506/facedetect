<h1 align="center">Face Detection as a Service</h1>

Facedetect is a pure Go face detection API which depends on [mtcnn](https://github.com/ipazc/mtcnn).

For more information on pigo, Follow this [paper](https://arxiv.org/pdf/1604.02878.pdf). 

### Demo
<img src="./test_images/me.png" alt="alt text" width="900"/>

Current demo shows json result as 
```
[
    "Faces": [
        {
            "bounds": {
                "y": 385,
                "x": 193,
                "width": 448,
                "height": 619
            },
            "left_eye": {
                "x": 487,
                "y": 445
            },
            ...
        }
         ....
]
```

## Install
```bash
$ git clone https://github.com/rohith2506/facedetect.git
$ docker build -t <your-docker-tag> .
$ docker run -d -p 8000:8000 <your-docker-tag>
```
Open localhost:8000 to access the web UI.

*Note: We store the web images in s3. please make sure to add `AWS_SECRET_ACCESS_KEY` and `AWS_ACCESS_KEY_ID` to Dockerfile before running it*

## Author

* Rohith Uppala
