package bus

import (
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type BusTimer struct {
	LimitSeconds     int
	HardLimitSeconds int

	paused        bool
	lapsedSeconds int
	starTime      time.Time
}

var timers = make(map[int]*BusTimer)
var mu = &sync.Mutex{}
var nextCheckTimer *time.Timer = time.NewTimer(0)
var tick_timer *time.Ticker = time.NewTicker(1 * time.Second)
var tick = 0

func GetTimerSecondsLeft(id int) int {
	mu.Lock()
	defer mu.Unlock()

	t, ok := timers[id]
	if !ok {
		return 0
	}

	if t.paused {
		return t.LimitSeconds - t.lapsedSeconds
	}

	now := time.Now()
	lapsed := t.lapsedSeconds + int(now.Sub(t.starTime).Seconds())
	return t.LimitSeconds - lapsed
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
					timer_init(msg.ID, d)
					msg.Respond("OK", nil)
				} else {
					log.Error().Msg("Invalid timer init data")
					msg.Respond("ERROR", errors.New("invalid timer init data"))
				}
			case "start":
				d, ok := msg.Data.(*B_TimerStart)
				if ok {
					timer_start(d.ID)
					msg.Respond("OK", nil)
				} else {
					log.Error().Msg("Invalid timer start data")
					msg.Respond("ERROR", errors.New("invalid timer start data"))
				}
			case "reset":
				d, ok := msg.Data.(*B_TimerReset)
				if ok {
					timer_reset(d.ID)
					msg.Respond("OK", nil)
				} else {
					log.Error().Msg("Invalid timer reset data")
					msg.Respond("ERROR", errors.New("invalid timer reset data"))
				}
			case "pause":
				d, ok := msg.Data.(*B_TimerPause)
				if ok {
					timer_pause(d.ID)
					msg.Respond("OK", nil)
				} else {
					log.Error().Msg("Invalid timer pause data")
					msg.Respond("ERROR", errors.New("invalid timer pause data"))
				}
			case "delete":
				d, ok := msg.Data.(*B_TimerDelete)
				if ok {
					mu.Lock()
					delete(timers, d.ID)
					mu.Unlock()
					msg.Respond("OK", nil)
				} else {
					log.Error().Msg("Invalid timer delete data")
					msg.Respond("ERROR", errors.New("invalid timer delete data"))
				}
			case "trigger":
				d, ok := msg.Data.(*B_TimerTrigger)
				if ok {
					mu.Lock()
					t, ok := timers[d.ID]
					if !ok {
						log.Error().Msgf("Timer %d does not exist", d.ID)
						msg.Respond("ERROR", errors.New("timer does not exist"))
					} else {
						t.lapsedSeconds = t.LimitSeconds
						t.paused = false
						msg.Respond("OK", nil)
					}
					mu.Unlock()
				} else {
					log.Error().Msg("Invalid timer trigger data")
					msg.Respond("ERROR", errors.New("invalid timer trigger data"))
				}
			case "tick":
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
			mu.Lock()
			left := make(map[int]int)
			tick++

			for id, t := range timers {
				if t.paused {
					continue
				}
				now := time.Now()
				lapsed := t.lapsedSeconds + int(now.Sub(t.starTime).Seconds())
				left[id] = t.LimitSeconds - lapsed
			}
			mu.Unlock()
			Send("timer", "tick", &B_TimerTick{
				Tick: tick,
				Left: left,
			})

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

	earliest_time_to_update := time.Now().Add(time.Hour)
	timer_needed := false

	for id, t := range timers {
		if t.paused {
			continue
		}
		lapsed := t.lapsedSeconds + int(time.Since(t.starTime).Seconds())
		if lapsed >= t.LimitSeconds {
			Send("timer", "done", &B_TimerDone{
				ID: id,
			})
			delete(timers, id) // remove timer
			continue
		}

		time_to_update := t.starTime.Add(time.Duration(t.LimitSeconds) * time.Second)
		if time_to_update.Before(earliest_time_to_update) {
			earliest_time_to_update = time_to_update
		}

		timer_needed = true
	}

	if !timer_needed {
		nextCheckTimer = time.AfterFunc(time.Until(earliest_time_to_update), func() {
			updateTimers()
		})
	}

}

func timer_init(id int, d *B_TimerInit) {
	mu.Lock()

	if _, ok := timers[id]; ok {
		log.Warn().Msgf("Timer %d already exists", id)
	}

	if d.LimitSeconds > d.HardLimitSeconds {
		log.Warn().Msgf("Timer %d has a limit greater than the hard limit", id)
	}

	timers[id] = &BusTimer{
		paused:           true,
		LimitSeconds:     d.LimitSeconds,
		HardLimitSeconds: d.HardLimitSeconds,
		lapsedSeconds:    0,
	}
	mu.Unlock()

	if d.Start {
		timer_start(id)
	}
}

func timer_start(id int) {
	mu.Lock()
	defer mu.Unlock()

	t, ok := timers[id]
	if !ok {
		log.Error().Msgf("Timer %d does not exist", id)
		return
	}

	if !t.paused {
		log.Warn().Msgf("Timer %d is already running", id)
		return
	}

	if t.lapsedSeconds >= t.LimitSeconds {
		log.Warn().Msgf("Timer %d has reached its limit", id)
		return
	}

	t.paused = false
	t.starTime = time.Now()
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
	t.lapsedSeconds += int(time.Since(t.starTime).Seconds())
}

func timer_reset(id int) {
	mu.Lock()
	defer mu.Unlock()

	t, ok := timers[id]
	if !ok {
		log.Error().Msgf("Timer %d does not exist", id)
		return
	}

	lapsed := t.lapsedSeconds
	if !t.paused {
		lapsed += int(time.Since(t.starTime).Seconds())
	}

	if lapsed >= t.HardLimitSeconds {
		log.Warn().Msgf("Timer %d has reached its hard limit", id)
		return
	}

	t.lapsedSeconds = 0
	t.HardLimitSeconds -= lapsed
	if t.LimitSeconds > t.HardLimitSeconds {
		t.lapsedSeconds = t.HardLimitSeconds
	}
	t.starTime = time.Now()
}