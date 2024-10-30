package bus

import (
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// common bus package

type Message struct {
	ID    int
	Topic string
	Type  string
	Data  interface{}

	TimerID   int
	Error     error
	RespondTo int

	OnCancel func(m *Message)
}

var BusTimeout = 60 * time.Second
var BusHardTimeout = 120 * time.Second

var ErrInvalidMessageData = errors.New("invalid message data")

type Subscriber interface {
	Notify(msg Message)
}

type Bus struct {
	Subscribers map[string][]chan *Message //topic -> subscribers
	M           sync.Mutex
	In          chan *Message
	NextID      int
}

var cb *Bus = &Bus{
	Subscribers: make(map[string][]chan *Message),
	In:          make(chan *Message, 1000),
	NextID:      0,
}

func Init() {
	go ProcessMessages()
	go ProcessTimers()
}

func ProcessMessages() {
	for msg := range cb.In {
		cb.M.Lock()
		subs, ok := cb.Subscribers[msg.Topic]
		if ok {
			for _, subscriber := range subs {
				subscriber <- msg
			}
		}
		cb.M.Unlock()
	}
}

func Subscribe(topic ...string) chan *Message {
	cb.M.Lock()
	defer cb.M.Unlock()

	ch := make(chan *Message, 1000)

	added := make(map[string]bool)

	for _, t := range topic {

		if _, ok := added[t]; ok { // prevent duplicate subscriptions
			continue
		}
		added[t] = true

		subs, ok := cb.Subscribers[t]
		if !ok {
			subs = make([]chan *Message, 0)
		}

		subs = append(subs, ch)
		cb.Subscribers[t] = subs
	}

	return ch
}

func Unsubscribe(ch chan *Message) {
	cb.M.Lock()
	defer cb.M.Unlock()

	for t, subs := range cb.Subscribers {
		for i := 0; i < len(subs); i++ {
			if subs[i] == ch {
				subs = append(subs[:i], subs[i+1:]...)
				i-- // Adjust index after removal
			}
		}
		if len(subs) == 0 {
			delete(cb.Subscribers, t)
		} else {
			cb.Subscribers[t] = subs
		}
	}

	close(ch)
}

func SendEx(topic, t string, data interface{}, timer_id int, respond_to int, err error) *Message {
	cb.M.Lock()
	defer cb.M.Unlock()

	cb.NextID++

	msg := &Message{
		ID:        cb.NextID,
		Topic:     topic,
		Type:      t,
		TimerID:   timer_id,
		Data:      data,
		Error:     err,
		RespondTo: respond_to}

	cb.In <- msg

	// if t != "tick" && t != "tick-10sec" && t != "tick-min" && t != "done" {
	// 	if respond_to != 0 {
	// 		log.Trace().Msgf("   %04d->%s: %s timer:%d respond to: %d, error: %v", cb.NextID, topic, t, timer_id, respond_to, err)
	// 	} else {
	// 		log.Trace().Msgf("   %04d->%s: %s timer:%d", cb.NextID, topic, t, timer_id)
	// 	}
	// }

	return msg
}

func Send(topic, t string, data interface{}) *Message {
	return SendEx(topic, t, data, 0, 0, nil)
}

func (m *Message) Respond(data interface{}, err error) *Message {
	return SendEx(m.Topic, m.Type+"_response", data, 0, m.ID, err)
}

// Chain fetch (on the same timer)
func (m *Message) Fetch(topic, t string, data interface{}) *Message {
	if m != nil {
		// log.Debug().Msgf("   CHAIN Fetch %d -> (%s/%s) %d", m.ID, topic, t, m.TimerID)
		return FetchEx(topic, t, data, m.TimerID, BusTimeout, BusHardTimeout)
	} else {
		return FetchEx(topic, t, data, 0, BusTimeout, BusHardTimeout)
	}
}

func Fetch(topic, t string, data interface{}) *Message {
	return FetchEx(topic, t, data,
		0,
		BusTimeout,
		BusHardTimeout)
}

func FetchEx(topic, t string, data interface{}, timer_id int, limit time.Duration, hardlimit time.Duration) *Message {
	var ch chan *Message
	if topic != "timer" {
		ch = Subscribe(topic, "timer")
	} else {
		ch = Subscribe("timer")
	}
	defer Unsubscribe(ch)

	if timer_id == 0 {
		m := Send("timer", "init", &B_TimerInit{
			Limit:     limit,
			HardLimit: hardlimit,
			Start:     true,
		})
		timer_id = m.ID
	} else {
		res := Fetch("timer", "init-hard", &B_TimerInitHard{
			TimerId:   timer_id,
			Limit:     limit,
			HardLimit: hardlimit,
			Start:     true,
		})
		if res.Error != nil {
			log.Error().Msgf("Error fetching timer init-hard: %v", res.Error)
			return res
		}
	}

	m := SendEx(topic, t, data, timer_id, 0, nil)
	// log.Trace().Msgf("   FETCH %04d->%s: %s timer_id: %d", id, topic, t, timer_id)

	for msg := range ch {
		if msg.Topic == topic && msg.RespondTo == m.ID {
			return msg
		}

		if msg.Topic == "timer" && msg.Type == "done" {
			if id, ok := msg.Data.(int); ok && id == timer_id {

				if m.OnCancel != nil {
					m.OnCancel(m)
					m.OnCancel = nil // prevent double cancel
				}
				break
			}
		}
	}

	return &Message{Error: errors.New("timeout")}
}

func (m *Message) Cancel() {
	if m.OnCancel != nil {
		m.OnCancel(m)
	}

	m.Respond(nil, errors.New("cancelled"))
}
