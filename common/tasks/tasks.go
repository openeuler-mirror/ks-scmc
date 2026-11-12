package tasks

import (
	"time"
)

type Task interface {
	Interval() time.Duration
	RunAtInit() bool
	Run()
}

func AddTask(ts ...Task) {
	for _, t := range ts {
		go func(t Task) {
			if t.RunAtInit() {
				t.Run()
			}

			ticker := time.NewTicker(t.Interval())
			for range ticker.C {
