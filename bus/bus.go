package bus

import (
	"errors"
	"sync"
	"time"

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

type Subscriber interface {
	Notify(msg Message)
}

type Bus struct {
	Subscribers map[string][]chan *Message //topic -> subscribers
	M           sync.Mutex
	In          chan *Message
	NextID      int
}

var cb *Bus // common bus

func Init() {
	cb = NewBus()
	go ProcessTimers()
}

func NewBus() *Bus {
	b := &Bus{
		Subscribers: make(map[string][]chan *Message),
		In:          make(chan *Message),
		NextID:      0,
	}

	go func() {
		for msg := range b.In {

			log.Trace().Msgf("bus.Dispatching to %s: %s", msg.Topic, msg.Type)

			b.M.Lock()

			subs, ok := b.Subscribers[msg.Topic]
			if ok {
				log.Trace().Msgf("Total subscribers for %s: %d", msg.Topic, len(subs))
				for _, subscriber := range subs {

					log.Debug().Msgf("channel len: %d, cap: %d", len(subscriber), cap(subscriber))

					if len(subscriber) <= cap(subscriber) {
						subscriber <- msg
					} else {
						log.Error().Msg("subscriber channel full")
					}
				}
			}

			b.M.Unlock()
		}
	}()

	return b
}

func Subscribe(topic ...string) chan *Message {
	log.Trace().Msgf("bus.Subscribing to %v", topic)

	cb.M.Lock()
	defer cb.M.Unlock()

	ch := make(chan *Message)

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

func SendEx(topic, t string, data interface{}, timer_id int) int {
	log.Trace().Msgf("bus.Sending to %s: %s", topic, t)

	cb.M.Lock()
	defer cb.M.Unlock()

	cb.NextID++
	cb.In <- &Message{
		ID:      cb.NextID,
		Topic:   topic,
		Type:    t,
		TimerID: timer_id,
		Data:    data}
	return cb.NextID
}

func Send(topic, t string, data interface{}) int {
	return SendEx(topic, t, data, 0)
}

func (m *Message) Respond(data interface{}, err error) int {
	log.Trace().Msgf("bus.Responding to %d: %s", m.ID, data)

	cb.M.Lock()
	defer cb.M.Unlock()

	cb.NextID++
	cb.In <- &Message{
		ID:        cb.NextID,
		Topic:     m.Topic,
		RespondTo: m.ID,
		Data:      data,
		Error:     err}
	return cb.NextID
}

func Fetch(topic, t string, data interface{}) *Message {
	return FetchEx(topic, t, data,
		time.Duration(cmn.Config.BusTimeout),
		time.Duration(cmn.Config.BusHardTimeout))
}

func FetchEx(topic, t string, data interface{}, limit time.Duration, hardlimit time.Duration) *Message {
	timer_id := Send("timer", "init", &BM_TimerInit{
		LimitSeconds:     int(limit.Seconds()),
		HardLimitSeconds: int(hardlimit.Seconds()),
	})

	id := SendEx(topic, t, data, timer_id)

	ch := Subscribe(topic, "time")
	defer Unsubscribe(ch)

	select {
	case response := <-ch:
		if response.RespondTo != id {
			return response
		}
	case <-time.After(limit):
		break
	}

	return &Message{Error: errors.New("timeout")}
}
