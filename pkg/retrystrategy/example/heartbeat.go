package main

import (
	"log"
	"sync"
	"time"

	"github.com/jursonmo/practise/pkg/retrystrategy"
)

func main() {
	hbBackoff := retrystrategy.NewHeartbeatBackoff(5*time.Second, time.Second)
	strategy := retrystrategy.NewRetryStrategy(3, hbBackoff)

	pingCh := make(chan struct{}, 3)
	replyCh := make(chan struct{}, 3)

	receive := 0
	sendheatbeat := func() {
		pingCh <- struct{}{}
		//wait for heatbeat response or timeout
		t := time.NewTimer(20 * time.Millisecond)
		select {
		case <-t.C:
			log.Println("wait for heatbeat reply timeout")
		case <-replyCh:
			log.Println("get heatbeat reply")
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			if !strategy.Retryable() {
				log.Printf("can't retry again, %d", strategy.Tried())
				close(pingCh)
				return
			}
			d := strategy.Duration()
			log.Printf("next duration:%v to send heartbeat", d)
			time.Sleep(d)
			//send heatbeet
			sendheatbeat()
		}
	}()

	//recevie heatbeat
	go func() {
		defer wg.Done()
		for {
			select {
			case _, open := <-pingCh:
				if !open {
					log.Println("net channel is closed")
					return
				}

				if receive >= 2 {
					log.Println("simulate can't get heatbeat")
					return
				}

				receive++
				log.Println("get heatbeat, and send heatbeat reply and reset strategy")
				strategy.Reset()
				//send heatbeat reply
				replyCh <- struct{}{}
			}
		}
	}()
	wg.Wait()
}
