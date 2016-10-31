## Being Micro Worker


### SingleBMW 
This worker uses Go-Channel as queue, Gin as API engine.

## MultiBMW
This struct could create multiple worker pool on different URIs. Each worker pool has its own message Waiter
and Pusher.

### Usage
See `worker/resource` folder.

```go
func main() {
	flag.Parse()
	loadConfig("./", "config")

	handler := &handler{}
	batchHandler := &batchHandler{}

	routes := []bmw.SingleRoute{
		bmw.SingleRoute{
			Path:    "service/resource/upload",
			Topic:   "upload",
			Handler: handler,
		},
		bmw.SingleRoute{
			Path:    "service/resource/upload/batch",
			Topic:   "upload-batch",
			Handler: batchHandler,
		},
	}
	handlerRoute := new(bmw.MultiHandlerRoutes)
	handlerRoute.Host = listenPort
	handlerRoute.Routes = routes

	goChan := bmw.GoChannelConfig{}
	bmw := bmw.NewMultiBMW(goChan, handlerRoute)

	engine := bmw.GetAPI().GetEngine()
	engine.GET("service/resource/key", findKeyQuery)

	bmw.Start()
}

type handler struct {
}

func (h *handler) Handle(job bmw.RetryJob) error {
	return handleRequest(&job)
}

type batchHandler struct {
}

func (h *batchHandler) Handle(job bmw.RetryJob) error {
	fmt.Println(job.ID)
	return nil
}
```

To send a job: `curl localhost:8085/service/resource/upload/batch -d "test"`