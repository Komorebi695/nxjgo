package nxjpool

import (
	nxjLog "github.com/Komorebi695/nxjgo/log"
	"time"
)

type Worker struct {
	pool *Pool
	// task 任务队列
	task chan func()
	// lastTime 执行任务的最后时间
	lastTime time.Time
}

func (w *Worker) run() {
	w.pool.incRunning()
	go w.running()
}

func (w *Worker) running() {
	defer func() {
		w.pool.descRunning()
		w.pool.workerCache.Put(w)
		if err := recover(); err != nil {
			// 捕获任务执行发生的panic
			if w.pool.PanicHandler != nil {
				w.pool.PanicHandler()
			} else {
				nxjLog.Default().Error(err)
			}
		}
		w.pool.cond.Signal()
	}()
	for f := range w.task {
		if f == nil {
			//w.pool.workerCache.Put(w)
			return
		}
		f()
		// 任务运行完成，worker空闲
		w.pool.PutWorker(w)
	}
}
