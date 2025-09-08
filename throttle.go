package main

import (
	"sync"
	"time"
)

const (
	rateLimit             = 30
	cooldown              = time.Minute
	maxConcurrentRequests = 2
)

var ticker = time.NewTicker(cooldown / rateLimit)
var attempts []time.Time
var attemptsLock sync.Mutex

var concurrentReqs = make(chan struct{}, maxConcurrentRequests)

func init() {
	for range maxConcurrentRequests {
		concurrentReqs <- struct{}{}
	}
}

func GetToken() func() {
	<-concurrentReqs
	return func() {
		concurrentReqs <- struct{}{}
	}
}

func Throttle() {
	for range ticker.C {
		attemptsLock.Lock()
		att := attempts
		if len(att) < rateLimit || time.Since(att[0]) > cooldown {
			att = append(att, time.Now())
			if len(att) > rateLimit {
				att = att[1:]
			}
			attempts = att
			attemptsLock.Unlock()
			return
		}
		attemptsLock.Unlock()
	}
	panic("")
}
