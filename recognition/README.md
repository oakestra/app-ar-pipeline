# Detection

Detection service for the demo AR pipeline.
It supports differently sized YOLOX networks.

## Setup
Install required dependencies:

`pip install -r requirements.txt`

Start flask webserver:

`flask run` or `python detection.py`

Available Config Flags:
- `recognition-address`: address of recognition service for frame forwarding
- `model`: specific YOLOx model size
- `entrypoint`: server entrypoint
- `max-latency`: maximum request latency
- `failure-threshold`: circuit breaker threshold
- `recovery-timeout`: time before next attempt at open circuit

## API Routes
`"/api/detection"`:
Expects an image as request body (with the appropriate content type header).
Following meta-data headers should be included:
- frame-number
- client-id
- client-address
- original-width
- original-height
