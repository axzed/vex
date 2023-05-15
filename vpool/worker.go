package vpool

import (
	vexLog "github.com/axzed/vex/log"
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
	// incr the running count of worker
	w.pool.incrTheRunningCount()
	go w.running()
}

// running is the function to run task in cycle
func (w *Worker) running() {
	// catch the panic by task
	defer func() {
		// handle some logic when panic happened
		w.pool.decrTheRunningCount()
		w.pool.workerCache.Put(w)
		if err := recover(); err != nil {
			if w.pool.PanicHandler != nil {
				w.pool.PanicHandler()
			} else {
				vexLog.Default().Error(err)
			}
		}
		w.pool.cond.Signal()
	}()
	for f := range w.task {
		if f == nil {
			// put the worker to cache when task had done
			w.pool.workerCache.Put(w)
			return
		}
		f()
		// task ending worker become idle, need to return the worker to pool
		w.pool.PutWorker(w)
	}
}
