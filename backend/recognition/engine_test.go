package recognition_test

import (
	"context"
	"math"
	"math/rand"
	"testing"

	"dhwani/metadata"
	"dhwani/recognition"
	"dhwani/recognition/audio"
)

// synth builds a deterministic "song": a new chord of three decaying tones every 0.2 s.
func synth(seed int64, seconds float64) []float32 {
	r := rand.New(rand.NewSource(seed))
	sr := float64(audio.SampleRate)
	x := make([]float32, int(seconds*sr))
	note := int(0.2 * sr)
	for start := 0; start < len(x); start += note {
		var fs [3]float64
		for i := range fs {
			fs[i] = 200 + r.Float64()*3000
		}
		for i := 0; i < note && start+i < len(x); i++ {
			env := math.Exp(-6 * float64(i) / float64(note))
			var v float64
			for _, f := range fs {
				v += math.Sin(2 * math.Pi * f * float64(i) / sr)
			}
			x[start+i] = float32(0.3 * env * v)
		}
	}
	return x
}

func addNoise(x []float32, snrDB float64, seed int64) []float32 {
	r := rand.New(rand.NewSource(seed))
	var p float64
	for _, v := range x {
		p += float64(v) * float64(v)
	}
	p /= float64(len(x))
	sigma := math.Sqrt(p / math.Pow(10, snrDB/10))
	out := make([]float32, len(x))
	for i, v := range x {
		out[i] = v + float32(r.NormFloat64()*sigma)
	}
	return out
}

func setup(t *testing.T) *metadata.Memory {
	st := metadata.NewMemory()
	for i := int64(1); i <= 3; i++ {
		_, n, err := st.AddSong(context.Background(), 0,
			metadata.Song{Title: "Song", Artist: string(rune('A' + i)), Duration: 30}, recognition.Fingerprint(synth(i, 30)))
		if err != nil || n == 0 {
			t.Fatalf("add song %d: n=%d err=%v", i, n, err)
		}
	}
	return st
}

func TestRecognize(t *testing.T) {
	st := setup(t)
	sr := audio.SampleRate
	song2 := synth(2, 30)
	clip := song2[10*sr : 18*sr]

	for name, q := range map[string][]float32{
		"clean":      clip,
		"noise 10dB": addNoise(clip, 10, 99),
		"noise 5dB":  addNoise(clip, 5, 99),
	} {
		res, _, err := recognition.Recognize(context.Background(), st, q)
		if err != nil || !res.Matched || res.SongID != 2 {
			t.Errorf("%s: want song 2, got %+v err=%v", name, res, err)
			continue
		}
		off := float64(res.OffsetFrames*recognition.Hop) / float64(sr)
		if math.Abs(off-10) > 0.15 {
			t.Errorf("%s: offset %.2fs, want ~10s", name, off)
		}
		t.Logf("%s: votes=%d confidence=%.2f offset=%.2fs others=%d", name, res.Votes, res.Confidence, off, len(res.Others))
	}
}

func TestNoMatch(t *testing.T) {
	st := setup(t)
	res, _, _ := recognition.Recognize(context.Background(), st, synth(777, 8))
	if res.Matched {
		t.Errorf("unknown song matched: %+v", res)
	}
	res, _, _ = recognition.Recognize(context.Background(), st, addNoise(make([]float32, 8*audio.SampleRate), -10, 1))
	if res.Matched {
		t.Errorf("silence+noise matched: %+v", res)
	}
}
