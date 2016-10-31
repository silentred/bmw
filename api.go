package bmw

import (
	"bmw/lib"
	"encoding/json"
	"io/ioutil"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

// RetryJob represents a job pushed into queue
type RetryJob struct {
	ID      string `json:"id"`
	Retry   uint8  `json:"retry"`
	Payload []byte `json:"payload"`
}

// DefaultAPI receives payload only from request body then push it into queue
type DefaultAPI struct {
	host    string
	engine  *gin.Engine
	routes  []*RouteConfig
	jobPool sync.Pool
}

type RouteConfig struct {
	path   string
	pusher Pusher
}

func NewDefaultAPI(host string, routes []*RouteConfig) *DefaultAPI {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	jobPool := sync.Pool{
		New: func() interface{} {
			return new(RetryJob)
		},
	}

	api := &DefaultAPI{
		host:    host,
		engine:  engine,
		routes:  routes,
		jobPool: jobPool,
	}
	return api
}

func (api *DefaultAPI) register(uri string, pusher Pusher) {
	var handler gin.HandlerFunc
	handler = func(c *gin.Context) {
		b, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			glog.Error(err)
		}

		retryJob := api.jobPool.Get().(*RetryJob)

		id := lib.RandomString(32)
		retryJob.ID = id
		retryJob.Retry = 0
		retryJob.Payload = b

		var jobByte []byte
		if jobByte, err = json.Marshal(retryJob); err != nil {
			glog.Error(err)
			c.JSON(200, gin.H{"error": err.Error()})
			return
		}

		api.jobPool.Put(retryJob)

		if err := pusher.Push(jobByte); err != nil {
			glog.Error(err)
			c.JSON(200, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"id": id})
	}

	api.engine.POST(uri, handler)
}

func (api *DefaultAPI) StartServe() error {
	for _, item := range api.routes {
		api.register(item.path, item.pusher)
	}

	return api.engine.Run(api.host)
}

// GetEngine returns the gin.Engine, so extra path handler can be added
func (api *DefaultAPI) GetEngine() *gin.Engine {
	return api.engine
}
