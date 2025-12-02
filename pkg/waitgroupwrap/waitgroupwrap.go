package waitgroupwrap

import "sync"

type WaitGroupWrapper struct {
	wg sync.WaitGroup
}

func (w *WaitGroupWrapper) AddRun(cb func()) {
	w.wg.Add(1)
	go func() {
		cb()
		w.wg.Done()
	}()
}
func (w *WaitGroupWrapper) Wait() {
	w.wg.Wait()
}
