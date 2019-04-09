package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	branch   = "development"
	revision = "latest"

	readyAfter = flag.Duration("readyAfter", 15*time.Second, "Duration it takes after this service reports it is ready.")
	exitAfter  = flag.Duration("exitAfter", 0, "Duration it takes after this service dies with defined exit code."+
		" This time starts after the service is ready. 0 == never exits.")
	exitCode = flag.Int("exitCode", 1, "Code which will be used if this service exits after the defined duration.")
	listen   = flag.String("listen", ":8080", "Where to listen with the health endpoint to.")

	ready = new(atomic.Value)
)

func init() {
	ready.Store(false)
}

func main() {
	log.Printf("kubor-demo1 (branch=%s, revision=%s) is starting...", branch, revision)
	flag.Parse()

	registerGracefulShutdown()
	go runServer()
	waitToBeReady()
	justRun()
	log.Printf("Good bye...")
	os.Exit(*exitCode)
}

func registerGracefulShutdown() {
	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	go func() {
		sig := <-gracefulStop
		log.Printf("Received %v signal. Bye!", sig)
		os.Exit(0)
	}()
}

func runServer() {
	log.Printf("Listen to %s...", *listen)
	if err := http.ListenAndServe(*listen, http.HandlerFunc(handler)); err != nil {
		log.Fatalf("Cannot listen to %s: %v", *listen, err)
	}
}

func waitToBeReady() {
	log.Printf("Waiting for %v to be ready...", *readyAfter)
	time.Sleep(*readyAfter)
	ready.Store(true)
}

func justRun() {
	if exitAfter == nil || *exitAfter == 0 {
		log.Printf("Running for ever...")
		blockForEver()
		return
	}
	log.Printf("Running for %v...", *exitAfter)
	time.Sleep(*exitAfter)
}

func blockForEver() {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	wg.Wait()
}

func handler(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(resp, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	switch req.URL.Path {
	case "/":
		handleRoot(resp, req)
	case "/healthz":
		handleHealth(resp, req)
	default:
		http.NotFound(resp, req)
	}
}

func handleHealth(resp http.ResponseWriter, req *http.Request) {
	r := ready.Load().(bool)
	var v string
	if r {
		resp.WriteHeader(http.StatusOK)
		v = "OK"
	} else {
		resp.WriteHeader(http.StatusServiceUnavailable)
		v = "NOT_READY"
	}
	resp.Header().Set("Content-Type", "text/plain")
	if _, err := fmt.Fprintf(resp, `%s`, v); err != nil {
		log.Printf("ERROR writing response (%s) to %v: %v", v, req.RemoteAddr, err)
	}
}

func handleRoot(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "text/plain")
	if _, err := resp.Write([]byte("Hello world!")); err != nil {
		log.Printf("ERROR writing response to %v: %v", req.RemoteAddr, err)
	}
}
