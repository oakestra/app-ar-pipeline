package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"log"
	"math/rand"
	"net"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
)

var fps *int
var bufferSize *int
var bbps *int
var showlatency *bool
var primaryentrypoint *string
var cameranum *int
var secondaryentrypoint *string
var serverport *int
var buffer []*gocv.Mat
var bbrequestchan chan framestruct
var randSeed = rand.New(rand.NewSource(time.Now().UnixNano()))
var clientid = randSeed.Intn(999999)
var width = 520.0
var heigth = 360.0

var lastBB BBresponse
var lastFeatures BBresponse
var bbwritelock sync.Mutex
var cloudoredge string

var scalingx = 1.0
var scalingy = 1.0

type Frame struct {
	Image       []byte `json:"frame"`
	OriginalW   int    `json:"origianlw"`
	OriginalH   int    `json:"origianlh"`
	YScaling    string `json:"y_scaling"`
	XScaling    string `json:"x_scaling"`
	ClientId    string `json:"client_id"`
	FrameId     string `json:"frame-number"`
	PreStarted  string `json:"pre_processing_started"`
	PreFinished string `json:"pre_processing_finished"`
}

type framestruct struct {
	framenumber    int64
	framebufferpos int
}

type responsebuffer struct {
	buffer *[]byte
	addr   *net.UDPAddr
}

type BBresponse struct {
	Frameid        string `json:"frame-number"`
	YScaling       string `json:"y_scaling"`
	XScaling       string `json:"x_scaling"`
	BBs            []BB   `json:"results"`
	Landmarks      int    `json:"landmarks"`
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
	cameranum = flag.Int("camera", 0, "wecam id to use")
	bufferSize = flag.Int("buffer", 1, "frame buffer size")
	bbps = flag.Int("bbps", 20, "bounding boxes per second to ber requested")
	primaryentrypoint = flag.String("entry", "131.159.24.170", "entrypoint for the pipeline")
	secondaryentrypoint = flag.String("backupentry", "0.0.0.0", "backup cloud entrypoint for the pipeline")
	showlatency = flag.Bool("latency", false, "show the latency")
	serverport = flag.Int("serverport", 50000, "server listening port")

	flag.Parse()
	fmt.Printf("Started client with id %d - %d fps, %d framebuffer size, %d BB per second \n", clientid, *fps, *bufferSize, *bbps)

	bbwritelock = sync.Mutex{}
	buffer = make([]*gocv.Mat, *bufferSize)
	bbrequestchan = make(chan framestruct, 0)
	for i := 0; i < *bufferSize; i++ {
		matrix := gocv.NewMat()
		buffer[i] = &matrix
	}

	window := gocv.NewWindow("Client")
	go frameCapture()
	go sendReceiveFrameRoutine()
	frameShow(window)
}

// endpoint /api/result
func newBB(data []byte, addr net.Addr) {
	var bb BBresponse
	err := json.Unmarshal(data, &bb)
	if err == nil {
		bbwritelock.Lock()
		scalingx, _ = strconv.ParseFloat(bb.XScaling, 64)
		scalingy, _ = strconv.ParseFloat(bb.YScaling, 64)
		defer bbwritelock.Unlock()
		fmt.Printf("Incoming frame %s \n", bb.Frameid)
		//update last bounding boxes
		if bb.Frameid >= lastBB.Frameid {
			lastBB = bb
			lastBB.DurationLeft = (*fps) * 2
			frameid, _ := strconv.Atoi(bb.Frameid)
			lastBB.Latency = getFrameNumber() - int64(frameid)
			lastBB.ResponseServer = addr.String()
		}
		//update last landmarks
		if bb.Landmarks > 0 && bb.Frameid >= lastFeatures.Frameid {
			lastFeatures = bb
			lastFeatures.DurationLeft = (*fps) * 2
			frameid, _ := strconv.Atoi(bb.Frameid)
			lastFeatures.Latency = getFrameNumber() - int64(frameid)
			lastFeatures.ResponseServer = addr.String()
		}
	} else {
		fmt.Printf("%v\n", err)
		debug.PrintStack()
	}
}

