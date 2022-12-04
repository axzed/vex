package vpool

import (
	"time"
)

// Worker handle the work
type Worker struct {
	pool     *Pool
	task     chan func() // handle work's function
	lastTime time.Time   // exec work's deadline time
}

// run is a method to run the Worker
// start running
func (w *Worker) run() {
	go w.running()
}

// running is the function to run task in cycle
func (w *Worker) running() {
	for f := range w.task {
		if f == nil {
			return
		}
		f()
		// task ending worker become idle, need to return the worker to pool
		w.pool.PutWorker(w)
		// set task - 1
		w.pool.decrTheRunningCount()
	}
}
