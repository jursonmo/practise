
# taskgo 管理goroutine, 记录goroutine 是否已经结束。用于检查是否有goroutine泄露。

 背景:
一个任务需要起几个goroutine去完成, 但是这个stop 这个任务，需要知道哪些goroutine已经
处理完成，哪些没有处理完成，不然你可能就有goroutine泄露, 我们不能等待goroutine多到影响业务的
时候从用pprof去查看，那时太晚了，而且不容易快速查出问题，
应该是在结束一个任务时，就要保证其下的goroutine的能正常地在规定的时间
退出,否则就打印error, 开发人员提前去查问题。