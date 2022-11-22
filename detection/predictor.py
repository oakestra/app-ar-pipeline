import os
import torch
from yolox.data.data_augment import ValTransform
from yolox.exp import get_exp, Exp
from yolox.utils import postprocess

COCO_CLASSES = (
    "person",
    "bicycle",
    "car",
    "motorcycle",
    "airplane",
    "bus",
    "train",
    "truck",
    "boat",
    "traffic light",
    "fire hydrant",
    "stop sign",
    "parking meter",
    "bench",
    "bird",
    "cat",
    "dog",
    "horse",
    "sheep",
    "cow",
    "elephant",
    "bear",
    "zebra",
    "giraffe",
    "backpack",
    "umbrella",
    "handbag",
    "tie",
    "suitcase",
    "frisbee",
    "skis",
    "snowboard",
    "sports ball",
    "kite",
    "baseball bat",
    "baseball glove",
    "skateboard",
    "surfboard",
    "tennis racket",
    "bottle",
    "wine glass",
    "cup",
    "fork",
    "knife",
    "spoon",
    "bowl",
    "banana",
    "apple",
    "sandwich",
    "orange",
    "broccoli",
    "carrot",
    "hot dog",
    "pizza",
    "donut",
    "cake",
    "chair",
    "couch",
    "potted plant",
    "bed",
    "dining table",
    "toilet",
    "tv",
    "laptop",
    "mouse",
    "remote",
    "keyboard",
    "cell phone",
    "microwave",
    "oven",
    "toaster",
    "sink",
    "refrigerator",
    "book",
    "clock",
    "vase",
    "scissors",
    "teddy bear",
    "hair drier",
    "toothbrush",
)


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
            model_name,
    ):
        self.model_name = model_name
        if torch.cuda.is_available():
            self.device = "cuda"
            print("CUDA DRIVER LOADED")
        else:
            self.device = "cpu"

        device = torch.device(self.device)
        exp: Exp = get_exp(exp_name=model_name)
        yolox_model = exp.get_model()

        with open(f'models/{model_name}.pth', 'rb') as f:
            state_dict = torch.load(f, map_location='cpu')
            if "model" in state_dict:
                state_dict = state_dict["model"]
            yolox_model.load_state_dict(state_dict)
            yolox_model.eval()

        yolox_model.to(device)

        self.model = yolox_model
        self.cls_names = COCO_CLASSES
        self.decoder = None
        self.num_classes = exp.num_classes
        self.confthre = exp.test_conf
        self.nmsthre = exp.nmsthre
        self.test_size = exp.test_size
        self.preproc = ValTransform(legacy=False)

    def inference(self, img):
        if self.device == "cuda":
            with torch.cuda.device(0):
                return self.infer(img)
        else:
            return self.infer(img)

    def infer(self, img):
        img_info = {"id": 0}

        height, width = img.shape[:2]
        img_info["height"] = height
        img_info["width"] = width
        img_info["raw_img"] = img

        ratio = min(self.test_size[0] / img.shape[0],
                    self.test_size[1] / img.shape[1])
        img_info["ratio"] = ratio

        img, _ = self.preproc(img, None, self.test_size)
        img = torch.from_numpy(img).unsqueeze(0)
        img = img.float()
        if self.device == "cuda":
            img = img.cuda()

        with torch.no_grad():
            outputs = self.model(img)
            if self.decoder is not None:
                outputs = self.decoder(outputs, dtype=outputs.type())
            outputs = postprocess(
                outputs, self.num_classes, self.confthre,
                self.nmsthre, class_agnostic=True
            )

        return outputs, img_info

    def calculate_box_dimensions(self, output, img_info, cls_conf):
        ratio = img_info["ratio"]
        img = img_info["raw_img"]
        if output is None:
            return img
        output = output.cpu()

        bboxes = output[:, 0:4]

        # preprocessing: resize
        bboxes /= ratio

        cls = output[:, 6]
        scores = output[:, 4] * output[:, 5]

        detections = list()

        for i in range(len(bboxes)):
            box = bboxes[i]
            cls_id = int(cls[i])
            score = scores[i]
            if score < cls_conf:
                continue
            x0 = int(box[0])
            y0 = int(box[1])
            x1 = int(box[2])
            y1 = int(box[3])

            detections.append({
                "x0": x0,
                "x1": x1,
                "y0": y0,
                "y1": y1,
                "conf": float(score),
                "label": self.cls_names[cls_id]
            })

        return detections
