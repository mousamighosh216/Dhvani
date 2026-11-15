//go:build !nodeps

package audio

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/hajimehoshi/go-mp3"
)

// DecodeMP3 decodes MP3 frames with go-mp3 (always 16-bit stereo output),
// downmixes to mono and scales to [-1.0, 1.0].
func DecodeMP3(r io.Reader) (*AudioData, error) {
	dec, err := mp3.NewDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedFormat, err)
	}
	raw, err := io.ReadAll(dec)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedFormat, err)
	}
	frames := len(raw) / 4 // 2 channels x 2 bytes
	if frames == 0 {
		return nil, fmt.Errorf("%w: no audio frames", ErrUnsupportedFormat)
	}
	out := make([]float64, frames)
	for i := 0; i < frames; i++ {
		l := float64(int16(binary.LittleEndian.Uint16(raw[i*4:])))
		rr := float64(int16(binary.LittleEndian.Uint16(raw[i*4+2:])))
		out[i] = ((l + rr) / 2) / 32768
	}
	return &AudioData{Samples: out, SampleRate: dec.SampleRate()}, nil
}