func frameCapture() {
	webcam, err := gocv.VideoCaptureDevice(*cameranum)
	webcam.Set(gocv.VideoCaptureFrameWidth, width)
	webcam.Set(gocv.VideoCaptureFrameHeight, heigth)
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
			drawbb := BBresponse{}
			if lastFeatures.DurationLeft > 0 {
				lastFeatures.DurationLeft = lastFeatures.DurationLeft - 1
				lastBB.DurationLeft = lastBB.DurationLeft - 1
				drawbb = lastFeatures
			} else if lastBB.DurationLeft > 0 {
				lastBB.DurationLeft = lastBB.DurationLeft - 1
				drawbb = lastBB
			}
			if drawbb.BBs != nil {
				for _, bb := range drawbb.BBs {
					drawBB(buffer[bufferPos], bb)
				}
				if *showlatency {
					drawLatency(buffer[bufferPos], drawbb)
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

	//gocv.PutText(mat, fmt.Sprintf("From: %s, %s", cloudoredge, strings.Split(bb.ResponseServer, ":")[0]), image.Point{X: 10, Y: 30}, gocv.FontHersheyPlain, 2, BLUE, 3)
	gocv.PutText(mat, fmt.Sprintf("Latency: %dms", bb.Latency), image.Point{X: 10, Y: 60}, gocv.FontHersheyPlain, 1.5, BLUE, 3)
	gocv.PutText(mat, fmt.Sprintf("Pre %dms", preend-prestart), image.Point{X: 10, Y: 90}, gocv.FontHersheyPlain, 1.5, YELLOW, 3)
	gocv.PutText(mat, fmt.Sprintf("Obj: %dms ", objend-objstart), image.Point{X: 10, Y: 120}, gocv.FontHersheyPlain, 1.5, RED, 3)
	if recend-recstart > 0 {
		gocv.PutText(mat, fmt.Sprintf("Rec: %dms ", recend-recstart), image.Point{X: 10, Y: 150}, gocv.FontHersheyPlain, 1.5, GREEN, 3)
	}

}

func drawBB(mat *gocv.Mat, bb BB) {
	if bb.Confidence > 0.2 {
		bb = scaleResult(bb)
		gocv.Rectangle(mat, image.Rect(bb.X1, bb.Y1, bb.X2, bb.Y2), RED, 4)
		if bb.Label == "person" {
			gocv.PutText(mat, bb.Label, image.Point{X: bb.X1, Y: bb.Y1 - 20}, gocv.FontHersheyComplexSmall, 2, GREEN, 4)
			if bb.Sex != "" {
				//gocv.PutText(mat, fmt.Sprintf("Sex: %s", bb.Sex), image.Point{X: bb.X1, Y: bb.Y1 + 30}, gocv.FontHersheyComplexSmall, 2, GREEN, 3)
				//gocv.PutText(mat, fmt.Sprintf("Age: %d", bb.Age), image.Point{X: bb.X1, Y: bb.Y1 + 60}, gocv.FontHersheyComplexSmall, 2, GREEN, 3)
			}
			if len(bb.Landmark) > 0 {
				for _, landmark := range bb.Landmark {
					gocv.Rectangle(mat, image.Rect(int(landmark[0]), int(landmark[1]), int(landmark[0]+1), int(landmark[1])+1), GREEN, 5)
				}
			}
			return
		}
		gocv.PutText(mat, bb.Label, image.Point{X: bb.X1, Y: bb.Y1 - 20}, gocv.FontHersheyComplexSmall, 2, RED, 4)
	}
}

func scaleResult(bb BB) BB {
	bb.X1 = int(float64(bb.X1) * scalingx)
	bb.X2 = int(float64(bb.X2) * scalingx)
	bb.Y1 = int(float64(bb.Y1) * scalingy)
	bb.Y2 = int(float64(bb.Y2) * scalingy)
	return bb
}

func sendReceiveFrameRoutine() {

	udp_chan := make(chan responsebuffer)

	rand.Seed(time.Now().UnixNano())
	// Get a random number between 30000 and 40000.
	port := rand.Intn(40000-30000+1) + 30000

	// Create a UDP socket.
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: port,
	})
	if err != nil {
		log.Fatalln(err)
	}

	//listen for UDP messages
	go func() {
		for {
			buffer := make([]byte, 50000)
			n, addr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				fmt.Println(err)
				return
			}

			// Send the buffer to the `udp_chan` channel.
			responseBuffer := buffer[:n]
			udp_chan <- responsebuffer{
				buffer: &responseBuffer,
				addr:   addr,
			}
		}
	}()

	//received UDP responses and send UDP frames
	timeout := time.After(5 * time.Second)
	for {
		select {
		case <-timeout:
			// if nothing happens for more than 1 sec reboot connection
			timeout = time.After(1 * time.Second)
			conn.Close()
			time.Sleep(time.Millisecond * 100)
			go sendReceiveFrameRoutine()
			return
		case responsebuffer := <-udp_chan:
			// Handle the UDP message.
			timeout = time.After(1 * time.Second)
			newBB(*responsebuffer.buffer, responsebuffer.addr)
		case data := <-bbrequestchan:
			// Send frame to UDP.
			sendFrameRequest(data, conn)
		}
	}

}

