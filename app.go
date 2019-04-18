package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	branch   = "development"
	revision = "latest"

	readyAfter = flag.Duration("readyAfter", 0, "Duration it takes after this service reports it is ready.")
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
	if *readyAfter > 0 {
		log.Printf("Waiting for %v to be ready...", *readyAfter)
		time.Sleep(*readyAfter)
	}
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
	switch req.URL.Path {
	case "/healthz":
		handleHealth(resp, req)
	default:
		handleEveryThingElse(resp, req)
	}
}

func methodNotAllowed(resp http.ResponseWriter) {
	http.Error(resp, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}

func handleHealth(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		methodNotAllowed(resp)
		return
	}
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

func handleEveryThingElse(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "application/json")
	plainStatusCode := req.URL.Query().Get("statusCode")
	if statusCode, err := strconv.Atoi(plainStatusCode); err == nil && statusCode >= 100 && statusCode < 1000 {
		resp.WriteHeader(statusCode)
	}
	enc := json.NewEncoder(resp)
	enc.SetIndent("", "  ")
	body := responseBodyFor(req)
	if err := enc.Encode(body); err != nil {
		log.Printf("ERROR writing response to %v: %v", req.RemoteAddr, err)
	}
}

func responseBodyFor(req *http.Request) (result responseBody) {
	result.Runtime.Branch = branch
	result.Runtime.Revision = revision
	result.Runtime.Platform = runtime.GOOS + "-" + runtime.GOARCH

	result.Request.Proto = req.Proto
	result.Request.Host = req.Host
	result.Request.Method = req.Method
	result.Request.RequestURI = req.RequestURI
	result.Request.Headers = req.Header
	result.Request.Form = req.Form
	result.Request.PostForm = req.PostForm
	return
}

type responseBody struct {
	Runtime runtimeBody `json:"runtime"`
	Request requestBody `json:"request"`
}

type runtimeBody struct {
	Branch   string `json:"branch"`
	Revision string `json:"revision"`
	Platform string `json:"platform"`
}

type requestBody struct {
	Proto      string              `json:"proto"`
	Host       string              `json:"host"`
	Method     string              `json:"method"`
	RequestURI string              `json:"requestURI"`
	Headers    map[string][]string `json:"headers,omitempty"`
	Form       map[string][]string `json:"form,omitempty"`
	PostForm   map[string][]string `json:"postForm,omitempty"`
}
