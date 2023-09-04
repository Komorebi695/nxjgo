package nxjpool

import (
	"errors"
	"fmt"
	"github.com/Komorebi695/nxjgo/config"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultExpire = 5
)

var (
	ErrorInValidCap    = errors.New("cap cannot be less than or equal to 0")
	ErrorInValidExpire = errors.New("expire cannot be less than or equal to 0")
	ErrorHasClosed     = errors.New("pool has been released")
)

type sig struct {
}

type Pool struct {
	// cap 容量 pool max map
	cap int32
	// running 正在运行的worker数量
	running int32
	// workers 空闲worker
	workers []*Worker
	// expire 过期时间，空闲worker超过这个时间回收掉
	expire time.Duration
	// release 释放资源 pool就不能使用了
	release chan sig
	// lock 保护pool里面的相关资源的安全
	lock sync.Mutex
	// once 释放只能调用一次，不能多次调用
	once sync.Once
	// workerCache worker缓存
	workerCache sync.Pool
	// cond
	cond *sync.Cond
	// PanicHandler
	PanicHandler func()
}

func NewPool(cap int) (*Pool, error) {
	return NewTimePool(cap, DefaultExpire)
}

func NewPoolConf() (*Pool, error) {
	capacity, ok := config.Conf.Pool["cap"]
	if !ok {
		panic("conf pool.cap not config")
	}
	return NewTimePool(int(capacity.(int64)), DefaultExpire)
}

func NewTimePool(cap int, expire int) (*Pool, error) {
	if cap <= 0 {
		return nil, ErrorInValidCap
	}
	if expire <= 0 {
		return nil, ErrorInValidExpire
	}
	p := &Pool{
		cap:     int32(cap),
		expire:  time.Duration(expire) * time.Second,
		release: make(chan sig, 1),
	}
	p.workerCache.New = func() any {
		return &Worker{
			pool: p,
			task: make(chan func(), 1),
		}
	}
	p.cond = sync.NewCond(&p.lock)
	go p.expireWorker()

	return p, nil
}

func (p *Pool) expireWorker() {
	// 定时清理过期的worker
	ticker := time.NewTicker(p.expire)
	for range ticker.C {
		if p.IsClosed() {
			break
		}
		// 循环空闲的workers 如果当前时间和worker的最后运行任务时间差大于expire，进行清理。
		p.lock.Lock()
		idleWorkers := p.workers
		n := -1
		for i, w := range idleWorkers {
			if time.Now().Sub(w.lastTime) <= p.expire {
				break
			}
			// 需要清理
			n = i
			w.task <- nil
			idleWorkers[i] = nil
		}
		if n > -1 {
			if n >= len(idleWorkers)-1 {
				p.workers = idleWorkers[:0]
			} else {
				p.workers = idleWorkers[n+1:]
			}
		}
		//fmt.Printf("running:%d, workers:%v\n", p.running, p.workers)
		p.lock.Unlock()
	}
}

func (p *Pool) Submit(task func()) error {
	if len(p.release) > 0 {
		return ErrorHasClosed
	}
	// 从协程池中获取一个worker执行任务
	w := p.GetWorker()
	w.task <- task
	return nil
}

func (p *Pool) GetWorker() *Worker {
	// 1. 目的获取pool里面的worker
	// 2. 如果有worker直接获取
	p.lock.Lock()
	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	if n >= 0 {
		w := idleWorkers[n]
		idleWorkers[n] = nil
		p.workers = idleWorkers[:n]
		p.lock.Unlock()
		return w
	}
	p.lock.Unlock()
	// 3. 如果没有空闲worker，需要新建worker
	if p.running < p.cap {
		// 运行的小于pool的容量
		c := p.workerCache.Get()
		var w *Worker
		if c == nil {
			w = &Worker{
				pool: p,
				task: make(chan func(), 1),
			}
		} else {
			w = c.(*Worker)
		}

		w.run()
		return w
	}
	// 4. 如果正在运行的worker 大于pool的容量，阻塞等待worker释放
	//for {
	//	p.lock.Lock()
	//	idleWorkers = p.workers
	//	n = len(idleWorkers) - 1
	//	if n < 0 {
	//		p.lock.Unlock()
	//		continue
	//	}
	//	w := idleWorkers[n]
	//	idleWorkers[n] = nil
	//	p.workers = idleWorkers[:n]
	//	p.lock.Unlock()
	//	return w
	//}
	return p.waitIdleWorker()
}

func (p *Pool) waitIdleWorker() *Worker {
	p.lock.Lock()
	p.cond.Wait()
	fmt.Println("被通知，有空闲worker.")
	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	if n < 0 {
		p.lock.Unlock()
		if p.running < p.cap {
			// 运行的小于pool的容量
			c := p.workerCache.Get()
			var w *Worker
			if c == nil {
				w = &Worker{
					pool: p,
					task: make(chan func(), 1),
				}
			} else {
				w = c.(*Worker)
			}
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

func (p *Pool) incRunning() {
	atomic.AddInt32(&p.running, 1)
}

func (p *Pool) PutWorker(w *Worker) {
	w.lastTime = time.Now()
	p.lock.Lock()
	p.workers = append(p.workers, w)
	p.cond.Signal()
	p.lock.Unlock()
}

func (p *Pool) descRunning() {
	atomic.AddInt32(&p.running, -1)
}

func (p *Pool) Release() {
	p.once.Do(func() {
		p.lock.Lock()
		workers := p.workers
		for i, w := range workers {
			w.task = nil
			w.pool = nil
			workers[i] = nil
		}
		p.workers = nil
		p.lock.Unlock()
		p.release <- sig{}
	})
}

func (p *Pool) IsClosed() bool {
	return len(p.release) > 0
}

func (p *Pool) Restart() bool {
	if len(p.release) <= 0 {
		return true
	}
	_ = <-p.release
	return true
}

func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

func (p *Pool) Free() int {
	return int(p.cap - p.running)
}
