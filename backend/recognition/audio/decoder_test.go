package audio

import (
	"bytes"
	"encoding/binary"
	"math"
	"os/exec"
	"testing"
)

// wav builds a WAV file from interleaved int values.
func wav(format, channels, rate, bits int, payload []byte) []byte {
	var b bytes.Buffer
	w := func(v any) { binary.Write(&b, binary.LittleEndian, v) }
	b.WriteString("RIFF")
	w(uint32(36 + len(payload)))
	b.WriteString("WAVEfmt ")
	w(uint32(16))
	w(uint16(format))
	w(uint16(channels))
	w(uint32(rate))
	w(uint32(rate * channels * bits / 8))
	w(uint16(channels * bits / 8))
	w(uint16(bits))
	b.WriteString("data")
	w(uint32(len(payload)))
	b.Write(payload)
	return b.Bytes()
}

func pcm16(vals ...int16) []byte {
	var b bytes.Buffer
	for _, v := range vals {
		binary.Write(&b, binary.LittleEndian, v)
	}
	return b.Bytes()
}

func TestDecodeWAV16MonoNormalization(t *testing.T) {
	a, err := DecodeWAV(bytes.NewReader(wav(1, 1, 44100, 16, pcm16(0, 16384, -32768, 32767))))
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{0, 0.5, -1, 32767.0 / 32768}
	if a.SampleRate != 44100 || len(a.Samples) != 4 {
		t.Fatalf("rate=%d n=%d", a.SampleRate, len(a.Samples))
	}
	for i, w := range want {
		if math.Abs(a.Samples[i]-w) > 1e-9 {
			t.Errorf("sample %d = %v, want %v", i, a.Samples[i], w)
		}
		if a.Samples[i] < -1 || a.Samples[i] > 1 {
			t.Errorf("sample %d out of range", i)
		}
	}
}

func TestDecodeWAVStereoDownmix(t *testing.T) {
	// L=16384 R=-16384 -> 0 ; L=R=8192 -> 0.25
	a, err := DecodeWAV(bytes.NewReader(wav(1, 2, 22050, 16, pcm16(16384, -16384, 8192, 8192))))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Samples) != 2 || a.Samples[0] != 0 || a.Samples[1] != 0.25 {
		t.Fatalf("got %v", a.Samples)
	}
}

func TestDecodeWAV8And24And32Float(t *testing.T) {
	a, _ := DecodeWAV(bytes.NewReader(wav(1, 1, 8000, 8, []byte{128, 255, 0})))
	if a.Samples[0] != 0 || a.Samples[2] != -1 || a.Samples[1] < 0.99 {
		t.Errorf("8-bit: %v", a.Samples)
	}
	// 24-bit: +0.5 and -0.5
	p24 := []byte{0x00, 0x00, 0x40, 0x00, 0x00, 0xC0}
	a, err := DecodeWAV(bytes.NewReader(wav(1, 1, 8000, 24, p24)))
	if err != nil || math.Abs(a.Samples[0]-0.5) > 1e-6 || math.Abs(a.Samples[1]+0.5) > 1e-6 {
		t.Errorf("24-bit: %v %v", a, err)
	}
	var f bytes.Buffer
	binary.Write(&f, binary.LittleEndian, []float32{0.25, -0.75})
	a, err = DecodeWAV(bytes.NewReader(wav(3, 1, 8000, 32, f.Bytes())))
	if err != nil || a.Samples[0] != 0.25 || a.Samples[1] != -0.75 {
		t.Errorf("float32: %v %v", a, err)
	}
}

func TestDecodeWAVStreamedSizeAndExtraChunk(t *testing.T) {
	b := wav(1, 1, 8000, 16, pcm16(1000, 2000, 3000))
	// A bogus huge data size (as streamed WAVs have) must still decode.
	binary.LittleEndian.PutUint32(b[40:], 0xFFFFFFFF)
	a, err := DecodeWAV(bytes.NewReader(b))
	if err != nil || len(a.Samples) != 3 {
		t.Fatalf("streamed: %v %v", a, err)
	}
	// LIST chunk before data.
	var c bytes.Buffer
	c.Write(b[:36])
	c.WriteString("LIST")
	binary.Write(&c, binary.LittleEndian, uint32(5))
	c.Write([]byte("abcde\x00")) // 5 bytes + pad
	c.Write(b[36:])
	a, err = DecodeWAV(bytes.NewReader(c.Bytes()))
	if err != nil || len(a.Samples) != 3 {
		t.Fatalf("extra chunk: %v %v", a, err)
	}
}

func TestDecodeWAVErrors(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty":    {},
		"garbage":  []byte("this is definitely not a wav file"),
		"no data":  wav(1, 1, 8000, 16, nil)[:36],
		"zero chn": wav(1, 0, 8000, 16, pcm16(1, 2)),
	} {
		if _, err := DecodeWAV(bytes.NewReader(data)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestDecodeAudioRoutesWAV(t *testing.T) {
	a, err := DecodeAudio(bytes.NewReader(wav(1, 1, 8000, 16, pcm16(1, 2, 3))))
	if err != nil || len(a.Samples) != 3 {
		t.Fatalf("%v %v", a, err)
	}
}

func TestDecodeAudioUnknownFormat(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if _, err := DecodeAudio(bytes.NewReader([]byte("plain text, not audio at all"))); err == nil {
		t.Fatal("expected error for non-audio data")
	}
}

func TestResample(t *testing.T) {
	// 1 kHz sine at 8 kHz -> 16 kHz keeps frequency and doubles length.
	in := make([]float64, 8000)
	for i := range in {
		in[i] = math.Sin(2 * math.Pi * 1000 * float64(i) / 8000)
	}
	out := Resample(in, 8000, 16000)
	if len(out) != 16000 {
		t.Fatalf("len %d", len(out))
	}
	for i := 100; i < 200; i++ {
		want := math.Sin(2 * math.Pi * 1000 * float64(i) / 16000)
		if math.Abs(out[i]-want) > 0.2 { // linear interpolation error bound
			t.Fatalf("i=%d got %v want %v", i, out[i], want)
		}
	}
	if got := Resample(in, 8000, 8000); len(got) != len(in) {
		t.Error("same-rate resample changed length")
	}
	if got := Resample(in, 44100, 22050); len(got) != 4000 {
		t.Errorf("downsample len %d", len(got))
	}
}

// ffmpeg-based formats: generate a tone as MP3 and WebM/Opus and decode them.
func TestDecodeFFmpegFormats(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	for _, c := range []struct {
		name, ext string
		args      []string
	}{
		{"mp3", "mp3", []string{"-c:a", "libmp3lame"}},
		{"webm-opus", "webm", []string{"-c:a", "libopus"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			args := append([]string{"-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=2"}, c.args...)
			args = append(args, "-f", c.ext, "pipe:1")
			out, err := exec.Command("ffmpeg", args...).Output()
			if err != nil || len(out) == 0 {
				t.Skipf("ffmpeg cannot encode %s here: %v", c.name, err)
			}
			a, err := DecodeAudio(bytes.NewReader(out))
			if err != nil {
				t.Fatal(err)
			}
			if d := a.Duration(); d < 1.8 || d > 2.3 {
				t.Errorf("duration %.2f s", d)
			}
			var peak float64
			for _, s := range a.Samples {
				peak = math.Max(peak, math.Abs(s))
			}
			if peak < 0.1 || peak > 1 {
				t.Errorf("peak %.3f", peak)
			}
		})
	}
}
