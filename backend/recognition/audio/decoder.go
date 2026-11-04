package audio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// holds monodata audio samples
type AudioData struct {
	Samples    []float64
	SampleRate int
}

// duration of our clip in seconds
func (a *AudioData) GetDuration() float64 {
	if a == nil || a.SampleRate == 0 {
		return 0
	}
	return float64(len(a.Samples)) / float64(a.SampleRate)
}

var ErrUnsupportedFormat = errors.New("audio: unsupported audio format")

func Decoder(r io.ReadSeeker) (*AudioData, error) {
	// read some part of audio file to determine format
	// refer Info.md (1) to know why 12 bytes
	buf := make([]byte, 12)
	// why readfull? it ensures buf is completely filled or returns an error if not enough bytes are read
	n, err := io.ReadFull(r, buf)
	if err != nil && errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, fmt.Errorf("%w : %v", ErrUnsupportedFormat, err)
	}
	buf = buf[:n]

	// readfull may read our audio file completely, so we need to seek back to the start of the file for further processing
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	switch {
	// check for wav format
	case bytes.Equal(buf[:4], []byte("RIFF")) && bytes.Equal(buf[8:12], []byte("WAVE")):
		return DecodeWAV(r)
	// check for mp3 format
	case bytes.Equal(buf[:3], []byte("ID3")),
		len(buf) >= 2 && buf[0] == 0xFF && buf[1]&0xE0 == 0xE0:
		return DecodeMP3(r)
	// fallback to ffmpeg for other formats
	default:
		return decodeWithFFmpeg(r)
	}
}
