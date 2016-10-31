package publisher

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/coreos/etcd/client"
)

var (
	DefaultPrefix string = "being/workers"
)

// EtcdPublisher publish sevice info to etcd
type EtcdPublisher struct {
	// should be in format as being/workers/{ServiceName}
	Prefix string
	TTL    time.Duration
	Client *client.Client
	Kapi   client.KeysAPI
}

// NewEtcdPublisher returns the publisher which refresh every ttl seconds
func NewEtcdPublisher(hosts []string, ttl int) *EtcdPublisher {
	return NewEtcdPublisherWithPrefix(hosts, DefaultPrefix, ttl)
}

func NewEtcdPublisherWithPrefix(hosts []string, prefix string, ttl int) *EtcdPublisher {
	cfg := client.Config{
		Endpoints:               hosts,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}

	cli, err := client.New(cfg)
	if err != nil {
		panic(err)
	}

	kapi := client.NewKeysAPI(cli)

	if len(prefix) == 0 {
		prefix = DefaultPrefix
	}

	return &EtcdPublisher{
		Prefix: prefix,
		TTL:    time.Duration(ttl) * time.Second,
		Client: &cli,
		Kapi:   kapi,
	}
}

// Register stores the info of service at registry, and keep it, refresh it
func (ep *EtcdPublisher) Register(service *Service) error {
	path := ep.getFullPath(service)

	opt := &client.SetOptions{TTL: ep.TTL}
	if service.lastIndex > 0 {
		opt.PrevIndex = service.lastIndex
	}
	resp, err := ep.Kapi.Set(context.Background(), path, service.String(), opt)
	if err != nil {
		log.Println(err)
		return err
	}
	service.SetIndex(resp.Index)

	return nil
}

// Unregister removes the Publisher.FullKey at registry
func (ep *EtcdPublisher) Unregister(service *Service) error {
	path := ep.getFullPath(service)

	opt := &client.DeleteOptions{PrevIndex: service.GetIndex()}
	resp, err := ep.Kapi.Delete(context.Background(), path, opt)
	if err != nil {
		log.Println(err)
		return err
	}

	service.SetIndex(resp.Index)
	service.Stop()

	return nil
}

// Heartbeat blocks and refresh TTL every {ttl} seconds until the service is Unregistered
func (ep *EtcdPublisher) Heartbeat(service *Service) {
	ticker := time.NewTicker(ep.TTL / 2)
	path := ep.getFullPath(service)
	opt := &client.SetOptions{
		Refresh: true,
		TTL:     ep.TTL,
	}

	for range ticker.C {
		select {
		case <-service.quit:
			return
		default:
			resp, err := ep.Kapi.Set(context.Background(), path, service.String(), opt)
			if err != nil {
				log.Println(err)
			}
			if resp.Index > 0 {
				service.SetIndex(resp.Index)
			}
		}
	}
}

func (ep *EtcdPublisher) getFullPath(service *Service) string {
	return fmt.Sprintf("%s/%s/%s", ep.Prefix, service.Name, service.ID)
}
