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
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type handler struct {
	*state
	H func(e *state, w http.ResponseWriter, r *http.Request) (result *taskResult, genericErr error)
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	taskResult, genericErr := h.H(h.state, w, r)
	if genericErr != nil {
		// I know that http.StatusBadRequest has to be here, but frankly speaking I simply like 418 status
		http.Error(w, fmt.Sprintf("Request error: %s", genericErr.Error()), http.StatusTeapot)
		return
	}
	if taskResult != nil {
		r, _ := json.MarshalIndent(taskResult, "", "  ")
		if taskResult.Retcode != 0 {
			http.Error(w, string(r), http.StatusInternalServerError)
		} else {
			w.Write(r)
		}
	}
}

func defaultHandler(env *state, w http.ResponseWriter, r *http.Request) (result *taskResult, genericErr error) {
	taskParts := strings.Split(r.URL.Path, "/")
	if taskParts[1] == "" {
		genericErr = errors.New("Looks like you didn't specify a task")
	} else {
		genericErr = errors.New("No such task")
	}
	env.log.Err(fmt.Sprintf("Bad request: '%s': %s", r.URL.Path, genericErr.Error()))
	return
}

func taskHandler(env *state, w http.ResponseWriter, r *http.Request) (result *taskResult, genericErr error) {
	preparedUrl := strings.TrimPrefix(strings.ToLower(r.URL.Path), env.config.UrlPrefix)
	urlParts := strings.Split(preparedUrl, "/")

	clientIpStr := r.URL.Query().Get("ip")
	if len(clientIpStr) == 0 || env.config.ExplicitIP == false {
		clientIpStr, _, genericErr = net.SplitHostPort(r.RemoteAddr)
		if genericErr != nil {
			return
		}
		if env.config.RealIPHeader != "" {
			clientIpStr = r.Header.Get(env.config.RealIPHeader)
			if clientIpStr == "" {
				genericErr = errors.New("Empty real IP header, looks like you misconfigured your reverse-proxy")
				return
			}
		}
	}
	clientIp := net.ParseIP(clientIpStr)
	if clientIp == nil {
		genericErr = errors.New("Malformed IP: " + clientIpStr)
		return
	}

	action := urlParts[2]
	taskName := urlParts[1]
	currentTask := env.config.Tasks[taskName]
	switch action {
	case "on":
		env.log.Info(fmt.Sprintf("Starting '%s' for %s...", currentTask.Name, clientIpStr))
		result = currentTask.Start(env, clientIp)
	case "off":
		env.log.Info(fmt.Sprintf("Stopping '%s' for %s by request...", currentTask.Name, clientIpStr))
		result = currentTask.Stop(env, clientIp)
	default:
		env.log.Info(fmt.Sprintf("No action specified, so starting '%s' for %s...", currentTask.Name, clientIpStr))
		result = currentTask.Start(env, clientIp)
	}
	if result.Retcode != 0 {
		env.log.Err(fmt.Sprintf("Failed to execute task '%s': %s", currentTask.Name, result.Err))
	}
	return
}
