package warnlog

import (
	"sync"
)

var instance = &Publisher{
	buffer: 5,
	subs:   make(map[chan interface{}]bool, 1),
}

type Publisher struct {
	sync.RWMutex

	buffer int
	subs   map[chan interface{}]bool
}

func (p *Publisher) Subscribe() chan interface{} {
	ch := make(chan interface{}, p.buffer)

	p.Lock()
	defer p.Unlock()
	p.subs[ch] = true

	return ch
}

func (p *Publisher) Publish(msg interface{}) {
	p.Lock()
	defer p.Unlock()

	for sub := range p.subs {
		sub <- msg
	}
}

func (p *Publisher) Evict(ch chan interface{}) {
	p.Lock()
	defer p.Unlock()

	delete(p.subs, ch)
	close(ch)
}

func (p *Publisher) Close() {
	p.Lock()
	defer p.Unlock()

	for sub := range p.subs {
		delete(p.subs, sub)
		close(sub)
	}
}

func Subscribe() chan interface{} {
	return instance.Subscribe()
}

func Publish(msg interface{}) {
	instance.Publish(msg)
}

func Evict(ch chan interface{}) {
	instance.Evict(ch)
}
