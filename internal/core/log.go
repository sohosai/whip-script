package core

import (
	"fmt"
	"log"
	"strings"
)

var Log_enable = true

func (c Config) Log() {
	if Log_enable {
		log.Print(`
			LOG_LEVEL: hoge
			DEVICE_ID: hoge
			RESOLUTION: hoge
			FRAMERATE: hoge
			VIDEO_CODEC_TYPE: hoge
			VIDEO_BIT_RATE: hoge
			AUDIO_BIT_RATE: hoge
			`,
		)
	}
}

func ErrorLog(s ...string) {
	if Log_enable {
		fmt.Printf(`
****************************************************
error occured!!
%s
****************************************************
`, strings.Join(s, " "))
	}
}
func Log(s ...string) {
	if Log_enable {
		fmt.Print(strings.Join(s, " "))
	}
}
