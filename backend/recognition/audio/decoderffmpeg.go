package audio

import (
	"fmt"
	"io"
)

func decodeWithFFmpeg(r io.Reader) (*AudioData, error) {
	// use ffmpeg to decode audio data
	// this is a placeholder for actual ffmpeg decoding logic
	return nil, fmt.Errorf("audio: ffmpeg decoding not implemented")
}
