package api

import (
	"golang.org/x/net/context"
	"net/http"
	"errors"
	"encoding/json"
	"github.com/golang/glog"
)

var (
	BadRequestError = errors.New("bad request")
	TimeoutError = errors.New("timeout")

	OkResposne = map[string]interface{}{"result":"ok"}
)

type ContextHandler interface {
	ServeHTTPContext(context.Context, http.ResponseWriter, *http.Request)
}

type ContextHandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

func (h ContextHandlerFunc) ServeHTTPContext(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
	h(ctx, rw, req)
}

type middleware func(ContextHandler) ContextHandler

type HttpHandler struct {
	Ctx        context.Context
	CtxHandler ContextHandler
}

func (s HttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//do something else.
	s.CtxHandler.ServeHTTPContext(s.Ctx, w, r)
}


// helper function. return error response.
func HandleHttpError(w http.ResponseWriter, err error) {
	if err == BadRequestError {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// helper function.
func WriteResult(wr http.ResponseWriter, r *http.Request, result map[string]interface{}) {
	var byteJson []byte
	var err error

	if byteJson, err = json.Marshal(result); err != nil {
		glog.Errorf("json.Marshal(\"%v\") failed (%v)", result, err)
		return
	}
	wr.Header().Set("Content-Type", "application/json;charset=utf-8")
	if _, err := wr.Write(byteJson); err != nil {
		glog.Errorf("http Write() error(%v)", err)
		return
	}
	glog.Infof("%s path:%s (params:%s, ret:%v)", r.Method, r.URL.Path, r.Form.Encode(), result)
}
