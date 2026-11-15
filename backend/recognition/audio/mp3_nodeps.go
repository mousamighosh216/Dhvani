//go:build nodeps

package audio

import "io"

// DecodeMP3 (nodeps build): go-mp3 is not available, so ffmpeg decodes MP3.
func DecodeMP3(r io.Reader) (*AudioData, error) {
	return decodeWithFFmpeg(r)
}
