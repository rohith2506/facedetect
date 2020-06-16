import select
import socket
import sys
import pdb
import json
from cv2 import cv2
from mtcnn import MTCNN
from queue import Queue

HOST = '127.0.0.1'
PORT = 3333

class MultiClientServer:
    def __init__(self):
        self.server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.server.setblocking(0)
        self.server_address = (HOST, PORT)
        self.detector = MTCNN()
        self.inputs, self.outputs, self.message_queues = [], [], {}

    def process_image(self, image_path):
        image_path = image_path.decode('utf-8').strip()
        result = ""
        if not image_path:
            result = "EMPTY_IMAGE_PATH"
        else:
            try:
                img = cv2.cvtColor(cv2.imread(image_path), cv2.COLOR_BGR2RGB)
                result = self.detector.detect_faces(img)
                result = json.dumps(result)
            except Exception as e:
                print("Error in running mtcnn: ", e)
                result = "ERROR"
        return result

    def connect(self):
        self.server.bind(self.server_address)
        self.server.listen(5)
        self.inputs.append(self.server)
        while self.inputs:
            readable, writable, exceptional = select.select(self.inputs, self.outputs, self.inputs)
            for s in readable:
                if s is self.server:
                    connection, client_addr = s.accept()
                    connection.setblocking(0)
                    self.inputs.append(connection)
                    self.message_queues[connection] = Queue()
                else:
                    data = s.recv(1024)
                    if data:
                        result = self.process_image(data)
                        result = result.encode("utf-8")
                        self.message_queues[s].put(result)
                        if s not in self.outputs:
                            self.outputs.append(s)
                    else:
                        if s in self.outputs:
                            self.outputs.remove(s)
                        self.inputs.remove(s)
                        s.close()
                        del self.message_queues[s]
            for s in writable:
                try:
                    next_message = self.message_queues[s].get_nowait()
                except Exception:
                    self.outputs.remove(s)
                else:
                    s.send(next_message)
            for s in exceptional:
                self.inputs.remove(s)
                if s in self.outputs:
                    self.outputs.remove(s)
                s.close()
                del self.message_queues[s]

if __name__ == "__main__":
    server = MultiClientServer()
    server.connect()
