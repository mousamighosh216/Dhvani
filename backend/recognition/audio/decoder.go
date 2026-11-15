// Package audio decodes audio files into normalized mono float samples.
package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os/exec"
	"time"
)

// AudioData holds mono samples in [-1.0, 1.0].
type AudioData struct {
	Samples    []float64
	SampleRate int
}

// Duration returns the clip length in seconds.
func (a *AudioData) Duration() float64 {
	if a == nil || a.SampleRate == 0 {
		return 0
	}
	return float64(len(a.Samples)) / float64(a.SampleRate)
}

var (
	// ErrUnsupportedFormat is returned for data that cannot be decoded.
	ErrUnsupportedFormat = errors.New("audio: unsupported or corrupt audio data")
	// ErrFFmpegMissing is returned when ffmpeg is needed but not installed.
	ErrFFmpegMissing = errors.New("audio: ffmpeg is required for this format but was not found in PATH")
)

// FFmpegSampleRate is the rate ffmpeg is asked to output.
const FFmpegSampleRate = 44100

// DecodeAudio sniffs the container and routes to the right decoder.
// WAV is decoded natively, MP3 via DecodeMP3, everything else (WebM/Opus
// from browsers, OGG, M4A, FLAC...) is converted with ffmpeg.
func DecodeAudio(r io.ReadSeeker) (*AudioData, error) {
	head := make([]byte, 12)
	n, err := io.ReadFull(r, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedFormat, err)
	}
	head = head[:n]
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	switch {
	case len(head) >= 12 && string(head[0:4]) == "RIFF" && string(head[8:12]) == "WAVE":
		return DecodeWAV(r)
	case len(head) >= 3 && string(head[0:3]) == "ID3",
		len(head) >= 2 && head[0] == 0xFF && head[1]&0xE0 == 0xE0:
		return DecodeMP3(r)
	default:
		return decodeWithFFmpeg(r)
	}
}

// DecodeWAV parses a WAV file (PCM 8/16/24/32-bit or IEEE float), downmixes
// to mono and scales to [-1.0, 1.0].
func DecodeWAV(r io.ReadSeeker) (*AudioData, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(buf) < 12 || string(buf[0:4]) != "RIFF" || string(buf[8:12]) != "WAVE" {
		return nil, fmt.Errorf("%w: missing RIFF/WAVE header", ErrUnsupportedFormat)
	}

	var (
		format, channels, bits int
		sampleRate             int
		haveFmt                bool
		data                   []byte
	)
	pos := 12
	for pos+8 <= len(buf) {
		id := string(buf[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(buf[pos+4 : pos+8]))
		body := pos + 8
		end := body + size
		if end > len(buf) || size < 0 { // streamed / truncated files: take the rest
			end = len(buf)
		}
		switch id {
		case "fmt ":
			if end-body < 16 {
				return nil, fmt.Errorf("%w: short fmt chunk", ErrUnsupportedFormat)
			}
			format = int(binary.LittleEndian.Uint16(buf[body:]))
			channels = int(binary.LittleEndian.Uint16(buf[body+2:]))
			sampleRate = int(binary.LittleEndian.Uint32(buf[body+4:]))
			bits = int(binary.LittleEndian.Uint16(buf[body+14:]))
			if format == 0xFFFE && end-body >= 26 { // WAVE_FORMAT_EXTENSIBLE
				format = int(binary.LittleEndian.Uint16(buf[body+24:]))
			}
			haveFmt = true
		case "data":
			data = buf[body:end]
		}
		if data != nil && haveFmt {
			break
		}
		pos = end + (size & 1) // chunks are word aligned
	}
	if !haveFmt || data == nil {
		return nil, fmt.Errorf("%w: missing fmt or data chunk", ErrUnsupportedFormat)
	}
	if channels < 1 || sampleRate < 1 {
		return nil, fmt.Errorf("%w: bad channel count or sample rate", ErrUnsupportedFormat)
	}

	bytesPer := bits / 8
	if bytesPer < 1 || (format != 1 && format != 3) {
		return nil, fmt.Errorf("%w: format %d with %d bits", ErrUnsupportedFormat, format, bits)
	}
	if format == 3 && bits != 32 && bits != 64 {
		return nil, fmt.Errorf("%w: float with %d bits", ErrUnsupportedFormat, bits)
	}
	if format == 1 && (bits < 8 || bits > 32 || bits%8 != 0) {
		return nil, fmt.Errorf("%w: pcm with %d bits", ErrUnsupportedFormat, bits)
	}

	frameBytes := bytesPer * channels
	frames := len(data) / frameBytes
	out := make([]float64, frames)
	for i := 0; i < frames; i++ {
		var sum float64
		for c := 0; c < channels; c++ {
			sum += readSample(data[i*frameBytes+c*bytesPer:], format, bits)
		}
		out[i] = sum / float64(channels)
	}
	return &AudioData{Samples: out, SampleRate: sampleRate}, nil
}

// readSample decodes one sample and normalizes it to [-1.0, 1.0).
func readSample(b []byte, format, bits int) float64 {
	if format == 3 {
		if bits == 32 {
			return float64(math.Float32frombits(binary.LittleEndian.Uint32(b)))
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(b))
	}
	switch bits {
	case 8: // unsigned
		return (float64(b[0]) - 128) / 128
	case 16:
		return float64(int16(binary.LittleEndian.Uint16(b))) / 32768
	case 24:
		v := int32(b[0]) | int32(b[1])<<8 | int32(b[2])<<16
		if v&0x800000 != 0 {
			v |= ^0xFFFFFF // sign extend
		}
		return float64(v) / 8388608
	default: // 32
		return float64(int32(binary.LittleEndian.Uint32(b))) / 2147483648
	}
}

// Resample converts samples between rates using linear interpolation.
func Resample(samples []float64, fromRate int, toRate int) []float64 {
	if fromRate == toRate || len(samples) == 0 || fromRate <= 0 || toRate <= 0 {
		out := make([]float64, len(samples))
		copy(out, samples)
		return out
	}
	n := int(float64(len(samples)) * float64(toRate) / float64(fromRate))
	out := make([]float64, n)
	ratio := float64(fromRate) / float64(toRate)
	last := len(samples) - 1
	for i := range out {
		pos := float64(i) * ratio
		i0 := int(pos)
		if i0 >= last {
			out[i] = samples[last]
			continue
		}
		frac := pos - float64(i0)
		out[i] = samples[i0]*(1-frac) + samples[i0+1]*frac
	}
	return out
}

// decodeWithFFmpeg converts any container to raw mono 16-bit PCM using ffmpeg.
func decodeWithFFmpeg(r io.Reader) (*AudioData, error) {
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, ErrFFmpegMissing
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path,
		"-hide_banner", "-loglevel", "error",
		"-i", "pipe:0", "-vn",
		"-f", "s16le", "-ac", "1", "-ar", fmt.Sprint(FFmpegSampleRate),
		"pipe:1")
	cmd.Stdin = r
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: ffmpeg: %v: %s", ErrUnsupportedFormat, err, bytes.TrimSpace(stderr.Bytes()))
	}
	raw := stdout.Bytes()
	n := len(raw) / 2
	if n == 0 {
		return nil, fmt.Errorf("%w: ffmpeg produced no audio", ErrUnsupportedFormat)
	}
	samples := make([]float64, n)
	for i := 0; i < n; i++ {
		samples[i] = float64(int16(binary.LittleEndian.Uint16(raw[i*2:]))) / 32768
	}
	return &AudioData{Samples: samples, SampleRate: FFmpegSampleRate}, nil
}
