import logging

from flask import Flask, request
from urllib.parse import urlsplit
import cv2
import numpy as np
import argparse
import datetime
import requests
import base64
import time
from predictor import Predictor
from circuitbreaker import circuit
import json
from datetime import datetime
from threading import Thread


class Compute(Thread):
    def __init__(self, request, config,predictor):
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
        }

        self.object_detection(
            headers,
            self.request.data
        )

    def object_detection(self, headers, image):

        @circuit(failure_threshold=self.config["failure_threshold"], recovery_timeout=self.config["recovery_timeout"])
        def fallback_client(headers):
            print(headers)
            headers['results'] = self.scale_results(headers['results'], float(headers['x_scaling']), float(headers['y_scaling']))
            requests.post(
                f'http://{headers["client_address"]}/api/result', data=json.dumps(headers),
                headers={"Content-Type": "application/json"})

        @circuit(failure_threshold=self.config["failure_threshold"], recovery_timeout=self.config["recovery_timeout"])
        def forward_frame(headers, data):
            headers["Content-Type"] = "image/jpg"
            r = requests.post(f'http://{config["recognition_address"]}/api/recognition',
                              headers=headers, timeout=config["max_latency"], data=data)
            return r.status_code

        headers['detection_processing_started'] = str(round(time.time() * 1000))

        client_id = headers["client_id"]
        y_scaling = float(headers["y_scaling"])
        x_scaling = float(headers["x_scaling"])

        npimage = np.frombuffer(image, np.uint8)
        decodedimage = cv2.imdecode(npimage, cv2.IMREAD_COLOR)

        outputs, img_info = self.predictor.inference(decodedimage)
        results = self.predictor.calculate_box_dimensions(
            outputs[0], img_info, 0.35)

        data = {
            'results': str(json.dumps(results)),
            'client_id': client_id,
            'y_scaling': str(y_scaling),
            'x_scaling': str(x_scaling)
        }
        headers.update(data)
        headers['detection_processing_finished'] = str(round(time.time() * 1000))

        status_code = 500
        try:
            status_code = forward_frame(headers, image)
        except Exception as e:
            print("IMPOSSIBLE TO FORWARD")
            print(e)
            status_code = fallback_client(headers)
        print("Detection status: " + str(status_code))

    def scale_results(self, results, scalingx, scalingy):
        res = json.loads(results)
        for bb in res:
            bb['x0'] = int(int(bb['x0']) * scalingx)
            bb['x1'] = int(int(bb['x1']) * scalingx)
            bb['y0'] = int(int(bb['y0']) * scalingy)
            bb['y1'] = int(int(bb['y1']) * scalingy)
        return res


def create_app(config=None):
    app = Flask(__name__)
    app.config.update(config)
    predictor = Predictor(config['model'])

    @app.route("/api/detection", methods=['POST'])
    def detect_objects_endpoint():
        if "client-id" not in request.headers or "client-address" not in request.headers:
            return {"error": "incorrect header fields"}, 400
        thread_a = Compute(request.__copy__(),config,predictor)
        thread_a.start()
        return {"message": "Scheduled"}, 200

    return app


if __name__ == '__main__':
    ap = argparse.ArgumentParser()
    ap.add_argument("--max-latency", default=400, type=float,
                    help="max latency for requests")
    ap.add_argument("--failure-threshold", default=3, type=int,
                    help="max number of failures for circuit breaker")
    ap.add_argument("--recovery-timeout", default=5, type=int,
                    help="first recovery point after open circuit breaker")
    ap.add_argument("--recognition-address", default="0.0.0.0:5002", type=str,
                    help="address of recognition service")
    ap.add_argument("--model", default="yolox_nano", type=str,
                    help="name of pretrained YOLOX model")
    ap.add_argument("--entrypoint", default="0.0.0.0:5001", type=str,
                    help="http server-entrypoint")

    config = vars(ap.parse_args())
    print(config)
    app = create_app(config)
    split = urlsplit('//' + config["entrypoint"])
    app.run(host=split.hostname, port=split.port)
