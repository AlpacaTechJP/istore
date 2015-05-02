package istore

import (
	"fmt"
	"runtime"
	"time"

	"github.com/golang/glog"
)

func humanSize(size uint64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%dKB", size/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%dMB", size/1024/1024)
	}
	return fmt.Sprintf("%dGB", size/1024/1024/1024)
}

func watcher() {
	for {
		if glog.V(3) {
			glog.Info(fmt.Sprintf("PROF: # goroutine = %d", runtime.NumGoroutine()))
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			glog.Info(fmt.Sprintf("PROF: Alloc = %s, HeapAlloc = %s, StackInuse = %s", humanSize(m.Alloc), humanSize(m.HeapAlloc), humanSize(m.StackInuse)))
		}

		select {
		case <-time.After(5 * time.Second):
		}
	}
}
