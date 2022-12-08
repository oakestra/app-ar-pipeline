# Demo Pipeline:

![demo.gif](img/demo.gif)
![pipeline](img/pipeline.png)

This is an AR pipeline composed of 3 microservices. 

- Pre: preprocessing microservice, collects the frames and adapts them for the model. 
- Object Detection: detects the bounding boxes inside the image. If Object Recognition is up and running, it forwards the frames there. Otherwise, it sends the bounding boxes back to the client.  
- Object Recognitions: it receives the frames from object detection. For each bounding box of type "Person" it detects the face features and sends them back to the client. 

# How to deploy the pipeline using oakestra

POST the following deployment descriptor using Oakestra `/api/application/` endpoint:

```
{
  "sla_version" : "v2.0",
  "customerID" : "Admin",
  "applications" : [
    {
      "applicationID" : "",
      "application_name" : "pipeline",
      "application_namespace" : "example",
      "application_desc" : "Demo pipeline",
      "microservices" : [
        {
          "microserviceID": "",
          "microservice_name": "Pre",
          "microservice_namespace": "deploy",
          "virtualization": "container",
          "cmd": ["./pre/main", "-port", "5001", "-obj", "10.30.10.10:10501", "-x","200","-y","100"],
          "memory": 50,
          "vcpus": 1,
          "vgpus": 0,
          "vtpus": 0,
          "bandwidth_in": 0,
          "bandwidth_out": 0,
          "storage": 0,
          "code": "docker.io/giobart/demo-pipeline:preprocessing",
          "state": "",
          "port": "5001:5001/tcp",
          "connectivity": [],
          "added_files": []
        },
        {
          "microserviceID": "",
          "microservice_name": "Obj",
          "microservice_namespace": "deploy",
          "virtualization": "container",
          "cmd": ["python3","detection.py","--recognition-address","10.30.10.20:10502","--entrypoint","0.0.0.0:10501","--model","yolox_nano","--max-latency","0.1","--recovery-timeout","1"],
          "memory": 100,
          "vcpus": 1,
          "vgpus": 0,
          "vtpus": 0,
          "bandwidth_in": 0,
          "bandwidth_out": 0,
          "storage": 0,
          "code": "docker.io/giobart/demo-pipeline:detection",
          "state": "",
          "port": "",
          "addresses": {
            "rr_ip": "10.30.10.10"
          },
          "connectivity": [],
          "added_files": []
        },
        {
          "microserviceID": "",
          "microservice_name": "Rec",
          "microservice_namespace": "deploy",
          "virtualization": "container",
          "cmd": ["python3","recognition.py","--entrypoint","0.0.0.0:10502"],
          "memory": 100,
          "vcpus": 1,
          "vgpus": 1,
          "vtpus": 0,
          "bandwidth_in": 0,
          "bandwidth_out": 0,
          "storage": 0,
          "code": "docker.io/giobart/demo-pipeline:recognition",
          "state": "",
          "port": "",
          "addresses": {
            "rr_ip": "10.30.10.20"
          },
          "connectivity": [],
          "added_files": []
        }
      ]
    }
  ]
}

```

## Please note the following:

-----> The currently published images only work on **amd64** architectures! For arm devices please build your own images. 

-----> Recognition is a very heavyweight service. The current image does not exploit cuda capabilities. If your hardware is not bulky enough try first only deploying pre and obj 

# Test the pipeline

To test the pipeline use the client inside the `client` folder.

1. Make sure the machine where Pre is stored and the machine you use to deploy the client are mutually reachable. In particular port `5001/tcp` for the server machine and port `40100/tcp` for the client machine. 
  
2. Make sure you have a working installation of GoLang. Check it out using the command `go version`

3. Make sure you have OpenCV 4.5.5 installed on your machine

4. Install client dependencies with `go get -u`

5. Run the client using: `go run main.go -entry <ip-of-preprocessing-host>:5001`

6. Use `go run main.go -h` for extra client parameters






