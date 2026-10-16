package warnlog

import (
	"sync"
)

type (
	subscriber chan interface{}
)
type Publisher struct {
	sync.RWMutex

	buffer int
	subs   map[subscriber]interface{}
}

func NewPublisher(buffer int) *Publisher {
	return &Publisher{
		buffer: buffer,
		subs:   make(map[subscriber]interface{}),
	}
}

func (p *Publisher) Subscribe(stream interface{}) chan interface{} {
	ch := make(chan interface{}, p.buffer)

	p.Lock()
	p.subs[ch] = stream
	p.Unlock()

	return ch
}

func (p *Publisher) Publish(message interface{}) {
	p.Lock()
	defer p.Unlock()

	for sub := range p.subs {
		sub <- message
	}
}

func (p *Publisher) Evict(sub chan interface{}) {
	p.Lock()
	defer p.Unlock()

	delete(p.subs, sub)
	close(sub)
}

func (p *Publisher) Close() {
	p.Lock()
	defer p.Unlock()

	for sub := range p.subs {
		delete(p.subs, sub)
		close(sub)
	}
}
