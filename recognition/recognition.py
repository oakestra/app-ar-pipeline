import base64

import cv2
import numpy as np
import argparse
import time
import concurrent.futures
import traceback
import grpc
from predictor import Predictor
import json

import queueService_pb2_grpc

inference_predictor = None


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


def object_recognition(image):
    image['rec_processing_started'] = str(round(time.time() * 1000))

    npimage = np.frombuffer(base64.b64decode(image["frame"]), np.uint8)
    decodedimage = cv2.imdecode(npimage, cv2.IMREAD_COLOR)

    faces = inference_predictor.inference(decodedimage)
    bbs = image["results"]
    for face in faces:
        inner = face.bbox.astype(int)
        for bb in bbs:
            if (bb['x0'] < inner[0] and bb['y0'] < inner[2]) and (bb['x1'] > inner[1] and bb['y1'] > inner[3]):
                bb["sex"] = face.sex
                bb["age"] = face.age
                if face.landmark_2d_106 is not None:
                    bb["landmarks"] = face.landmark_2d_106
                else:
                    bb["landmarks"] = face.kps

    image["frame"]=""
    image['results'] = scale_results(bbs, image["x_scaling"], image["y_scaling"])
    image['landmarks']=1
    image['rec_processing_finished'] = str(round(time.time() * 1000))

    return image


def scale_results(results, scalingx, scalingy):
    for bb in results:
        if bb.get('landmarks') is not None:
            bb['landmarks'][:, 0] = bb['landmarks'][:, 0] * float(scalingx)
            bb['landmarks'][:, 1] = bb['landmarks'][:, 1] * float(scalingy)
            bb['landmarks'] = bb['landmarks'].tolist()
    return results


class StreamingService(queueService_pb2_grpc.QueueServiceServicer):
    def NextFrame(self, request, context):
        try:
            frame = json.loads(request.data)
            request.data = json.dumps(object_recognition(frame)).encode("utf-8")
            return request
        except Exception as e:
            traceback.print_exc()
        return request


if __name__ == '__main__':
    ap = argparse.ArgumentParser()
    ap.add_argument("--model", default="buffalo_s", type=str,
                    help="name of pretrained Face Analysis model")
    ap.add_argument("--entrypoint", default="0.0.0.0:4042", type=str,
                    help="http server-entrypoint")

    config = vars(ap.parse_args())
    inference_predictor = Predictor(config['model'])
    print(config)
    server = grpc.server(concurrent.futures.ThreadPoolExecutor(max_workers=10))
    queueService_pb2_grpc.add_QueueServiceServicer_to_server(
        StreamingService(), server)
    server.add_insecure_port(config["entrypoint"])
    server.start()
    server.wait_for_termination()
