import socket
from mtcnn import MTCNN
from cv2 import cv2
import pdb

HOST = '127.0.0.1'
PORT = 3333

print("Creating a simple python server.....")
with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
    s.bind((HOST, PORT))
    s.listen()
    conn, addr = s.accept()
    with conn:
        while True:
            image_path = conn.recv(1024)
            if not image_path: break
            image_path = image_path.decode('utf-8').strip()
            img = cv2.cvtColor(cv2.imread(image_path), cv2.COLOR_BGR2RGB)
            detector = MTCNN()
            result = detector.detect_faces(img)
            result = str(result).encode('utf-8')
            conn.sendall(result)