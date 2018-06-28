// Copyright (c) 2018, Kirill Ovchinnikov
// All rights reserved.

// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:

// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.

// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package main

import (
	"os"
	"time"
)

type taskMonitor struct {
	scheduleStop chan scheduledCancel
	cancelStop   chan scheduledCancel
	terminate    chan os.Signal
	queue        []scheduledCancel
}

type scheduledCancel struct {
	cmd       string
	startTime time.Time
	timeout   time.Duration
}

func newTaskMonitor() *taskMonitor {
	monitor := taskMonitor{
		scheduleStop: make(chan scheduledCancel),
		cancelStop:   make(chan scheduledCancel),
		terminate:    make(chan os.Signal, 1),
		queue:        make([]scheduledCancel, 0)}
	return &monitor
}

func (m taskMonitor) ScheduleTaskToStop(cancelCmd string, timeout uint16) {
	taskToStop := scheduledCancel{cancelCmd, time.Now(), time.Duration(timeout) * time.Second}
	m.scheduleStop <- taskToStop
}

func (m taskMonitor) CancelTask(cmd string) {
	cancelTask := scheduledCancel{cmd, time.Time{}, 0}
	m.cancelStop <- cancelTask
}

func (m taskMonitor) Process() {
	const PollPeriod = 5
	for {
		select {
		case task := <-m.scheduleStop:
			m.queue = append(m.queue, task)
		case task := <-m.cancelStop:
			tmp := m.queue[:0]
			for _, p := range m.queue {
				if p.cmd != task.cmd {
					tmp = append(tmp, p)
				}
			}
			m.queue = tmp
		case <-m.terminate:
			if len(m.queue) > 0 {
				for _, p := range m.queue {
					runTask(p.cmd)
				}
			}
			os.Exit(0)
		case <-time.After(PollPeriod * time.Second):
			tmp := m.queue[:0]
			for _, p := range m.queue {
				if time.Since(p.startTime) < p.timeout {
					tmp = append(tmp, p)
				} else {
					runTask(p.cmd)
				}
			}
			m.queue = tmp
		}
	}
}
