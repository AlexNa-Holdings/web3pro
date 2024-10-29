package bus

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type BusTimer struct {
	Limit     time.Duration
	HardLimit time.Duration

	paused   bool
	lapsed   time.Duration
	starTime time.Time
}

var timers = make(map[int]*BusTimer)
var mu = &sync.Mutex{}
var nextCheckTimer *time.Timer = time.NewTimer(0)
var tick_timer *time.Ticker = time.NewTicker(1 * time.Second)
var tick = 0

func GetTimeLeft(id int) time.Duration {
	mu.Lock()
	defer mu.Unlock()

	t, ok := timers[id]
	if !ok {
		return 0
	}

	if t.paused {
		return t.Limit - t.lapsed
	}

	now := time.Now()
	lapsed := t.lapsed + now.Sub(t.starTime)
	return t.Limit - lapsed
}

func ProcessTimers() {
	ch := Subscribe("timer")

	for {
		select {
		case msg := <-ch:
			if msg.RespondTo != 0 {
				continue // ignore responses
			}
			switch msg.Type {
			case "init":
				d, ok := msg.Data.(*B_TimerInit)
				if ok {
					err := timer_init(msg.ID, d)
					if err != nil {
						msg.Respond("ERROR", err)
					} else {
						msg.Respond("OK", nil)
					}
				} else {
					log.Error().Msg("Invalid timer init data")
					msg.Respond("ERROR", errors.New("invalid timer init data"))
				}
			case "init-hard":
				d, ok := msg.Data.(*B_TimerInitHard)
				if ok {
					err := timer_init_hard(d)
					if err != nil {
						msg.Respond("ERROR", err)
					} else {
						msg.Respond("OK", nil)
					}
				} else {
					log.Error().Msg("Invalid timer init data")
					msg.Respond("ERROR", errors.New("invalid timer init data"))
				}
			case "start", "resume":
				id, ok := msg.Data.(int)
				if ok {
					err := timer_start(id)
					if err != nil {
						msg.Respond("ERROR", err)
					} else {
						msg.Respond("OK", nil)
					}
				} else {
					log.Error().Msg("Invalid timer start data")
					msg.Respond("ERROR", errors.New("invalid timer start data"))
				}
			case "reset":
				id, ok := msg.Data.(int)
				if ok {
					timer_reset(id)
					msg.Respond("OK", nil)
				} else {
					log.Error().Msg("Invalid timer reset data")
					msg.Respond("ERROR", errors.New("invalid timer reset data"))
				}
			case "pause":
				id, ok := msg.Data.(int)
				if ok {
					timer_pause(id)
					msg.Respond("OK", nil)
				} else {
					log.Error().Msg("Invalid timer pause data")
					msg.Respond("ERROR", errors.New("invalid timer pause data"))
				}
			case "delete":
				id, ok := msg.Data.(int)
				if ok {
					delete(timers, id)
					msg.Respond("OK", nil)
				} else {
					log.Error().Msg("Invalid timer delete data")
					msg.Respond("ERROR", errors.New("invalid timer delete data"))
				}
			case "trigger":
				id, ok := msg.Data.(int)
				if ok {
					mu.Lock()
					t, ok := timers[id]
					if !ok {
						log.Error().Msgf("Timer:trigger %d does not exist", id)
						msg.Respond("ERROR", errors.New("timer does not exist"))
					} else {
						t.lapsed = t.Limit
						t.paused = false
						msg.Respond("OK", nil)
					}
					mu.Unlock()
				} else {
					log.Error().Msg("Invalid timer trigger data")
					msg.Respond("ERROR", errors.New("invalid timer trigger data"))
				}
			case "left":
				id, ok := msg.Data.(int)
				if ok {
					msg.Respond(GetTimeLeft(id), nil)
				} else {
					log.Error().Msg("Invalid timer left data")
					msg.Respond("ERROR", errors.New("invalid timer left data"))
				}
			case "tick", "tick-10sec", "tick-min":
				continue
			case "done":
				continue
			default:
				log.Error().Msgf("Invalid timer message type %s", msg.Type)
				msg.Respond("ERROR", errors.New("invalid timer message type"))
			}
			updateTimers()
		case <-nextCheckTimer.C:
			updateTimers()
		case <-tick_timer.C:
			left := make(map[int]time.Duration)
			tick++

			mu.Lock()
			for id, t := range timers {
				if t.paused {
					left[id] = t.Limit - t.lapsed
				} else {
					now := time.Now()
					left[id] = t.Limit - (t.lapsed + now.Sub(t.starTime))
				}
			}
			mu.Unlock()

			Send("timer", "tick", &B_TimerTick{
				Tick: tick,
				Left: left,
			})

			if tick%10 == 0 {
				Send("timer", "tick-10sec", &B_TimerTick{
					Tick: tick,
					Left: left,
				})
			}

			if tick%60 == 0 {
				Send("timer", "tick-min", &B_TimerTick{
					Tick: tick,
					Left: left,
				})
			}

			tick_timer.Reset(1 * time.Second)
		}
	}
}

