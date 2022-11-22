package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"log"
	"math/rand"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

var fps *int
var bufferSize *int
var bbps *int
var port *int
var showlatency *bool
var primaryentrypoint *string
var secondaryentrypoint *string
var buffer []*gocv.Mat
var bbrequestchan chan framestruct
var randSeed = rand.New(rand.NewSource(time.Now().UnixNano()))
var clientid = randSeed.Intn(999999)

var lastBB BBresponse
var bbwritelock sync.Mutex
var cloudoredge string

type framestruct struct {
	framenumber    int64
	framebufferpos int
}

type BBresponse struct {
	Frameid        string `json:"frame-number"`
	BBs            []BB   `json:"results"`
	PreStart       string `json:"pre_processing_started"`
	PreEnd         string `json:"pre_processing_finished"`
	ObjStart       string `json:"detection_processing_started"`
	ObjEnd         string `json:"detection_processing_finished"`
	RecStart       string `json:"rec_processing_started"`
	RecEnd         string `json:"rec_processing_finished"`
	Latency        int64
	DurationLeft   int
	ResponseServer string
}

type BB struct {
	X1         int         `json:"x0"`
	X2         int         `json:"x1"`
	Y1         int         `json:"y0"`
	Y2         int         `json:"y1"`
	Confidence float64     `json:"conf"`
	Label      string      `json:"label"`
	Sex        string      `json:"sex"`
	Age        int         `json:"age"`
	Landmark   [][]float64 `json:"landmarks"`
}

var WHITE = color.RGBA{
	R: uint8(255),
	G: uint8(255),
	B: uint8(255),
	A: uint8(255),
}
var RED = color.RGBA{
	R: uint8(255),
	G: uint8(0),
	B: uint8(0),
	A: uint8(255),
}
var YELLOW = color.RGBA{
	R: uint8(0),
	G: uint8(255),
	B: uint8(255),
	A: uint8(255),
}
var GREEN = color.RGBA{
	R: uint8(0),
	G: uint8(255),
	B: uint8(0),
	A: uint8(255),
}

var BLUE = color.RGBA{
	R: uint8(0),
	G: uint8(0),
	B: uint8(255),
	A: uint8(255),
}

func main() {
	fps = flag.Int("fps", 30, "set the frames per second captured by the came")
	bufferSize = flag.Int("buffer", 1, "frame buffer size")
	bbps = flag.Int("bbps", 10, "bounding boxes per second to ber requested")
	port = flag.Int("port", 40100, "port exposed to get the answer from the pipeline")
	primaryentrypoint = flag.String("entry", "0.0.0.0:5000", "entrypoint for the pipeline")
	secondaryentrypoint = flag.String("backupentry", "0.0.0.0:5000", "backup cloud entrypoint for the pipeline")
	showlatency = flag.Bool("latency", false, "show the latency")

	flag.Parse()
	fmt.Printf("Started client with id %d - %d fps, %d framebuffer size, %d BB per second \n", clientid, *fps, *bufferSize, *bbps)
	fmt.Printf("Display server exposed on port %d \n", *port)

	bbwritelock = sync.Mutex{}
	buffer = make([]*gocv.Mat, *bufferSize)
	bbrequestchan = make(chan framestruct, 0)
	for i := 0; i < *bufferSize; i++ {
		matrix := gocv.NewMat()
		buffer[i] = &matrix
	}

	window := gocv.NewWindow("Client")
	go frameCapture()
	go sendFrame()
	go handleRequests()
	frameShow(window)
}

// endpoint /api/result
func newBB(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var bb BBresponse
	err := decoder.Decode(&bb)
	if err == nil {
		bbwritelock.Lock()
		defer bbwritelock.Unlock()
		fmt.Printf("Incoming frame %s \n", bb.Frameid)
		if bb.Frameid >= lastBB.Frameid {
			lastBB = bb
			lastBB.DurationLeft = (*fps) * 2
			frameid, _ := strconv.Atoi(bb.Frameid)
			lastBB.Latency = getFrameNumber() - int64(frameid)
			lastBB.ResponseServer = r.RemoteAddr
		}
	} else {
		fmt.Printf("%v\n", err)
		debug.PrintStack()
	}
}

func handleRequests() {
	clientRouter := mux.NewRouter().StrictSlash(true)
	clientRouter.HandleFunc("/api/result", newBB).Methods("POST")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), clientRouter))
}

func frameCapture() {
	webcam, err := gocv.VideoCaptureDevice(0)
	webcam.Set(gocv.VideoCaptureFrameWidth, 720)
	webcam.Set(gocv.VideoCaptureFrameHeight, 340)
	if err != nil {
		fmt.Printf("Error %v", err)
		panic(err)
	}

	framesForBBRequest := 0
	bufferPos := 0
	for {
		frameNumber := getFrameNumber()
		webcam.Read(buffer[bufferPos])
		framesForBBRequest = (framesForBBRequest + 1) % int(*fps / *bbps)
		if framesForBBRequest == 0 {
			bbrequestchan <- framestruct{
				framenumber:    frameNumber,
				framebufferpos: bufferPos,
			}
		}
		bufferPos = (bufferPos + 1) % (*bufferSize)
		time.Sleep(time.Duration(1000/(*fps)) * time.Millisecond)
	}
}

