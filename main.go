package main

import (
	"context"
	"errors"
	"fmt"
	"jack-henry-project/weatherserver"
	"net"
	"net/http"
	"os/signal"
	"sync"
)

const (
	bindAddress = "localhost:8080"
)

func main() {

	// A few shortcuts I'll take right away for this exercise:
	//
	//  1. I won't worry about any kind of configuration or logging/tracing.
	//     I'll hardcode the IP/port and use fmt.Printf for everything.
	//
	//  2. I won't worry about any daemon compatibility (eg, systemd). I'll just
	//     start the server in the main function, and stop it on sigint. Also
	//     not setting any exit codes.

	masterContext, masterCancel := context.WithCancel(context.Background())

	defer masterCancel()

	go signals()
	signal.Notify(sigchan)

	// just putting this here for easy testing.
	fmt.Printf("Use SIGINT/Ctrl-C to stop.\nFor example, run the following for Kansas City weather\n\tcurl '%s/weather?lat=39.0997&lon=-94.5786'\n", bindAddress)

	if err := startServer(masterContext); err != nil {
		fmt.Printf("Error with server (shutdown): %v\n", err)
	}
	return
}

var listener net.Listener
var listnerMutext sync.Mutex

// startServer starts the HTTP server and blocks until it's done. This is safe
// to call concurrently with stopServer.
func startServer(ctx context.Context) (err error) {

	listnerMutext.Lock()

	if listener != nil {
		listnerMutext.Unlock()
		return fmt.Errorf("server already started")
	}

	// shortcut: in big apps I prefer net.ListenConfig, so I can use
	// ctx in resolving the connection. But we'll just use net.Listen directly.

	fmt.Printf("Listening for connections at %s...\n", bindAddress)
	if listener, err = net.Listen("tcp", bindAddress); err != nil {
		return
	}

	listnerMutext.Unlock()

	defer func() {
		_ = listener.Close()
	}()

	// note: Typically I'd configure much more than this. But we'll keep it
	// simple with all the default values.
	httpServer := &http.Server{
		Handler: &weatherserver.Server{
			Client: &http.Client{},
		},
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}

	// another shortcut, exclusion of TLS. Usually I want to make sure http/2
	// works and disabled unencrypted connections completely. But we'll keep it
	// simple so we can avoid moving certificates around.
	//httpServer.ServeTLS(...)
	err = httpServer.Serve(listener)

	if errors.Is(err, net.ErrClosed) {
		// this is the expected error when the server is stopped, so nil it out
		// as to not return a false positive.
		err = nil
	}

	return
}

// stopServer stops the HTTP server if there is one. Safe to call concurrently
// with startServer.
func stopServer() {
	listnerMutext.Lock()
	if listener == nil {
		return
	}
	_ = listener.Close()
	listener = nil
	listnerMutext.Unlock()
}
