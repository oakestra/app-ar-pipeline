package taskQueue

import (
	"sync"
)

type Frame struct {
	Image         []byte
	OriginalW     int
	OriginalH     int
	ClientId      string
	FrameId       string
	ClientAddress string
	ArrivalTime   string
}

type taskFifo struct {
	currentFrame *Frame
	nextTask     *taskFifo
}

type TaskQueueStruct struct {
	taskQueueSize int
	firstTask     *taskFifo
	lastTask      *taskFifo
	addRemoveLock *sync.Mutex
	notify        chan bool
}

var taskQueue TaskQueueStruct
var once sync.Once

const (
	MaxTaskQueueSize = 5
)

type FrameTaskQueue interface {
	Push(f *Frame)
	Pop() *Frame
	GetNotificationChannel() chan bool
}

func GetTaskQueue() FrameTaskQueue {
	once.Do(func() {
		taskQueue = TaskQueueStruct{
			taskQueueSize: 0,
			firstTask:     nil,
			addRemoveLock: &sync.Mutex{},
			notify:        make(chan bool, 999),
		}
	})
	return &taskQueue
}

func (t *TaskQueueStruct) Push(f *Frame) {
	t.addRemoveLock.Lock()
	defer t.addRemoveLock.Unlock()
	if t.lastTask == nil {
		t.lastTask = &taskFifo{
			currentFrame: f,
			nextTask:     nil,
		}
		t.firstTask = t.lastTask
		t.taskQueueSize = 1
	} else {
		newLastTask := &taskFifo{
			currentFrame: f,
			nextTask:     nil,
		}
		t.lastTask.nextTask = newLastTask
		t.lastTask = newLastTask
		t.taskQueueSize = t.taskQueueSize + 1
		if t.taskQueueSize > MaxTaskQueueSize {
			t.taskQueueSize = t.taskQueueSize - 1
			t.firstTask = t.firstTask.nextTask
		}
	}
	t.notify <- true
}

func (t *TaskQueueStruct) Pop() *Frame {
	t.addRemoveLock.Lock()
	defer t.addRemoveLock.Unlock()
	if t.firstTask == nil {
		return nil
	}
	returnTask := t.firstTask
	t.firstTask = t.firstTask.nextTask
	if t.firstTask == nil {
		t.lastTask = nil
	}
	return returnTask.currentFrame
}

func (t TaskQueueStruct) GetNotificationChannel() chan bool {
	return t.notify
}
