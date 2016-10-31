package main

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/golang/glog"
)

type UserFriend struct {
	TpName string `json:"tp_name"`
	TpID   string `json:"tp_id"`
	// UID is empty at first
	UID string `json:"uid"`
}

// User represents a user in main app, he has multiple bind-accounts and many friends in other apps
type User struct {
	UID string `json:"uid"`
	// tp_type => ty_id
	BindApp  map[string]string `json:"bind_app"`
	bindLock sync.RWMutex
	// tp_type => []UserFriend
	Friends     map[string][]UserFriend `json:"friends"`
	friendsLock sync.RWMutex
}

// FriendResult is only used for storing final result
type FriendResult struct {
	UID    string `json:"uid"`
	TpName string `json:"tp_name"`
}

// InputRequest represents an input
type InputRequest struct {
	UID     string        `json:"uid"`
	Type    string        `json:"type"`
	TpID    string        `json:"tp_id"`
	TpName  string        `json:"tp_name"`
	Friends []InputFriend `json:"friends"`
}

// InputFriend is a temp struct in the request
type InputFriend struct {
	TpName string `json:"tp_name"`
	TpID   string `json:"tp_id"`
}

type checkFriendsJob struct {
	user   *User
	other  *User
	tpType string
}

// UnbindRequest is the request to remove user.BindApp[type] and TypeUIDMap[type][uid]
type UnbindRequest struct {
	UID  string `json:"uid"`
	Type string `json:"type"`
}

var (
	validTypes = []string{"fb", "phone"}
	// type => []UID， app_type => [user_uid, ...]; 某个type下有哪些 UID; for store
	typeUIDMap = &TypeUIDMap{typeUID: make(map[string][]string)}

	// UID => User
	usersContainer = &UsersContainer{users: make(map[string]*User)}
	createUserLock = sync.Mutex{}

	checkFriendsChan = make(chan checkFriendsJob, 1000)
	// uid => [{uid, tp_name}...]; 关系结果，某个UID关联的账号，每个账号包含 TpName, UID
	// relationResult = make(map[string][]FriendResult)

	// 收到input, 包含 {uid, type, tp_id, friends:[ {tp_id, tp_name}, ...]}
	// 查看是否有这个user, 没有的话， 新建一个 {uid, bindapp[type]:tp_id, friends:[] }
	// 如果有, 检查是否有 user.bindapp[type], 更新 bindapp[type] = tp_id
	// 查看 typeUidMap[type] 中是否有 UID, 没有则添加进去

	// 遍历 typeUidMap[type]的UID，取出 user.BindApp[type] 的tp_id，看 input.friends 中是否相等,
	// 如果相等，创建一个 UserFriend{tp_id, tp_name, uid}, 放入 user.Friends[type]

)

func decodeInput(payload []byte) *InputRequest {
	inputReq := new(InputRequest)
	if err := json.Unmarshal(payload, inputReq); err != nil {
		glog.Error(err)
		return nil
	}

	if !isValidType(inputReq.Type) {
		return nil
	}

	return inputReq
}

func isValidType(tpType string) bool {

	for _, item := range validTypes {
		if item == tpType {
			return true
		}
	}

	glog.Errorf("type %s is invalid", tpType)
	return false
}

func (user *User) getTpID(tpType string) string {
	user.bindLock.RLock()
	defer user.bindLock.RUnlock()

	if val, ok := user.BindApp[tpType]; ok {
		return val
	}

	return ""
}

func (user *User) getBoundAppTypes() []string {
	var types []string

	user.bindLock.RLock()
	for tpType := range user.BindApp {
		types = append(types, tpType)
	}
	user.bindLock.RUnlock()

	return types
}

func (user *User) setTpID(tpType, tpID string) {
	user.bindLock.Lock()
	user.BindApp[tpType] = tpID
	user.bindLock.Unlock()
}

func (user *User) removeTpID(tpType string) {
	user.bindLock.Lock()
	delete(user.BindApp, tpType)
	user.bindLock.Unlock()
}

func (user *User) getFriendsByType(tpType string) []UserFriend {
	user.friendsLock.RLock()

	if friends, ok := user.Friends[tpType]; ok {
		user.friendsLock.RUnlock()

		return friends
	} else {
		user.friendsLock.RUnlock()

		user.friendsLock.Lock()
		user.Friends[tpType] = make([]UserFriend, 0)
		user.friendsLock.Unlock()
		return user.Friends[tpType]
	}
}

func (user *User) appendFriends(tpType string, newFriends []UserFriend) {
	oldFriends := user.getFriendsByType(tpType)

	user.friendsLock.Lock()
	user.Friends[tpType] = append(oldFriends, newFriends...)
	user.friendsLock.Unlock()
}

func (user *User) setFriends(tpType string, friends []UserFriend) {
	user.friendsLock.Lock()
	user.Friends[tpType] = friends
	user.friendsLock.Unlock()
}

func (user *User) remvoeFriendsByType(tpType string) {
	user.friendsLock.Lock()
	delete(user.Friends, tpType)
	user.friendsLock.Unlock()
}

func (user *User) handleUnbind(req *UnbindRequest) {
	user.removeTpID(req.Type)
	user.remvoeFriendsByType(req.Type)

	typeUIDMap.removeUID(req.Type, req.UID)

	storeUser(user)
}

