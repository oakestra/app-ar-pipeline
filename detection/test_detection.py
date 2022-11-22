import pytest
import cv2
import requests_mock
from detection import create_app


@pytest.fixture()
def app():
    app = create_app({
        "model": "yolox_m",
        "failure_threshold": 5,
        "recovery_timeout": 5,
        "max_latency": 0.2,
        "recognition_address": "127.0.0.1",
        "TESTING": True
    })

    yield app


@pytest.fixture()
def client(app):
    return app.test_client()


def test_detect_objects(client):
    with requests_mock.Mocker() as m:
        m.post(f'http://127.0.0.1/api/recognition')

        tmp = cv2.imread("../test_image.jpg")
        tmp = cv2.resize(tmp, (640, 640))
        _, tmp = cv2.imencode(".jpg", tmp)
        response = client.post("/api/detection",
                               data=tmp.tobytes(),
                               headers={"content-type": "image/jpeg", "frame-number": 10, "client-id": "abc", "client-address": "192.168.178.1", "ratio-y": 3.9063, "ratio-y": 2.5453})

        assert response.status_code == 200
        assert response.json['results'] != None
