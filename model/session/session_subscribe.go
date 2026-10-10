package session

import (
	"scmc/common"
	"sync"
	"time"
)

type sessionTimer struct {
	c chan int // channel to notify on timer invoke
	t *time.Timer
}

type sessionTimerManager struct {
	sync.Mutex

	timers map[string]*sessionTimer // sessionKey as map key
}

func (u *sessionTimerManager) Add(sessionKey string, expiredAt int64, c chan int) {
	u.Lock()
	defer u.Unlock()

	t := time.AfterFunc(time.Until(time.Unix(expiredAt, 0)), func() {
		Timers.TriggerAndRemove(sessionKey)
	})

	u.timers[sessionKey] = &sessionTimer{
		c: c,
		t: t,
	}
}

func (u *sessionTimerManager) Reset(sessionKey string, expiredAt int64) {
	u.Lock()
	defer u.Unlock()

	sess, ok := u.timers[sessionKey]
	if !ok {
		return
	}

	sess.t.Reset(time.Until(time.Unix(expiredAt, 0)))
}

func (u *sessionTimerManager) TriggerAndRemove(sessionKey string) {
	u.Lock()
	defer u.Unlock()

	sess, ok := u.timers[sessionKey]
	if !ok {
		return
	}

	defer common.RecoverAndPrintStack() // avoid send on closed channel
	sess.t.Stop()
	sess.c <- 0

	delete(u.timers, sessionKey)
}

var Timers = &sessionTimerManager{
	timers: make(map[string]*sessionTimer),
}
