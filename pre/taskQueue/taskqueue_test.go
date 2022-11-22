package taskQueue

import (
	"fmt"
	"sync"
	"testing"
)

func TestTaskAdd1(t *testing.T) {
	once = sync.Once{}
	mytaskqueue := GetTaskQueue()
	mytaskqueue.Push(&Frame{
		Image:      nil,
		ClientId:   "1",
		FrameId:    "1",
		ClientPort: "80",
	})
	frameres := mytaskqueue.Pop()
	if frameres.FrameId != "1" {
		t.Fatalf("Frame id %s!=%s", frameres.FrameId, "1")
	}
}

func TestTaskAdd2(t *testing.T) {
	once = sync.Once{}
	mytaskqueue := GetTaskQueue()
	mytaskqueue.Push(&Frame{
		Image:      nil,
		ClientId:   "1",
		FrameId:    "1",
		ClientPort: "80",
	})
	mytaskqueue.Push(&Frame{
		Image:      nil,
		ClientId:   "1",
		FrameId:    "2",
		ClientPort: "80",
	})
	frameres := mytaskqueue.Pop()
	if frameres.FrameId != "1" {
		t.Fatalf("Frame id %s!=%s", frameres.FrameId, "1")
	}
}

func TestTaskAddN(t *testing.T) {
	once = sync.Once{}
	mytaskqueue := GetTaskQueue()
	for i := 0; i < 52; i++ {
		mytaskqueue.Push(&Frame{
			Image:      nil,
			ClientId:   "1",
			FrameId:    fmt.Sprintf("%d", i),
			ClientPort: "80",
		})
	}
	frameres := mytaskqueue.Pop()
	if frameres.FrameId != "2" {
		t.Fatalf("Frame id %s!=%s", frameres.FrameId, "3")
	}
}

func TestPopN(t *testing.T) {
	once = sync.Once{}
	mytaskqueue := GetTaskQueue()
	for i := 0; i < 52; i++ {
		mytaskqueue.Push(&Frame{
			Image:      nil,
			ClientId:   "1",
			FrameId:    fmt.Sprintf("%d", i),
			ClientPort: "80",
		})
	}
	for i := 2; i < 52; i++ {
		frameres := mytaskqueue.Pop()
		if frameres.FrameId != fmt.Sprintf("%d", i) {
			t.Fatalf("Frame id %s!=%d", frameres.FrameId, i)
		}
	}
	frameres := mytaskqueue.Pop()
	if frameres != nil {
		t.Fatalf("Nor more frame showld be left in the queue")
	}
}
