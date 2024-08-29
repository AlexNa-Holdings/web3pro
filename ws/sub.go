package ws

import (
	"fmt"
	"sync"
)

type appSubscription struct {
	id     string
	event  string
	params any
}

type subManger struct {
	nextId int
	subs   map[string][]appSubscription //url -> subscriptions
	mutex  sync.Mutex
}

func newSubManager() *subManger {
	return &subManger{
		nextId: 0,
		subs:   make(map[string][]appSubscription),
		mutex:  sync.Mutex{},
	}
}

func (sm *subManger) addSubscription(url string, event string, params any) string {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.nextId++
	s_id := fmt.Sprintf("0x%x", sm.nextId)
	sm.subs[url] = append(sm.subs[url], appSubscription{
		id:     s_id,
		event:  event,
		params: params,
	})

	return s_id
}

func (sm *subManger) removeSubscription(url string, id string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for i, sub := range sm.subs[url] {
		if sub.id == id {
			sm.subs[url] = append(sm.subs[url][:i], sm.subs[url][i+1:]...)
			return
		}
	}
}

func (sm *subManger) getSubsForEvent(url string, event string) []appSubscription {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	var subs []appSubscription
	for _, sub := range sm.subs[url] {
		if sub.event == event {
			subs = append(subs, sub)
		}
	}

	return subs
}
