package main

import (
	"fmt"
	"os"
	"syscall"
)

var sigchan = make(chan os.Signal, 1)

func signals() {
	for {
		s := <-sigchan
		switch s {
		case syscall.SIGTERM:
			fallthrough
		case syscall.SIGINT:
			fallthrough
		case syscall.SIGQUIT:
			fallthrough
		case syscall.SIGHUP:
			fallthrough
		case syscall.SIGKILL:
			fmt.Println("got signal: ", s.String())
			stopServer()
			return
		}
	}
}
