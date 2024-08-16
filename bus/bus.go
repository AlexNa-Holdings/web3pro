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
	log.Trace().Msgf("bus.Subscribing to %v", topic)

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
	log.Trace().Msg("bus.Unsubscribing")

	cb.M.Lock()
	defer cb.M.Unlock()

	for t, subs := range cb.Subscribers {
		for i, subscriber := range subs {
			if subscriber == ch {
				subs = append(subs[:i], subs[i+1:]...)
				cb.Subscribers[t] = subs
				break
			}
		}
	}

	close(ch)
}

func SendEx(topic, t string, data interface{}, timer_id int, respond_to int, err error) int {
	if t != "tick" {
		if respond_to != 0 {
			log.Trace().Msgf("   %04d->%s: %s respond to: %d, error: %v", cb.NextID, topic, t, respond_to, err)
		} else {
			log.Trace().Msgf("   %04d->%s: %s", cb.NextID, topic, t)
		}
	}

	cb.M.Lock()
	defer cb.M.Unlock()

	cb.NextID++
	cb.In <- &Message{
		ID:        cb.NextID,
		Topic:     topic,
		Type:      t,
		TimerID:   timer_id,
		Data:      data,
		Error:     err,
		RespondTo: respond_to}

	return cb.NextID
}

func Send(topic, t string, data interface{}) int {
	return SendEx(topic, t, data, 0, 0, nil)
}

func (m *Message) Respond(data interface{}, err error) int {
	return SendEx(m.Topic, m.Type+"_response", data, 0, m.ID, err)
}

func Fetch(topic, t string, data interface{}) *Message {
	return FetchEx(topic, t, data,
		BusTimeout,
		BusHardTimeout,
		nil,
		0)
}

func FetchWithHail(topic, t string, data interface{}, hail *B_Hail, hail_delay int) *Message {
	return FetchEx(topic, t, data,
		BusTimeout,
		BusHardTimeout,
		hail,
		hail_delay)
}

func FetchEx(topic, t string, data interface{}, limit time.Duration, hardlimit time.Duration, hail *B_Hail, hail_delay int) *Message {

	if topic == "ui" && hail != nil {
		return &Message{Error: errors.New("cannot fetch 'ui' with hail")}
	}

	ch := Subscribe(topic, "timer", "ui")
	defer Unsubscribe(ch)

	timer_id := Send("timer", "init", &B_TimerInit{
		Limit:     limit,
		HardLimit: hardlimit,
		Start:     true,
	})

	id := SendEx(topic, t, data, timer_id, 0, nil)

	log.Trace().Msgf("   FETCH %04d->%s: %s timer_id: %d", id, topic, t, timer_id)

	timer := time.After(time.Duration(hail_delay) * time.Second)
	for {
		select {
		case <-timer:
			if hail != nil {
				hail.OnCancel = func(m *Message) {
					log.Debug().Msgf("Send 'trigger' to timer:%d", timer_id)
					Send("timer", "trigger", timer_id)
				}
				SendEx("ui", "hail", hail, timer_id, 0, nil)
			}
		case msg := <-ch:
			if msg.Topic == topic {
				if msg.RespondTo == id {
					Send("timer", "delete", timer_id)
					if hail != nil {
						Send("ui", "remove-hail", hail)
					}
					return msg
				}
			}

			if msg.Topic == "timer" && msg.Type == "done" {
				if id, ok := msg.Data.(int); ok {
					if id == timer_id {
						return &Message{Error: errors.New("timeout")}
					}
				}
			}
		}
	}
}
