package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sigcommdemo/pre/taskQueue"
	"time"
)

var frameTaskQueue taskQueue.FrameTaskQueue
var port *int
var sizeX *int
var sizeY *int
var objAddr *string

func main() {
	port = flag.Int("port", 4040, "entrypoint api port")
	sizeX = flag.Int("x", 600, "resize size for X axis")
	sizeY = flag.Int("y", 600, "resize size for Y axis")
	objAddr = flag.String("obj", "10.30.30.31:4041", "address of object detection")
	flag.Parse()
	frameTaskQueue = taskQueue.GetTaskQueue()
	go frameProcessor()
	handleRequests()
}

func handleRequests() {
	preRouter := mux.NewRouter().StrictSlash(true)
	preRouter.HandleFunc("/api/entrypoint", newIncomingFrame).Methods("POST")
	fmt.Printf("Started listending at port %d \n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), preRouter))
}

//Endpoint /api/entrypoint
func newIncomingFrame(w http.ResponseWriter, r *http.Request) {
	clientId := r.Header.Get("client-id")
	frameId := r.Header.Get("frame-number")
	clientPort := r.Header.Get("client-result-port")
	clientFrameImage, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
	}

	clientip, _, _ := net.SplitHostPort(r.RemoteAddr)
	frameTaskQueue.Push(&taskQueue.Frame{
		Image:         clientFrameImage,
		ClientId:      clientId,
		FrameId:       frameId,
		ClientAddress: fmt.Sprintf("%s:%s", clientip, clientPort),
		ArrivalTime:   fmt.Sprintf("%d", makeTimestamp()),
	})

	_, _ = w.Write(nil)
	if err != nil {
		return
	}
}

//When a new frame is available, it resizes it and sends to the ObjectDetector
func frameProcessor() {
	notificationChannel := frameTaskQueue.GetNotificationChannel()
	for true {
		select {
		case <-notificationChannel:
			frame := frameTaskQueue.Pop()
			if frame != nil {
				resized, err := resizeFrame(*frame)
				if err == nil {
					sendRequestToObjectDetector(resized, frame.ArrivalTime)
				}
			}
		}
	}
}

func sendRequestToObjectDetector(frame taskQueue.Frame, started string) {
	client := http.Client{
		Timeout: 150 * time.Millisecond,
	}
	url := fmt.Sprintf("http://%s/api/detection", *objAddr)
	req, err := http.NewRequest("POST", url, bytes.NewReader(frame.Image))
	if err != nil {
		fmt.Printf("ERROR %v \n", err)
		return
	}
	req.Header.Set("Content-Type", "image/jpg")
	req.Header.Set("client-id", frame.ClientId)
	req.Header.Set("frame-number", frame.FrameId)
	req.Header.Set("client-address", frame.ClientAddress)
	req.Header.Set("y_scaling", fmt.Sprintf("%f", float64(frame.OriginalH)/float64(*sizeY)))
	req.Header.Set("x_scaling", fmt.Sprintf("%f", float64(frame.OriginalW)/float64(*sizeX)))
	req.Header.Set("pre_processing_started", started)
	req.Header.Set("pre_processing_finished", fmt.Sprintf("%d", makeTimestamp()))
	do, err := client.Do(req)
	if err != nil {
		fmt.Printf("ERROR %v \n", err)
		return
	}
	if do.StatusCode != 200 {
		fmt.Printf("ERROR wrong response status code, %d \n", do.StatusCode)
	}
}

func resizeFrame(frame taskQueue.Frame) (taskQueue.Frame, error) {
	fullSizeImage, _, err := image.Decode(bytes.NewReader(frame.Image))
	shrankImage := resize.Resize(uint(*sizeX), uint(*sizeY), fullSizeImage, resize.NearestNeighbor)
	//graySharnkImage := image.NewGray(shrankImage.Bounds())
	frame.Image = make([]byte, int(*sizeX)*int(*sizeY))
	imageBuffer := new(bytes.Buffer)
	err = jpeg.Encode(imageBuffer, shrankImage, nil)
	if err != nil {
		fmt.Printf("ERROR, impossible to encode frame \n")
		return frame, err
	}
	frame.Image = imageBuffer.Bytes()
	frame.OriginalH = fullSizeImage.Bounds().Size().Y
	frame.OriginalW = fullSizeImage.Bounds().Size().X
	fmt.Printf("resized from %dx%d to-> %dx%d\n", frame.OriginalH, frame.OriginalW, shrankImage.Bounds().Size().X, shrankImage.Bounds().Size().Y)
	return frame, nil
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
