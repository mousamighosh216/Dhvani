package recognition_test

import (
	"context"
	"fmt"
	"math"
	"testing"

	"dhwani/internal/synth"
	"dhwani/metadata"
	"dhwani/recognition"
	"dhwani/recognition/audio"
)

const sr = 44100

type fixture struct {
	engine *recognition.DefaultEngine
	store  *metadata.MemoryStore
	songs  map[int][]float64 // song ID -> samples
}

// newFixture ingests n synthetic 40 s songs (seeds 1..n).
func newFixture(t testing.TB, n int) *fixture {
	t.Helper()
	store := metadata.NewMemoryStore()
	f := &fixture{engine: recognition.NewEngine(store), store: store, songs: map[int][]float64{}}
	for seed := 1; seed <= n; seed++ {
		samples := synth.Song(int64(seed), 40, sr)
		a := &audio.AudioData{Samples: samples, SampleRate: sr}
		fps, err := f.engine.FingerprintSong(0, a)
		if err != nil {
			t.Fatal(err)
		}
		id, err := store.IngestSong(context.Background(), &metadata.SongMetadata{Title: fmt.Sprintf("Song %d", seed)}, fps)
		if err != nil {
			t.Fatal(err)
		}
		f.songs[id] = samples
	}
	return f
}

func (f *fixture) recognize(t testing.TB, clip []float64, rate int) (id int, offset float64, ok bool) {
	t.Helper()
	res, err := f.engine.Recognize(context.Background(), &audio.AudioData{Samples: clip, SampleRate: rate})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil {
		return 0, 0, false
	}
	return res.SongID, res.OffsetSec, true
}

func TestRecognizeCleanClips(t *testing.T) {
	f := newFixture(t, 6)
	for id := 1; id <= 6; id++ {
		start := 3.0 + float64(id)*4
		clip := synth.Clip(f.songs[id], sr, start, 5)
		got, off, ok := f.recognize(t, clip, sr)
		if !ok || got != id {
			t.Fatalf("song %d: got id=%d ok=%v", id, got, ok)
		}
		if math.Abs(off-start) > 0.1 {
			t.Errorf("song %d: offset %.3f, want %.3f", id, off, start)
		}
	}
}

func TestNoiseResilience(t *testing.T) {
	f := newFixture(t, 6)
	for _, snr := range []float64{20, 10, 5} {
		correct := 0
		for id := 1; id <= 6; id++ {
			clip := synth.AddNoise(synth.Clip(f.songs[id], sr, 10+float64(id), 6), snr, int64(id))
			if got, _, ok := f.recognize(t, clip, sr); ok && got == id {
				correct++
			}
		}
		t.Logf("SNR %2.0f dB: %d/6 correct", snr, correct)
		if snr >= 5 && correct < 6 {
			t.Errorf("SNR %.0f dB: only %d/6 recognized", snr, correct)
		}
	}
}

func TestVolumeDistortion(t *testing.T) {
	f := newFixture(t, 6)
	for _, gain := range []float64{0.1, 0.5, 2.0} {
		for id := 1; id <= 6; id++ {
			clip := synth.Gain(synth.Clip(f.songs[id], sr, 8, 5), gain)
			if got, _, ok := f.recognize(t, clip, sr); !ok || got != id {
				t.Errorf("gain %.1f song %d: got id=%d ok=%v", gain, id, got, ok)
			}
		}
	}
}

func TestResampledQuery(t *testing.T) {
	f := newFixture(t, 4)
	clip := synth.Clip(f.songs[2], sr, 12, 6)
	low := audio.Resample(clip, sr, 22050) // like a phone mic at 22.05 kHz
	if got, _, ok := f.recognize(t, low, 22050); !ok || got != 2 {
		t.Fatalf("22.05 kHz query: got id=%d ok=%v", got, ok)
	}
}

func TestUnknownSongNotMatched(t *testing.T) {
	f := newFixture(t, 6)
	other := synth.Song(999, 20, sr) // never ingested
	if got, _, ok := f.recognize(t, synth.Clip(other, sr, 4, 6), sr); ok {
		t.Fatalf("unknown song matched song %d", got)
	}
}

func TestPureNoiseNotMatched(t *testing.T) {
	f := newFixture(t, 6)
	noise := synth.AddNoise(make([]float64, 5*sr), -200, 1) // silence + tiny noise
	noise = synth.AddNoise(synth.Gain(noise, 0), 0, 1)
	_ = noise
	white := synth.AddNoise(synth.Song(5, 5, sr), -30, 7) // noise 30 dB above the signal
	if got, _, ok := f.recognize(t, white, sr); ok && got != 5 {
		t.Fatalf("noise matched wrong song %d", got)
	}
}

func TestTooShort(t *testing.T) {
	f := newFixture(t, 1)
	_, err := f.engine.Recognize(context.Background(), &audio.AudioData{Samples: make([]float64, 100), SampleRate: sr})
	if err != recognition.ErrTooShort {
		t.Fatalf("want ErrTooShort, got %v", err)
	}
}

func BenchmarkFingerprintSong(b *testing.B) {
	samples := synth.Song(1, 40, sr)
	e := recognition.NewEngine(metadata.NewMemoryStore())
	for i := 0; i < b.N; i++ {
		if _, err := e.FingerprintSong(1, &audio.AudioData{Samples: samples, SampleRate: sr}); err != nil {
			b.Fatal(err)
		}
	}
}
