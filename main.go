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
	"fmt"
	"log"
	"log/syslog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

const verMajor = 0
const verMinor = 1

type state struct {
	config  *appConfig
	monitor *taskMonitor
	log     *syslog.Writer
}

func printUsage() {
	log.Print("Usage: ./labean <config_file_path>")
	log.Print("Default config file path is ./labean.conf")
}

func main() {
	syslogger, _ := syslog.New(syslog.LOG_INFO, "labean")
	syslogger.Notice(fmt.Sprintf("Labean %d.%d started.", verMajor, verMinor))

	configPath := "./labean.conf"

	if len(os.Args) > 1 {
		if os.Args[1] == "?" || os.Args[1] == "-?" || os.Args[1] == "help" ||
			os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "/?" {
			printUsage()
			os.Exit(0)
		}
		configPath = os.Args[1]
	}

	config, err := loadConfig(configPath)
	if err != nil {
		log.Printf("Failed to load config: %s", err)
		log.Print("Please run `labean -h` for usage info and see docs for config examples")
		os.Exit(1)
	}

	if os.Geteuid() != 0 {
		syslogger.Warning("Warning: It seems that you started me without root permissions")
		syslogger.Warning("So, if your tasks require superuser rights they will not work")
	}

	monitor := newTaskMonitor()
	signal.Notify(monitor.terminate, syscall.SIGINT, syscall.SIGTERM)
	env := &state{config, monitor, syslogger}

	http.Handle("/", handler{env, defaultHandler})
	for _, cmd := range config.Tasks {
		http.Handle("/"+cmd.ID+"/", handler{env, taskHandler})
		syslogger.Info(fmt.Sprintf("Adding handle for '%s' task...", cmd.ID))
	}

	go monitor.Process()

	syslogger.Info(fmt.Sprintf("Starting server on '%s'...", config.Listen))
	err = http.ListenAndServe(config.Listen, nil)
	if err != nil {
		syslogger.Crit(err.Error())
		os.Exit(1)
	}
}
