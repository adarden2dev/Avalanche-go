// (c) 2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package scheduler

import (
	"time"

	"github.com/ava-labs/avalanchego/snow/engine/common"
)

const (
	fromVMSize = 1024
)

type Scheduler interface {
	Dispatch(startTime time.Time)
	SetStartTime(t time.Time)
	Close()
}

// scheduler to control the signal dispatching from a vm to consensus engine
type scheduler struct {
	activationTime time.Time
	fromVM         <-chan common.Message
	toEngine       chan<- common.Message
	newStartTime   chan time.Time
}

func New(toEngine chan<- common.Message, activationTime time.Time) (Scheduler, chan<- common.Message) {
	vmToEngine := make(chan common.Message, fromVMSize)
	return &scheduler{
		activationTime: activationTime,
		fromVM:         vmToEngine,
		toEngine:       toEngine,
		newStartTime:   make(chan time.Time),
	}, vmToEngine
}

func (s *scheduler) Dispatch(startTime time.Time) {
waitloop:
	for {
		timer := time.NewTimer(time.Until(startTime))
		select {
		case <-timer.C:
		case newStartTime, ok := <-s.newStartTime:
			if !ok {
				timer.Stop()
				return
			}
			startTime = newStartTime
			timer.Stop()
			continue waitloop
		}

		msgSent := false // send only one signal to engine per time window, after Fork
		for {
			select {
			case msg := <-s.fromVM:
				if !msgSent {
					s.toEngine <- msg
					msgSent = !time.Now().Before(s.activationTime)
				}
			case newStartTime, ok := <-s.newStartTime:
				if !ok {
					return
				}
				startTime = newStartTime
				continue waitloop
			}
		}
	}
}

func (s *scheduler) SetStartTime(t time.Time) {
	s.newStartTime <- t
}

func (s *scheduler) Close() {
	close(s.newStartTime)
}
