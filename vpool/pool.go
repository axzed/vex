package vpool

import (
	"errors"
	"fmt"
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

// Pool is a place to store the worker
type Pool struct {
	cap          int32         // pool's max size
	running      int32         // worker's count which is running
	workers      []*Worker     // idle worker in pool set in the pool
	expire       time.Duration // work's expire time (beyond this time: need to clean it)
	release      chan sig      // release the resource (pool disable)
	lock         sync.Mutex    // protect the pool's resource for worker
	once         sync.Once     // only release once
	workerCache  sync.Pool     // workerCache to cache
	cond         *sync.Cond    // cond is a condition variable, a rendezvous point for goroutines waiting for or announcing the occurrence of an event.
	PanicHandler func()
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
	// init the pool
	p := &Pool{
		cap:     int32(cap),
		running: 0,
		workers: nil,
		expire:  time.Duration(expire) * time.Second,
		release: make(chan sig, 1),
		lock:    sync.Mutex{},
		once:    sync.Once{},
	}
	p.workerCache.New = func() any {
		return &Worker{
			pool: p,
			task: make(chan func(), 1),
		}
	}
	// init the cond
	p.cond = sync.NewCond(&p.lock)
	// clean the idle worker in goroutine
	go p.expireWorker()
	return p, nil
}

// expireWorker clean the expired idle workers
func (p *Pool) expireWorker() {
	// Do Regular cleaning expired idle worker
	ticker := time.NewTicker(p.expire)
	for range ticker.C {
		// pool is closed, break
		if p.IsClosed() {
			break
		}
		p.lock.Lock()
		// 循环空闲 worker，如果当前时间 - worker最后运行时间 > expire => 清理
		idleWorkers := p.workers
		n := len(idleWorkers) - 1
		if n >= 0 {
			for i, w := range idleWorkers {
				// 没有过期
				if time.Now().Sub(w.lastTime) <= p.expire {
					break
				}
				// 要删除的下标
				n = i
				// put the nil to start the worker.running()
				w.task <- nil
			}
			// 删除过期的idleWorker
			if n >= len(idleWorkers)-1 {
				// 全部要删
				p.workers = idleWorkers[:0]
			} else {
				// 删除部分
				p.workers = idleWorkers[n+1:]
			}
			fmt.Printf("cleaning expired workers done, running:%d, workers:%v \n", p.running, p.workers)
		}
		p.lock.Unlock()
	}
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

	// 3. if don't had idle worker, then new worker
	if p.running < p.cap {
		c := p.workerCache.Get()
		var w *Worker
		// don't had any worker in workerCache you still need to create a new worker
		if c == nil {
			// new a worker
			w = &Worker{
				pool:     p,
				task:     make(chan func(), 1),
				lastTime: time.Time{},
			}
		} else {
			w = c.(*Worker)
		}
		// run the worker
		w.run()
		return w
	}
	// 4. if running worker + idle worker > pool.size then block and wait the worker release
	return p.waitIdleWorker()
}

// waitIdleWorker to wait the idle worker
func (p *Pool) waitIdleWorker() *Worker {
	p.lock.Lock()
	// wait the idle worker
	p.cond.Wait()

	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	if n < 0 {
		p.lock.Unlock()
		// new a worker when you cannot wait the idle worker
		if p.running < p.cap {
			c := p.workerCache.Get()
			var w *Worker
			// don't had any worker in workerCache you still need to create a new worker
			if c == nil {
				// new a worker
				w = &Worker{
					pool:     p,
					task:     make(chan func(), 1),
					lastTime: time.Time{},
				}
			} else {
				w = c.(*Worker)
			}
			// run the worker
			w.run()
			return w
		}
		return p.waitIdleWorker()
	}
	w := idleWorkers[n]
	idleWorkers[n] = nil
	p.workers = idleWorkers[:n]
	p.lock.Unlock()
	return w
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
	// notify the worker had been put to the pool
	p.cond.Signal()
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

// IsClosed to check the pool is closed
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
