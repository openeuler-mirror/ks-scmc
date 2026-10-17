package tasks

import (
	"time"

	"scmc/model/registry"
)

type PushImage struct{}

func (p *PushImage) Interval() time.Duration {
	return time.Minute
}

func (p *PushImage) RunAtInit() bool {
	return true
}

func (p *PushImage) Run() {
	registry.CheckImagePush()
}