func frameShow(window *gocv.Window) {
	bufferPos := 0
	time.Sleep(time.Duration(1000 / *fps * (*bufferSize+10)) * time.Millisecond)
	for {
		if dims := buffer[bufferPos].Size(); len(dims) > 0 {
			bbwritelock.Lock()
			if lastBB.DurationLeft > 0 {
				lastBB.DurationLeft = lastBB.DurationLeft - 1
				for _, bb := range lastBB.BBs {
					drawBB(buffer[bufferPos], bb)
				}
				if *showlatency {
					drawLatency(buffer[bufferPos], lastBB)
				}
			}
			bbwritelock.Unlock()
			window.IMShow(*buffer[bufferPos])
			bufferPos = (bufferPos + 1) % (*bufferSize)
		}
		time.Sleep(time.Duration(1000/(*fps)) * time.Millisecond)
		window.WaitKey(1)
	}

}

func drawLatency(mat *gocv.Mat, bb BBresponse) {
	prestart, err := strconv.Atoi(bb.PreStart)
	preend, err := strconv.Atoi(bb.PreEnd)
	if err != nil {
		prestart = 0
		preend = 0
	}
	objstart, err := strconv.Atoi(bb.ObjStart)
	objend, err := strconv.Atoi(bb.ObjEnd)
	if err != nil {
		objstart = 0
		objend = 0
	}
	recstart, err := strconv.Atoi(bb.RecStart)
	recend, err := strconv.Atoi(bb.RecEnd)
	if err != nil {
		recstart = 0
		recend = 0
	}

	gocv.PutText(mat, fmt.Sprintf("From: %s, %s", cloudoredge, strings.Split(bb.ResponseServer, ":")[0]), image.Point{X: 10, Y: 30}, gocv.FontHersheyPlain, 2, BLUE, 3)
	gocv.PutText(mat, fmt.Sprintf("Latency: %dms", bb.Latency), image.Point{X: 10, Y: 60}, gocv.FontHersheyPlain, 2, BLUE, 3)
	gocv.PutText(mat, fmt.Sprintf("Pre %dms", preend-prestart), image.Point{X: 10, Y: 90}, gocv.FontHersheyPlain, 2, YELLOW, 3)
	gocv.PutText(mat, fmt.Sprintf("Obj: %dms ", objend-objstart), image.Point{X: 10, Y: 120}, gocv.FontHersheyPlain, 2, RED, 3)
	gocv.PutText(mat, fmt.Sprintf("Rec: %dms ", recend-recstart), image.Point{X: 10, Y: 150}, gocv.FontHersheyPlain, 2, GREEN, 3)

}

func drawBB(mat *gocv.Mat, bb BB) {
	if bb.Confidence > 0.1 {
		gocv.Rectangle(mat, image.Rect(bb.X1, bb.Y1, bb.X2, bb.Y2), RED, 4)
		gocv.PutText(mat, bb.Label, image.Point{X: bb.X1, Y: bb.Y1 - 20}, gocv.FontHersheyComplexSmall, 2, RED, 4)
		if bb.Label == "person" {
			if bb.Sex != "" {
				gocv.PutText(mat, fmt.Sprintf("Sex: %s", bb.Sex), image.Point{X: bb.X1, Y: bb.Y1 + 30}, gocv.FontHersheyComplexSmall, 2, GREEN, 3)
				gocv.PutText(mat, fmt.Sprintf("Age: %d", bb.Age), image.Point{X: bb.X1, Y: bb.Y1 + 60}, gocv.FontHersheyComplexSmall, 2, GREEN, 3)
			}
			if len(bb.Landmark) > 0 {
				for _, landmark := range bb.Landmark {
					gocv.Rectangle(mat, image.Rect(int(landmark[0]), int(landmark[1]), int(landmark[0]+1), int(landmark[1])+1), GREEN, 5)
				}
			}
		}
	}
}

func sendFrame() {
	client := http.Client{
		Timeout: 500 * time.Millisecond,
	}
	url1 := fmt.Sprintf("http://%s/api/entrypoint", *primaryentrypoint)
	url2 := fmt.Sprintf("http://%s/api/entrypoint", *secondaryentrypoint)

	for {
		frame := <-bbrequestchan
		fmt.Printf("request,framenum %d \n", frame.framenumber)
		err := postFrameRequest(frame, url1, client)
		if err != nil {
			clientCloud := http.Client{
				Timeout: 900 * time.Millisecond,
			}
			fmt.Printf("Unable to perform EDGE request for ,framenum %d \n", frame.framenumber)
			err := postFrameRequest(frame, url2, clientCloud)
			if err != nil {
				cloudoredge = "Offline"
				fmt.Printf("Unable to perform CLOUD request for ,framenum %d \n", frame.framenumber)
			} else {
				cloudoredge = "Cloud"
			}
		} else {
			cloudoredge = "Edge"
		}
	}
}

func postFrameRequest(frame framestruct, url string, client http.Client) error {
	encoded, err := gocv.IMEncode(".jpg", *buffer[frame.framebufferpos])
	if err != nil {
		fmt.Printf("ERROR: %v", err)
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(encoded.GetBytes()))
	if err != nil {
		fmt.Printf("ERROR: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "image/jpg")
	req.Header.Set("client-id", fmt.Sprintf("%d", clientid))
	req.Header.Set("frame-number", fmt.Sprintf("%d", frame.framenumber))
	req.Header.Set("client-result-port", fmt.Sprintf("%d", *port))
	do, err := client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: %v", err)
		return err
	}
	if do.StatusCode != 200 {
		return errors.New("Wrong status code")
	}
	return nil
}

func getFrameNumber() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
