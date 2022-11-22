import pytest
import cv2
import requests_mock
from recognition import create_app


@pytest.fixture()
def app():
    app = create_app({
        "model": "buffalo_sc",
        "root": "demo-pipeline/recognition",
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
        response = client.post("/api/recognition",
                               data=tmp.tobytes(),
                               headers={"content-type": "image/jpeg", "frame-number": 10, "client-id": "abc", "client-address": "192.168.178.1", "y_scaling": 1.6875, "x_scaling": 3, 'results': """[{"x0": 615, "x1": 727, "y0": 261.336875, "y1": 842.07, "conf": 0.9182909727096558, "label": "person"}]"""})


        assert response.status_code == 200
