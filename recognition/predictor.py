import imp
import cv2
import numpy as np
from insightface.app import FaceAnalysis


def singleton(class_):
    instances = {}

    def getinstance(*args, **kwargs):
        if class_ not in instances:
            instances[class_] = class_(*args, **kwargs)
        return instances[class_]

    return getinstance


@singleton
class Predictor(object):
    def __init__(
            self,
            model_name
    ):
        self.model = FaceAnalysis(name=model_name)
        self.model.prepare(ctx_id=0, det_size=(640, 640))

    def inference(self, img):
        return self.model.get(img)


if __name__ == '__main__':
    #warmup
    model = FaceAnalysis(name="buffalo_s", providers=['CPUExecutionProvider'])
