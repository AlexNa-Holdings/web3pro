package sound

import (
	"bytes"
	_ "embed"
	"io"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto"
	"github.com/rs/zerolog/log"
)

//go:embed mp3/level-up.mp3
var level_up []byte

var soundFiles = map[string][]byte{
	"level-up": level_up,
}

var sounds = map[string]*mp3.Decoder{}
var defaultSound string
var player *oto.Player

func Init() {
	for name, data := range soundFiles {
		decoder, err := mp3.NewDecoder(bytes.NewReader(data))
		if err != nil {
			log.Error().Msgf("Error decoding sound file: %v", err)
			continue
		}
		sounds[name] = decoder

		if defaultSound == "" {
			defaultSound = name
		}
	}
	go Loop()
}

func Loop() {
	// Create a new context for audio playback
	context, err := oto.NewContext(44100, 2, 2, 1000)
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

		d, ok := sounds[name]
		if !ok {
			log.Error().Msgf("sound: sound not found: %v", name)
			return
		}

		if _, err := io.Copy(player, d); err != nil {
			log.Error().Msgf("sound: error playing sound: %v", err)
		}
	case "list":
		l := []string{}
		for name := range sounds {
			l = append(l, name)
		}
		msg.Respond(l, nil)
	}

}
