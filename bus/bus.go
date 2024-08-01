package bus

import (
	"errors"
	"sync"

	"github.com/AlexNa-Holdings/web3pro/cmn"
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
	if topic != "timer" {
		log.Trace().Msgf("bus.Sending to %s: %s", topic, t)
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
		cmn.Config.BusTimeout,
		cmn.Config.BusHardTimeout)
}

func FetchEx(topic, t string, data interface{}, limit int, hardlimit int) *Message {

	if topic == "timer" {
		return &Message{Error: errors.New("invalid topic to fetch")}
	}

	ch := Subscribe(topic, "timer")
	defer Unsubscribe(ch)

	timer_id := Send("timer", "init", &B_TimerInit{
		LimitSeconds:     limit,
		HardLimitSeconds: hardlimit,
		Start:            true,
	})

	id := SendEx(topic, t, data, timer_id, 0, nil)

	for msg := range ch {

		log.Debug().Msgf("bus.Fetch: received %s/%s (RespondTo: %d)", msg.Topic, msg.Type, msg.RespondTo)

		switch msg.Topic {
		case topic:
			if msg.RespondTo == id {
				log.Trace().Msgf("bus.Fetch: received response for %s", t)

				Send("timer", "delete", &B_TimerDelete{ID: timer_id})
				return msg
			}
		case "timer":
			if d, ok := msg.Data.(*B_TimerDone); ok {
				if d.ID == timer_id {
					return &Message{Error: errors.New("timeout")}
				}
			}

		}
	}

	return &Message{Error: errors.New("fetch error")}
}
