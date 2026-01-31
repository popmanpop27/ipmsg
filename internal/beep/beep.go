package beep

import (
	"io"
	"math"
	"time"

	"github.com/hajimehoshi/oto/v2"
)

type toneReader struct {
	sampleRate   int
	frequency    float64
	duration     time.Duration
	sampleIndex  int
	totalSamples int
}

func (t *toneReader) Read(p []byte) (int, error) {
	if t.sampleIndex >= t.totalSamples {
		return 0, io.EOF
	}

	for i := 0; i+1 < len(p) && t.sampleIndex < t.totalSamples; i += 2 {
		// генерируем 16‑битный PCM
		v := int16(0.3 * 32767 * math.Sin(2*math.Pi*t.frequency*float64(t.sampleIndex)/float64(t.sampleRate)))
		p[i] = byte(v)
		p[i+1] = byte(v >> 8)
		t.sampleIndex++
	}

	return len(p), nil
}

func PlayBeep(freq float64, dur time.Duration) error {
	const sampleRate = 44100
	const channelCount = 1
	const bytesPerSample = 2 // PCM16

	// создаём Context (только один раз за программу)
	ctx, ready, err := oto.NewContext(sampleRate, channelCount, bytesPerSample)
	if err != nil {
		return err
	}
	<-ready

	// создаём наш генератор
	tr := &toneReader{
		sampleRate:   sampleRate,
		frequency:    freq,
		duration:     dur,
		totalSamples: int(float64(sampleRate) * dur.Seconds()),
	}

	// создаём Player из Reader
	player := ctx.NewPlayer(tr)

	// Play() асинхронно
	player.Play()

	// ждём, пока проигрывается
	for player.IsPlaying() {
		time.Sleep(time.Millisecond * 10)
	}

	return nil
}