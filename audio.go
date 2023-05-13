package main

import (
	"io"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

const (
	sampleRate     = 48000
	bytesPerSample = 4
)

type musicType int

const (
	typeOgg musicType = iota
	typeMP3
	typeWav
	typeFlac
)

type Player struct {
	audioContext *audio.Context
	audioPlayer  *audio.Player
	totalStr     string
	current      time.Duration
	total        time.Duration
	volume128    int
	musicType    musicType
}

func NewPlayer(f *os.File, audioContext *audio.Context, musicType musicType) (*Player, error) {
	type audioStream interface {
		io.ReadSeeker
		Length() int64
	}
	var s audioStream

	switch musicType {
	case typeOgg:
		var err error
		s, err = vorbis.DecodeWithSampleRate(sampleRate, f)
		if err != nil {
			return nil, err
		}
	case typeMP3:
		var err error
		s, err = mp3.DecodeWithSampleRate(sampleRate, f)
		if err != nil {
			return nil, err
		}
	default:
		panic("not reached")
	}

	p, err := audioContext.NewPlayer(s)
	if err != nil {
		return nil, err
	}
	player := &Player{
		audioContext: audioContext,
		audioPlayer:  p,
		total:        time.Second * time.Duration(s.Length()) / bytesPerSample / sampleRate,
		volume128:    128,
		musicType:    musicType,
	}
	if player.total == 0 {
		player.total = 1
	}
	player.totalStr = player.total.Truncate(time.Second).String()
	player.audioPlayer.Play()
	return player, nil
}

func (p *Player) Close() error {
	return p.audioPlayer.Close()
}

func (p *Player) TogglePause() string {
	if p.audioPlayer.IsPlaying() {
		p.audioPlayer.Pause()
		p.current = p.audioPlayer.Current()
		return "  "
	}
	p.audioPlayer.Play()
	return "  "
}

// HACK: I cannot rewind the stream while paused, so I seek the file manually to
// start.
func (p *Player) Rewind(idx int) {
	if p.audioPlayer.IsPlaying() {
		p.audioPlayer.Rewind()
		p.Close()
	} else {
		fileList[idx].file.Seek(0, 0)
	}
}

func (p *Player) SetVolume(chg int) {
	p.volume128 += chg
	if 128 < p.volume128 {
		p.volume128 = 128
	}
	if p.volume128 < 0 {
		p.volume128 = 0
	}
	p.audioPlayer.SetVolume(float64(p.volume128) / 128)
}

func (p *Player) Seek(chg int) error {
	pos := p.audioPlayer.Current() + time.Second*time.Duration(chg)
	if pos > p.total {
		pos = p.total
	}
	if pos < 0 {
		pos = 0
	}
	if err := p.audioPlayer.Seek(pos); err != nil {
		return err
	}
	p.current = pos
	return nil
}

func (p *Player) Update() (string, float64) {
	if p.audioPlayer.IsPlaying() {
		p.current = p.audioPlayer.Current()
	}
	timeFrac := float64(p.current.Milliseconds()) / float64(p.total.Milliseconds())
	return p.current.Truncate(time.Second).String() + " / " + p.totalStr, timeFrac
}
