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
	"fmt"
	"net"
	"os/exec"
	"strings"
	"syscall"
)

type IpAddrType int64

const (
	IPv4 IpAddrType = 0
	IPv6            = 1
)

type task struct {
	Name        string `json:"name"`
	TurnOn      string `json:"on_command"`
	TurnOff     string `json:"off_command"`
	TurnOnIpV6  string `json:"on_command_v6"`
	TurnOffIpV6 string `json:"off_command_v6"`
	Timeout     uint16 `json:"timeout"`
}

type taskResult struct {
	Command string `json:"commandLine"`
	Retcode int    `json:"returnCode"`
	Err     string `json:"error,omitempty"`
	StdErr  string `json:"stderr,omitempty"`
	StdOut  string `json:"stdout,omitempty"`
	Timeout uint16 `json:"timeoutInSeconds,omitempty"`
	Ip      net.IP `json:"clientIp"`
}

func prepareIp(ip net.IP) (net.IP, IpAddrType) {
	// let's perform conversion first to check if we have IPv4 or IPv6
	// and second to extract IPv4 from something like "::FFFF:192.168.0.1"
	ipv4 := ip.To4()
	if ipv4 == nil {
		return ip, IPv6
	} else {
		return ip.To4(), IPv4
	}
}

func generateCommand(ip net.IP, ServerIp net.IP, cmd string) string {
	s := strings.Replace(cmd, "{clientIP}", ip.String(), -1)
	s = strings.Replace(s, "{serverIP}", ServerIp.String(), -1)
	return s
}

func resultCommandMissing(ip net.IP) taskResult {
	var result taskResult
	result.Retcode = -1
	result.Ip = ip
	result.Err = "No command declared in config for the client's type of IP address"
	return result
}

func runCmd(cmd string) taskResult {
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

func (c task) Start(env *state, ip net.IP) *taskResult {
	ip, ipType := prepareIp(ip)
	var cmd string
	if ipType == IPv4 {
		if c.TurnOn == "" {
			result := resultCommandMissing(ip)
			return &result
		}
		if c.Timeout != 0 && c.TurnOff == "" {
			result := resultCommandMissing(ip)
			return &result
		}
		cmd = generateCommand(ip, env.config.ServerIp, c.TurnOn)
	} else {
		if c.TurnOnIpV6 == "" {
			result := resultCommandMissing(ip)
			return &result
		}
		if c.Timeout != 0 && c.TurnOffIpV6 == "" {
			result := resultCommandMissing(ip)
			return &result
		}
		cmd = generateCommand(ip, env.config.ServerIpv6, c.TurnOnIpV6)
	}

	result := runCmd(cmd)
	result.Timeout = c.Timeout
	result.Ip = ip

	env.log.Info(fmt.Sprintf("Task start result: %#v", result))
	if result.Retcode == 0 && c.Timeout != 0 {
		if ipType == IPv4 {
			cmd = generateCommand(ip, env.config.ServerIp, c.TurnOff)
		} else {
			cmd = generateCommand(ip, env.config.ServerIpv6, c.TurnOffIpV6)
		}
		env.monitor.ScheduleTaskToStop(cmd, c.Timeout)
	}
	return &result
}

func (c task) Stop(env *state, ip net.IP) *taskResult {
	ip, ipType := prepareIp(ip)
	var cmd string
	if ipType == IPv4 {
		if c.TurnOff == "" {
			result := resultCommandMissing(ip)
			return &result
		}
		cmd = generateCommand(ip, env.config.ServerIp, c.TurnOff)
	} else {
		if c.TurnOffIpV6 == "" {
			result := resultCommandMissing(ip)
			return &result
		}
		cmd = generateCommand(ip, env.config.ServerIpv6, c.TurnOffIpV6)
	}
	result := runCmd(cmd)
	result.Ip = ip
	env.log.Info(fmt.Sprintf("Task stop result: %#v", result))
	if result.Retcode == 0 {
		env.monitor.CancelTask(cmd)
	}
	return &result
}
