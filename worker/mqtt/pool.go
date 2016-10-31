package mqtt

import (
	"errors"
	"time"

	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	Pool "github.com/jolestar/go-commons-pool"
	"github.com/golang/glog"
	"fmt"
	"bmw/lib"
)

var (
	poolConfig *Pool.ObjectPoolConfig

	errorPoolObjType error = errors.New("obj in Pool is not *MQTT.Client")
	waitTimeout         = time.Second * 10
)

func init() {
	initConfig()
}

func newPool(host, username, password string) *Pool.ObjectPool {
	mqttPool := Pool.NewObjectPool(&MqttPoolFactory{host, username, password}, poolConfig)
	return mqttPool
}

func initConfig() {
	poolConfig = Pool.NewDefaultPoolConfig()
	poolConfig.TestWhileIdle = true
	poolConfig.TimeBetweenEvictionRunsMillis = 60 * 1000
}

func newMqttClient(host, username, password string) (*MQTT.Client, error) {
	unixNano := time.Now().UnixNano()

	opts := MQTT.NewClientOptions().AddBroker(host)
	opts.SetClientID(fmt.Sprintf("Robot-%d-%s", unixNano, lib.RandomString(6))).SetPassword(password).SetUsername(username)

	c := MQTT.NewClient(opts)
	return c, nil
}

type MqttPoolFactory struct {
	host     string
	username string
	password string
}

func (this *MqttPoolFactory) MakeObject() (*Pool.PooledObject, error) {
	glog.Info("Making Mqtt Connections.....")
	conn, err := newMqttClient(this.host, this.username, this.password)
	obj := Pool.NewPooledObject(conn)

	return obj, err
}

func (f *MqttPoolFactory) DestroyObject(object *Pool.PooledObject) error {
	if conn, ok := object.Object.(*MQTT.Client); ok {
		conn.Disconnect(30)
		conn = nil
	}
	return errorPoolObjType
}

func (f *MqttPoolFactory) ValidateObject(object *Pool.PooledObject) bool {
	if conn, ok := object.Object.(*MQTT.Client); ok {
		glog.Info("Object validated: ok.")
		return conn.IsConnected()
	}

	return false
}

func (f *MqttPoolFactory) ActivateObject(object *Pool.PooledObject) error {
	if conn, ok := object.Object.(*MQTT.Client); ok {
		if !conn.IsConnected() {
			if token := conn.Connect(); token.WaitTimeout(waitTimeout) && token.Error() != nil {
				glog.Error(token.Error())
				return token.Error()
			} else {
				return nil
			}
		}
		return nil
	}

	return errorPoolObjType
}

func (f *MqttPoolFactory) PassivateObject(object *Pool.PooledObject) error {
	return nil
}