package sound

import (
	"bytes"
	_ "embed"
	"io"
	"sync"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto"
	"github.com/rs/zerolog/log"
)

var sounds = map[string]*mp3.Decoder{}
var defaultSound string
var player *oto.Player
var playerMutex = sync.Mutex{}

func Init() {
	for _, sf := range soundFiles {
		decoder, err := mp3.NewDecoder(bytes.NewReader(sf.data))
		if err != nil {
			log.Error().Msgf("Error decoding sound file: %v", err)
			continue
		}
		sounds[sf.name] = decoder

		if defaultSound == "" {
			defaultSound = sf.name
		}
	}
	go Loop()
}

func Loop() {
	// Create a new context for audio playback
	context, err := oto.NewContext(44100, 2, 2, 44000)
	if err != nil {
		log.Error().Msgf("Error creating audio context: %v", err)
		return
	}
	player = context.NewPlayer()
	defer player.Close()

	ch := bus.Subscribe("sound")

	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}
		go process(msg)
	}
}

func process(msg *bus.Message) {
	switch msg.Type {
	case "play":
		name, ok := msg.Data.(string)
		if !ok || name == "" {
			if defaultSound == "" {
				log.Error().Msg("sound: no default sound")
				return
			} else {
				name = defaultSound
			}
		}

		s, ok := sounds[name]
		if !ok {
			log.Error().Msgf("sound: sound not found: %v", name)
			return
		}

		play(s)

	case "list":
		l := []string{}
		for _, sf := range soundFiles {
			l = append(l, sf.name)
		}
		msg.Respond(l, nil)
	}

}

func play(d *mp3.Decoder) {
	if cmn.CurrentWallet == nil || !cmn.CurrentWallet.SoundOn {
		return
	}

	go func() {
		playerMutex.Lock()
		defer playerMutex.Unlock()

		d.Seek(0, io.SeekStart)
		if _, err := io.Copy(player, d); err != nil {
			log.Error().Msgf("sound: error playing sound: %v", err)
		}
	}()
}
