package bus

import (
	"errors"
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
						t.lapsed = t.Limit
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
			left := make(map[int]time.Duration)
			tick++

			for id, t := range timers {
				if t.paused {
					continue
				}
				now := time.Now()
				left[id] = t.Limit - (t.lapsed + now.Sub(t.starTime))
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

	update_after := 60 * time.Minute // 1 hour
	timer_needed := false

	for id, t := range timers {
		if !t.paused {
			l := t.lapsed + time.Since(t.starTime)
			if l >= t.Limit {
				Send("timer", "done", &B_TimerDone{
					ID: id,
				})
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

func timer_init(id int, d *B_TimerInit) {
	mu.Lock()

	if _, ok := timers[id]; ok {
		log.Warn().Msgf("Timer %d already exists", id)
	}

	if d.Limit > d.HardLimit {
		log.Warn().Msgf("Timer %d has a limit greater than the hard limit", id)
	}

	timers[id] = &BusTimer{
		paused:    true,
		Limit:     d.Limit,
		HardLimit: d.HardLimit,
		lapsed:    0,
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

	if t.lapsed >= t.Limit {
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
		t.lapsed = t.HardLimit
	}
	t.starTime = time.Now()
}
