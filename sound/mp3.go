package sound

import (
	_ "embed"
)

//go:embed mp3/level-up.mp3
var level_up []byte

//go:embed mp3/level-up-2.mp3
var level_up_2 []byte

//go:embed mp3/bottle.mp3
var bottle []byte

//go:embed mp3/soft.mp3
var soft []byte

//go:embed mp3/simple.mp3
var simple []byte

//go:embed mp3/ringtone.mp3
var ringtone []byte

//go:embed mp3/message.mp3
var message []byte

var soundFiles = []struct {
	name string
	data []byte
}{
	{"level-up", level_up},
	{"level-up2", level_up_2},
	{"bottle", bottle},
	{"soft", soft},
	{"simple", simple},
	{"ringtone", ringtone},
	{"message", message},
}
