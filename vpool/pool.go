package vpool

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const DEFAULTEXPIRETIME = 3

var (
	ErrorInvalidCap        = errors.New("cap can not <= 0")
	ErrorInvalidExpireTime = errors.New("expire time can not <= 0")
	ErrorPoolReleased      = errors.New("pool has been release")
)

// sig is signal empty struct
type sig struct{}

// Pool is a place to store the work
type Pool struct {
	cap     int32         // pool's max size
	running int32         // worker's count which is running
	workers []*Worker     // idle worker in pool set in the pool
	expire  time.Duration // work's expire time (beyond this time: need to clean it)
	release chan sig      // release the resource (pool disable)
	lock    sync.Mutex    // protect the pool's resource for worker
	once    sync.Once     // only release once
}

func NewPool(cap int) (*Pool, error) {
	return NewTimePool(cap, DEFAULTEXPIRETIME)
}

func NewTimePool(cap int, expire int) (*Pool, error) {
	if cap <= 0 {
		return nil, ErrorInvalidCap
	}
	if expire <= 0 {
		return nil, ErrorInvalidExpireTime
	}
	p := &Pool{
		cap:     int32(cap),
		running: 0,
		workers: nil,
		expire:  time.Duration(expire) * time.Second,
		release: make(chan sig, 1),
		lock:    sync.Mutex{},
		once:    sync.Once{},
	}
	go expireWorker()
	return p, nil
}

// expireWorker clean the idle worker in a long time
func expireWorker() {

}

// Submit the task
func (p *Pool) Submit(task func()) error {
	if len(p.release) > 0 {
		return ErrorPoolReleased
	}
	// get a worker in the pool, then exec it with the task
	w := p.GetWorker()
	w.task <- task

	// add task you should increase the running worker's count
	w.pool.incrTheRunningCount()
	return nil
}

// GetWorker is a core method of the worker and task
// when you set the task and bind a worker you will run the worker in this method
func (p *Pool) GetWorker() *Worker {
	// 1. get the worker in pool
	// 2. if had idle worker just get it
	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	if n >= 0 {
		p.lock.Lock()
		w := idleWorkers[n]
		idleWorkers[n] = nil
		p.workers = idleWorkers[:n]
		p.lock.Unlock()
		return w
	}
	// 3. don't had idle worker, then new worker
	if p.running < p.cap {
		// new a worker
		w := &Worker{
			pool:     p,
			task:     make(chan func(), 1),
			lastTime: time.Time{},
		}
		// run the worker
		w.run()
		return w
	}
	// 4. if running worker + idle worker > pool.size then block and wait the worker release
	for {
		p.lock.Lock()
		idleWorkers := p.workers
		n := len(idleWorkers) - 1
		if n < 0 {
			p.lock.Unlock()
			continue
		}
		w := idleWorkers[n]
		idleWorkers[n] = nil
		p.workers = idleWorkers[:n]
		p.lock.Unlock()
		return w
	}
}

// incrTheRunningCount to increase the running worker's count
func (p *Pool) incrTheRunningCount() {
	atomic.AddInt32(&p.running, 1)
}

// decrTheRunningCount to decrease the running worker's count
func (p *Pool) decrTheRunningCount() {
	atomic.AddInt32(&p.running, -1)
}

// PutWorker to set idle worker to the pool
func (p *Pool) PutWorker(w *Worker) {
	w.lastTime = time.Now()
	p.lock.Lock()
	p.workers = append(p.workers, w)
	p.lock.Unlock()
}

// Release the pool
func (p *Pool) Release() {
	p.once.Do(func() {
		// do release once
		p.lock.Lock()
		workers := p.workers
		for i, w := range workers {
			w.task = nil
			w.pool = nil
			workers[i] = nil
		}
		p.lock.Unlock()
		p.release <- sig{}
	})
}

func (p *Pool) IsClosed() bool {
	return len(p.release) <= 0
}

// Restart to restart the pool
func (p *Pool) Restart() bool {
	if len(p.release) <= 0 {
		return true
	}
	_ = <-p.release
	return true
}