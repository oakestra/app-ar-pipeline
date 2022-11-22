from flask import Flask, request
from urllib.parse import urlsplit
import cv2
import numpy as np
import argparse
import requests
import time
from predictor import Predictor
from circuitbreaker import circuit
import json
from threading import Thread


class Compute(Thread):
    def __init__(self, request, config, predictor):
        Thread.__init__(self)
        self.request = request
        self.config = config
        self.predictor = predictor

    def run(self):
        headers = {
            'frame-number': self.request.headers.get("frame-number"),
            'client_id': self.request.headers.get("client-id"),
            'client_address': self.request.headers.get("client-address"),
            'y_scaling': self.request.headers.get("y-scaling"),
            'x_scaling': self.request.headers.get("x-scaling"),
            'pre_processing_started': self.request.headers.get("pre_processing_started"),
            'pre_processing_finished': self.request.headers.get("pre_processing_finished"),
            'detection_processing_started': self.request.headers.get("detection_processing_started"),
            'detection_processing_finished': self.request.headers.get("detection_processing_finished"),
            'results': self.request.headers.get("results")
        }

        self.object_recognition(
            headers,
            self.request.data
        )

    def object_recognition(self, headers, image):

        @circuit(failure_threshold=self.config["failure_threshold"], recovery_timeout=self.config["recovery_timeout"])
        def fallback_client(body):
            body['results'] = self.scale_results(body['results'], float(body['x_scaling']),
                                                    float(body['y_scaling']))
            requests.post(
                f'http://{body["client_address"]}/api/result', data=json.dumps(body),
                headers={"Content-Type": "application/json"})

        headers['rec_processing_started'] = str(round(time.time() * 1000))

        client_id = headers.get("client_id")

        npimage = np.frombuffer(image, np.uint8)
        decodedimage = cv2.imdecode(npimage, cv2.IMREAD_COLOR)

        faces = self.predictor.inference(decodedimage)
        bbs = json.loads(headers.get("results"))

        for face in faces:
            inner = face.bbox.astype(int)
            for bb in bbs:
                if (bb['x0'] < inner[0] and bb['y0'] < inner[2]) and (bb['x1'] > inner[1] and bb['y1'] > inner[3]):
                    bb["sex"] = face.sex
                    bb["age"] = face.age
                    bb["landmarks"] = face.landmark_2d_106

        data = {
            'results': bbs,
            'client_id': client_id,
        }

        headers.update(data)
        headers['rec_processing_finished'] = str(round(time.time() * 1000))

        status_code = 500
        try:
            status_code = fallback_client(headers)
        except Exception as e:
            print(e)
            print("Detection status: " + str(status_code))

    def scale_results(self, results, scalingx, scalingy):
        for bb in results:
            bb['x0'] = int(int(bb['x0']) * scalingx)
            bb['x1'] = int(int(bb['x1']) * scalingx)
            bb['y0'] = int(int(bb['y0']) * scalingy)
            bb['y1'] = int(int(bb['y1']) * scalingy)
            bb['landmarks'][:,0] = bb['landmarks'][:,0] * scalingx
            bb['landmarks'][:,1] = bb['landmarks'][:,1] * scalingy
            bb['landmarks'] = bb['landmarks'].tolist()
        return results

def create_app(config=None):
    app = Flask(__name__)
    app.config.update(config)
    predictor = Predictor(config['model'])

    @app.route("/api/recognition", methods=['POST'])
    def detect_objects_endpoint():
        if "client-id" not in request.headers or "client-address" not in request.headers:
            return {"error": "incorrect header fields"}, 400
        thread_a = Compute(request.__copy__(), config, predictor)
        thread_a.start()
        return {"message": "Scheduled"}, 200

    return app


if __name__ == '__main__':
    ap = argparse.ArgumentParser()
    ap.add_argument("--max-latency", default=20, type=int,
                    help="max latency for requests")
    ap.add_argument("--failure-threshold", default=1, type=int,
                    help="max number of failures for circuit breaker")
    ap.add_argument("--recovery-timeout", default=0.1, type=int,
                    help="first recovery point after open circuit breaker")
    ap.add_argument("--model", default="buffalo_sc", type=str,
                    help="name of pretrained Face Analysis model")
    ap.add_argument("--entrypoint", default="0.0.0.0:5002", type=str,
                    help="http server-entrypoint")

    config = vars(ap.parse_args())
    print(config)
    app = create_app(config)
    split = urlsplit('//' + config["entrypoint"])
    app.run(host=split.hostname, port=split.port)
