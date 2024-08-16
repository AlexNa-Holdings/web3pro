package ws

import "sync"

type appSubscription struct {
	id     int
	event  string
	params any
}

type subManger struct {
	nexId int
	subs  map[string][]appSubscription //url -> subscriptions
	mutex sync.Mutex
}

func newSubManager() *subManger {
	return &subManger{
		nexId: 0,
		subs:  make(map[string][]appSubscription),
		mutex: sync.Mutex{},
	}
}

func (sm *subManger) addSubscription(url string, event string, params any) int {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.nexId++
	sm.subs[url] = append(sm.subs[url], appSubscription{
		id:     sm.nexId,
		event:  event,
		params: params,
	})

	return sm.nexId
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