func updateTimers() {
	mu.Lock()
	defer mu.Unlock()

	// reset the timer
	if nextCheckTimer != nil {
		nextCheckTimer.Stop()
	}

	update_after := 60 * time.Minute // 1 hour
	timer_needed := false

	for id, t := range timers {
		if !t.paused {
			l := t.lapsed + time.Since(t.starTime)
			if l >= t.Limit {
				Send("timer", "done", id)
				delete(timers, id) // remove timer
				continue
			}
			fires_in := t.Limit - l
			if update_after > fires_in {
				update_after = fires_in
			}

			timer_needed = true
		}
	}

	if timer_needed {
		nextCheckTimer = time.NewTimer(update_after)
	}

}

func timer_init(id int, d *B_TimerInit) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := timers[id]; ok {
		log.Warn().Msgf("Timer %d already exists", id)
		return errors.New("timer already exists")
	}

	if d.Limit > d.HardLimit {
		log.Warn().Msgf("Timer %d has a limit greater than the hard limit", id)
		return errors.New("limit greater than hard limit")
	}

	timers[id] = &BusTimer{
		paused:    !d.Start,
		Limit:     d.Limit,
		HardLimit: d.HardLimit,
		lapsed:    0,
		starTime:  time.Now(),
	}
	return nil
}

func timer_init_hard(d *B_TimerInitHard) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := timers[d.TimerId]; !ok {
		log.Warn().Msgf("Timer %d does not exist", d.TimerId)
		return errors.New("timer does not exist")
	}

	timers[d.TimerId] = &BusTimer{
		paused:    false,
		Limit:     d.Limit,
		HardLimit: d.HardLimit,
		lapsed:    0,
		starTime:  time.Now(),
	}
	return nil
}

func timer_start(id int) error {
	mu.Lock()
	defer mu.Unlock()

	t, ok := timers[id]
	if !ok {
		log.Error().Msgf("Timer %d does not exist", id)
		return errors.New("timer does not exist")
	}

	if !t.paused {
		log.Warn().Msgf("Timer %d is already running", id)
		return errors.New("timer is already running")
	}

	if t.lapsed >= t.Limit {
		log.Warn().Msgf("Timer %d has reached its limit", id)
		return errors.New("timer has reached its limit")
	}

	t.paused = false
	t.starTime = time.Now()

	return nil
}

func timer_pause(id int) {
	mu.Lock()
	defer mu.Unlock()

	t, ok := timers[id]
	if !ok {
		log.Error().Msgf("Timer %d does not exist", id)
		return
	}

	if t.paused {
		log.Warn().Msgf("Timer %d is already paused", id)
		return
	}

	t.paused = true
	t.lapsed += time.Since(t.starTime)
}

func timer_reset(id int) {
	mu.Lock()
	defer mu.Unlock()

	t, ok := timers[id]
	if !ok {
		log.Error().Msgf("Timer %d does not exist", id)
		return
	}

	l := t.lapsed
	if !t.paused {
		l += time.Since(t.starTime)
	}

	if l >= t.HardLimit {
		log.Warn().Msgf("Timer %d has reached its hard limit", id)
		return
	}

	t.lapsed = 0
	t.HardLimit -= l
	if t.Limit > t.HardLimit {
		t.Limit = t.HardLimit
	}
	t.starTime = time.Now()

	// reset the timer
	if nextCheckTimer != nil {
		nextCheckTimer.Reset(0)
	} else {
		nextCheckTimer = time.NewTimer(0)
	}
}

func TimerLoop(seconds int, every int, cancel_timer int, f func() (any, error, bool)) (any, error) {

	start := time.Now()
	duration := 60 * time.Second

	ch := Subscribe("timer")
	defer Unsubscribe(ch)

	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}

		switch msg.Type {
		case "tick":
			if time.Now().After(start.Add(duration)) {
				break
			}

			if tick%every == 0 {
				data, err, done := f()
				if done {
					return data, err
				}
			}
		case "done":
			timer_id, ok := msg.Data.(int)
			if ok && timer_id == cancel_timer {
				return nil, errors.New("timeout")
			}
		}

	}
	return nil, fmt.Errorf("timeout")
}