// update BindApp and Friends; has many write op
func (user *User) handleInput(input *InputRequest) {
	if input.TpID != "" {
		user.setTpID(input.Type, input.TpID)
	}

	// if _, ok := user.Friends[input.Type]; !ok {
	// 	// TODO lock
	// 	user.Friends[input.Type] = make([]UserFriend, 0)
	// }

	//glog.Infof("uid=%s user.Friends[%s] is %+v", user.UID, input.Type, user.getFriendsByType(input.Type))

	readyToMerge := make([]UserFriend, 0, len(input.Friends))
	if len(input.Friends) > 0 {
		for _, item := range input.Friends {
			exists := false
			// check if already exists
			for _, old := range user.getFriendsByType(input.Type) {
				if item.TpID == old.TpID {
					exists = true
					break
				}
			}

			if !exists {
				userFriend := UserFriend{
					TpID:   item.TpID,
					TpName: item.TpName,
				}
				readyToMerge = append(readyToMerge, userFriend)
			}
		}
	}

	//glog.Infof("uid=%s readyToMerge is %+v", user.UID, readyToMerge)

	user.appendFriends(input.Type, readyToMerge)
	//user.Friends[input.Type] = append(user.Friends[input.Type], readyToMerge...)

	user.scanAllFriends(input.Type)

	// store to bolt
	storeUser(user)

}

func (user *User) scanAllFriends(tpType string) {
	var friends []UserFriend
	var backCheckUsers []*User
	userFriends := user.getFriendsByType(tpType)

	user.friendsLock.Lock()
	for _, item := range userFriends {
		// search for existing User whose has same tp_id and tp_name
		var otherUID string
		typeUIDMap.iter(tpType, func(uid string) {

			if user.UID != uid {
				if otherUser := findUser(uid); otherUser != nil {
					id := otherUser.getTpID(tpType)
					//glog.Infof("uid=%s comparing %s and %s", user.UID, id, item.TpID)
					if id == item.TpID {
						otherUID = otherUser.UID
						glog.Infof("type=%s uid=%s found friend=%s", tpType, user.UID, otherUID)
						// reverse check, if otherUser.Friends[type] has
						// otherUser.checkFriend(user, tpType)
						backCheckUsers = append(backCheckUsers, otherUser)
					}
				}
			}
		})

		item.UID = otherUID
		friends = append(friends, item)
	}

	//glog.Infof("uid=%s user.Friends[%s] will be %+v", user.UID, tpType, friends)
	//user.setFriends(tpType, friends)
	user.Friends[tpType] = friends
	user.friendsLock.Unlock()

	for _, otherUser := range backCheckUsers {
		job := checkFriendsJob{otherUser, user, tpType}
		checkFriendsChan <- job
		//otherUser.checkFriend(user, tpType)
	}
}

func (user *User) checkFriend(jerry *User, tpType string) bool {
	var friends []UserFriend
	var isFriend = false

	jerryTpID := jerry.getTpID(tpType)
	userFriends := user.getFriendsByType(tpType)

	user.friendsLock.Lock()
	if len(userFriends) > 0 && jerryTpID != "" {
		friends = make([]UserFriend, 0, len(userFriends))

		for _, item := range userFriends {
			if item.TpID == jerryTpID {
				// if uid has not been set.
				item.UID = jerry.UID
				glog.Infof("[checkFriend] type=%s uid=%s found jerry.UID=%s", tpType, user.UID, jerry.UID)
				isFriend = true
			}
			friends = append(friends, item)
		}
	}

	//user.setFriends(tpType, friends)
	user.Friends[tpType] = friends
	user.friendsLock.Unlock()

	storeUser(user)

	return isFriend
}

func (user *User) getRelation(tpType string) []FriendResult {
	var result = make([]FriendResult, 0)
	for _, val := range user.getFriendsByType(tpType) {
		if val.UID != "" {
			friend := FriendResult{UID: val.UID, TpName: val.TpName}
			result = append(result, friend)
		}
	}

	return result
}

func startCheckFriends() {
	go func() {
		for job := range checkFriendsChan {
			job.user.checkFriend(job.other, job.tpType)
		}
	}()

	// continiously check the relation
	// go func() {
	// 	tick := time.Tick(10 * time.Second)
	// 	tpType := "fb"
	// 	for range tick {
	// 		UIDs := typeUIDMap.getUIDs(tpType)
	// 		for _, uid := range UIDs {

	// 		}
	// 	}
	// }()

}

// startWork
func startWork(payload []byte) {
	req := decodeInput(payload)
	if req == nil {
		glog.Error("req is nil")
		return
	}

	createUserLock.Lock()
	user := findUser(req.UID)
	if user == nil {
		user = createNewUser(req)
	}
	createUserLock.Unlock()

	typeUIDMap.appendUID(req.Type, req.UID)

	// TODO req must not have duplicated req.Friends.TpID
	user.handleInput(req)

}

// ===== unbind =====

func unbindAccount(payload []byte) error {
	req := decodeUnbindReq(payload)
	if req == nil {
		err := fmt.Errorf("req is nil")
		glog.Error("req is nil")
		return err
	}

	user := findUser(req.UID)
	if user == nil {
		err := fmt.Errorf("user(id=%s) not found", req.UID)
		glog.Error(err)
		return err
	}

	user.handleUnbind(req)
	return nil
}

func decodeUnbindReq(payload []byte) *UnbindRequest {
	inputReq := new(UnbindRequest)
	if err := json.Unmarshal(payload, inputReq); err != nil {
		glog.Error(err)
		return nil
	}

	if !isValidType(inputReq.Type) {
		return nil
	}

	return inputReq
}
