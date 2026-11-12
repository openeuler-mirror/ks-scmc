package tasks

import (
	"time"
)

type Task interface {
	Interval() time.Duration
	RunAtInit() bool
	Run()
