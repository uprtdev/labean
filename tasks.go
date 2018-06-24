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
	"bytes"
	"os/exec"
	"strings"
	"syscall"
)

type task struct {
	ID      string `json:"name"`
	TurnOn  string `json:"on_command"`
	TurnOff string `json:"off_command"`
	Timeout uint16 `json:"timeout"`
}

type taskResult struct {
	Command string `json:"commandLine"`
	Retcode int    `json:"returnCode"`
	Err     string `json:"error,omitempty"`
	StdErr  string `json:"stderr,omitempty"`
	StdOut  string `json:"stdout,omitempty"`
	Timeout uint16 `json:"timeoutInSeconds,omitempty"`
}

func prepareCommand(ip string, ServerIP string, cmd string) string {
	s := strings.Replace(cmd, "{clientIP}", ip, -1)
	s = strings.Replace(s, "{ServerIP}", ServerIP, -1)
	return s
}

func runTask(cmd string) taskResult {
	var outbuf, errbuf bytes.Buffer

	var result taskResult
	result.Command = cmd

	args := strings.Fields(cmd)
	command := exec.Command(args[0], args[1:]...)
	command.Stdout = &outbuf
	command.Stderr = &errbuf
	err := command.Run()
	result.StdErr = errbuf.String()
	result.StdOut = outbuf.String()
	if err != nil {
		result.Retcode = -1
		result.Err = err.Error()
	}

	if exitError, ok := err.(*exec.ExitError); ok {
		result.Retcode = exitError.Sys().(syscall.WaitStatus).ExitStatus()
	}
	return result
}

func (c task) Start(env *state, ip string) *taskResult {
	cmd := prepareCommand(ip, env.config.ServerIP, c.TurnOn)
	result := runTask(cmd)
	result.Timeout = c.Timeout
	if result.Retcode != 0 {
		return &result
	}
	if c.Timeout != 0 {
		cmd := prepareCommand(ip, env.config.ServerIP, c.TurnOff)
		env.monitor.ScheduleTaskToStop(cmd, c.Timeout)
	}
	return &result
}

func (c task) Stop(env *state, ip string) *taskResult {
	cmd := prepareCommand(ip, env.config.ServerIP, c.TurnOff)
	result := runTask(cmd)
	result.Timeout = c.Timeout
	if result.Retcode == 0 {
		env.monitor.CancelTask(cmd)
	}
	return &result
}
