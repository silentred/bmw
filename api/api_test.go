package api

import (
	"testing"
	"net/http"
	"fmt"
	"golang.org/x/net/context"
	"io/ioutil"
	"time"
)

func TestAPI(t *testing.T) {
	chain := NewChain(logging)
	//chain := NewChain()

	ctx := context.Background()
	helloHandler := HttpHandler{ctx, chain.Then(ContextHandler(ContextHandlerFunc(hello)))}
	sayhiHandler := HttpHandler{ctx, chain.Then(ContextHandler(ContextHandlerFunc(sayhi)))}

	mux := http.NewServeMux()
	mux.Handle("/", helloHandler)
	mux.Handle("/hi", sayhiHandler)

	http.ListenAndServe(":6061", mux)
}

func middlwareA(next ContextHandler) ContextHandler {
	var h ContextHandlerFunc
	h = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
		fmt.Println("middleware A. Before")
		next.ServeHTTPContext(ctx, rw, req)
		fmt.Println("middleware A. After")
	}

	return ContextHandler(h)
}

func logging(next ContextHandler) ContextHandler {
	var h ContextHandlerFunc
	h = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
		start := time.Now()
		next.ServeHTTPContext(ctx, rw, req)

		duration := time.Since(start)
		fmt.Println("req time: ", duration)
	}

	return ContextHandler(h)
}

func hello(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
	//fmt.Fprint(rw, "hello world")
	body, _ := ioutil.ReadAll(req.Body)
	fmt.Println("body is ", string(body))

	req.ParseForm()
	fmt.Println("form is ", req.Form)
}

func sayhi(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
	fmt.Println("inner before")
	fmt.Fprint(rw, "Hi!!!!!")
	fmt.Println("inner after")
}