package main

import "sync"

type UsersContainer struct {
	sync.RWMutex
	users map[string]*User
}

func (c *UsersContainer) getUser(UID string) *User {
	c.RLock()
	defer c.RUnlock()

	if user, ok := c.users[UID]; ok {
		return user
	}

	return nil
}

func (c *UsersContainer) setUser(UID string, user *User) {
	c.Lock()
	defer c.Unlock()

	c.users[UID] = user
}

type TypeUIDMap struct {
	sync.RWMutex
	typeUID map[string][]string
}

func (m *TypeUIDMap) getUIDs(typeName string) []string {
	m.RLock()

	if IDs, ok := m.typeUID[typeName]; ok {
		m.RUnlock()

		return IDs
	} else {
		m.RUnlock()

		m.Lock()
		m.typeUID[typeName] = make([]string, 0, 10)
		m.Unlock()
	}

	return m.typeUID[typeName]
}

func (m *TypeUIDMap) appendUID(typeName, UID string) {
	in, oldUIDs := inTypeUIDMap(typeName, UID)
	if !in {
		//glog.Infof("appending to map[%s] UID=%s", typeName, UID)
		m.Lock()
		m.typeUID[typeName] = append(oldUIDs, UID)
		m.Unlock()
	}

}

func (m *TypeUIDMap) removeUID(typeName, UID string) {
	in, oldUIDs := inTypeUIDMap(typeName, UID)
	if in {
		m.Lock()
		for i, value := range oldUIDs {
			if value == UID {
				m.typeUID[typeName] = remove(i, oldUIDs)
			}
		}
		m.Unlock()
	}

}

func remove(i int, items []string) []string {
	return append(items[:i], items[i+1:]...)
}

func (m *TypeUIDMap) iter(typeName string, closure func(string)) {
	UIDsCopy := m.getUIDs(typeName)
	for _, uid := range UIDsCopy {
		closure(uid)
	}
}

func inTypeUIDMap(tpType, UID string) (bool, []string) {
	UIDs := typeUIDMap.getUIDs(tpType)

	for _, val := range UIDs {
		if UID == val {
			return true, UIDs
		}
	}

	return false, UIDs
}

func findUser(id string) *User {
	return usersContainer.getUser(id)
}

func createNewUser(input *InputRequest) *User {

	user := &User{
		UID:         input.UID,
		BindApp:     make(map[string]string),
		Friends:     make(map[string][]UserFriend),
		bindLock:    sync.RWMutex{},
		friendsLock: sync.RWMutex{},
	}

	// TODO only need excecute onece
	usersContainer.setUser(user.UID, user)

	return user
}
