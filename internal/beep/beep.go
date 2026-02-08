package beep

import (
	"errors"
	"io"
	"math"
	"sync"
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

var (
    ctx  *oto.Context
    once sync.Once
	beepQueue chan struct{}
)

func Init() error {
    var err error
    once.Do(func() {
		var ready <-chan struct{}
        ctx, ready, err = oto.NewContext(44100, 1, 2)
		if err != nil {
			return
		}
		<-ready
		beepQueue = make(chan struct{}, 10)
		go audioWorker()
    })
    return err
}

func Close()  {
	if beepQueue != nil {
		close(beepQueue)
	}
}

func audioWorker()  {
	for _ = range beepQueue {
		playBeep(1318.51, time.Millisecond*100)
		playBeep(1396.91, time.Millisecond * 50)
		playBeep(1567.98, time.Millisecond * 50)
		playBeep(1396.91, time.Millisecond * 50)
		playBeep(1318.51, time.Millisecond*100)
	}
}

func Beep()  {
	if beepQueue == nil {
		return
	}

	select {
	case beepQueue <- struct{}{}: // added to queue
	default:                      // too much beeps, skipping
	}
}

func (t *toneReader) Read(p []byte) (int, error) {
	if t.sampleIndex >= t.totalSamples {
		return 0, io.EOF
	}

	n := 0
	for n+1 < len(p) && t.sampleIndex < t.totalSamples {
		v := int16(
			0.3 * 32767 *
				math.Sin(2*math.Pi*t.frequency*
					float64(t.sampleIndex)/float64(t.sampleRate)),
		)

		p[n] = byte(v)
		p[n+1] = byte(v >> 8)

		n += 2
		t.sampleIndex++
	}

	return n, nil
}


func playBeep(freq float64, dur time.Duration) error {
	const sampleRate = 44100
	const channelCount = 1
	const bytesPerSample = 2 // PCM16

	if ctx == nil {
		return errors.New("voice context is null")
	}

	tr := &toneReader{
		sampleRate:   sampleRate,
		frequency:    freq,
		duration:     dur,
		totalSamples: int(float64(sampleRate) * dur.Seconds()),
	}

	player := ctx.NewPlayer(tr)
	defer player.Close()
	player.Play()

	for player.IsPlaying() {
		time.Sleep(time.Millisecond * 10)
	}

	return nil
}