package mqtt

import (
	"bmw"
	"net/http"
	"bmw/api"
	"golang.org/x/net/context"
	"io/ioutil"
	"github.com/golang/glog"
	"time"
)

type MqttAPI struct {
	pusher bmw.Pusher
	config *MqttApiConfig
}

type MqttApiConfig struct {
	Host string
	Port string
}

func NewMqttApi(config *MqttApiConfig, pusher bmw.Pusher) bmw.API {
	return &MqttAPI{
		config: config,
		pusher: pusher,
	}
}

func (mqttApi *MqttAPI) StartServe(host, port string) error {
	chain := api.NewChain(logging)
	ctx := context.Background()

	mqttHandler := api.HttpHandler{
		Ctx: ctx,
		CtxHandler: chain.Then(mqttApi.makeMqttHandler()),
	}

	mux := http.NewServeMux()
	mux.Handle("/mqtt", mqttHandler)

	return http.ListenAndServe(":" + mqttApi.config.Port, mux)
}

func logging(next api.ContextHandler) api.ContextHandler {
	var h api.ContextHandlerFunc
	h = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
		start := time.Now()
		next.ServeHTTPContext(ctx, rw, req)

		duration := time.Since(start)
		glog.Info("req time: ", duration)
	}

	return api.ContextHandler(h)
}

func (mqttApi *MqttAPI) makeMqttHandler() api.ContextHandler {
	var h api.ContextHandlerFunc

	h = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			api.HandleHttpError(rw, api.BadRequestError)
		}

		bodyCopy := ioutil.NopCloser(req.Body)
		bodyByte, err := ioutil.ReadAll(bodyCopy)
		if err != nil {
			glog.Error(err)
			api.HandleHttpError(rw, err)
		}

		if err := mqttApi.pusher.Push(bodyByte); err != nil {
			glog.Error(err)
			api.HandleHttpError(rw, err)
		}

		api.WriteResult(rw, req, api.OkResposne)
	}

	return api.ContextHandler(h)
}