func connect() *net.UDPConn {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.ParseIP(*primaryentrypoint),
		Port: *serverport,
	})
	if err != nil {
		fmt.Printf("ERROR: %v size: %d", err)
		fmt.Printf("Unable to perform EDGE request \n")
		//if it fails try sending to secondary endpoint
		conn, err = net.DialUDP("udp", nil, &net.UDPAddr{
			IP:   net.ParseIP(*primaryentrypoint),
			Port: *serverport,
		})
		if err != nil {
			cloudoredge = "Offline"
			fmt.Printf("Unable to perform CLOUD request\n")
		}
		cloudoredge = "Cloud"
	} else {
		cloudoredge = "Edge"
	}
	return conn
}

func sendFrameRequest(data framestruct, conn *net.UDPConn) {
	fmt.Printf("request,framenum %d \n", data.framenumber)
	if buffer[data.framebufferpos].Empty() {
		return
	}
	encoded, err := gocv.IMEncodeWithParams(gocv.JPEGFileExt, *buffer[data.framebufferpos], []int{gocv.IMWriteJpegQuality, 40}) //gocv.IMEncode(".jpg", *buffer[data.framebufferpos])
	//compressed_bytes = gocv.EncodeImage(".jpg", mat, quality=95)
	if err != nil {
		fmt.Printf("ERROR: %v", err)
		return
	}

	frame := Frame{
		Image:       encoded.GetBytes(),
		OriginalW:   int(width),
		OriginalH:   int(heigth),
		YScaling:    "",
		XScaling:    "",
		ClientId:    fmt.Sprintf("%d", clientid),
		FrameId:     fmt.Sprintf("%d", data.framenumber),
		PreStarted:  "",
		PreFinished: "",
	}

	byteMessage, err := json.Marshal(frame)
	if err != nil {
		fmt.Printf("ERROR: %v", err)
		return
	}
	//try sending to primary endpoint
	//_, err = conn.WriteToUDP(byteMessage, &net.UDPAddr{
	//	IP:   net.ParseIP(*primaryentrypoint),
	//	Port: *serverport,
	//})
	fmt.Printf("send to: %s:%d \n", *primaryentrypoint, *serverport)
	_, err = conn.WriteToUDP(byteMessage, &net.UDPAddr{
		IP:   net.ParseIP(*primaryentrypoint),
		Port: *serverport,
	})
	if err != nil {
		fmt.Printf("ERROR: %v \n", err)
		fmt.Printf("Message size: %d \n", len(byteMessage))
	}
}

func getFrameNumber() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
