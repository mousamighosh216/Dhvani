// Package audio decodes any audio container to mono float32 at 44.1 kHz using ffmpeg.
package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strings"
)

const SampleRate = 44100

func Decode(r io.Reader) ([]float32, error) {
	f, err := os.CreateTemp("", "dhwani-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return nil, err
	}
	f.Close()

	cmd := exec.Command("ffmpeg", "-v", "error", "-nostdin", "-i", f.Name(),
		"-vn", "-f", "f32le", "-ac", "1", "-ar", fmt.Sprint(SampleRate), "pipe:1")
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("could not decode audio: %s", strings.TrimSpace(stderr.String()))
	}
	b := out.Bytes()
	if len(b) < 4 {
		return nil, fmt.Errorf("no audio found in file")
	}
	x := make([]float32, len(b)/4)
	for i := range x {
		x[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return x, nil
}
