package main

import (
	"bmw/lib"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
)

var (
	userStore  *lib.BoltStorage
	bucketName = []byte("fame_user")
)

func init() {
	openAndRestore()
	go startSignal()
}

func openAndRestore() {
	store, err := lib.OpenBoltStorage("users.db", bucketName)
	if err != nil {
		panic(err)
	}

	userStore = store

	restore()
}

func close() {
	userStore.Close()
}

func restore() {
	if userStore != nil {
		userStore.ForEach(bucketName, func(k, v []byte) error {
			UID := string(k)
			user := new(User)

			if err := json.Unmarshal(v, user); err != nil {
				glog.Error(err)
				return err
			}

			usersContainer.setUser(UID, user)

			types := user.getBoundAppTypes()
			for _, value := range types {
				typeUIDMap.appendUID(value, UID)
			}

			return nil
		})
	}
}

func storeUser(user *User) {
	user.bindLock.Lock()
	defer user.bindLock.Unlock()

	key := []byte(user.UID)
	value, err := json.Marshal(user)
	if err != nil {
		glog.Error(err)
	}

	//glog.Infof("user_json=%s", value)

	err = userStore.Set(bucketName, key, []byte(value))
	if err != nil {
		glog.Error(err)
	}
}

// startSignal register signals handler.
func startSignal() {
	var (
		c chan os.Signal
		s os.Signal
	)
	c = make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM,
		syscall.SIGINT, syscall.SIGSTOP)
	// Block until a signal is received.
	for {
		s = <-c
		glog.Infof("get a signal %s", s.String())
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGINT:
			close()
			os.Exit(0)
			return
		case syscall.SIGHUP:
			// TODO reload
			//return
			close()
			openAndRestore()
		default:
			return
		}
	}
}
