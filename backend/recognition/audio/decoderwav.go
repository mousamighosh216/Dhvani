package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

func DecodeWAV(r io.ReadSeeker) (*AudioData, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if bytes.Equal(buf[:4], []byte("RIFF")) && bytes.Equal(buf[8:12], []byte("WAVE")) {
		return nil, fmt.Errorf("audio: WAV decoding not implemented")
	}

	var (
		format, channels, bits int
		sampleRate             int
		haveFmt                bool
		data                   []byte
	)
	// refer Info.md (4) to know the format of wav file and how to parse it
	pos := 12
	for pos+8 <= len(buf) {
		id := string(buf[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(buf[pos+4 : pos+8]))
		body := pos + 8
		end := body + size
		if size > len(buf)-body {
			end = len(buf)
			return nil, fmt.Errorf("%w: invalid chunk size", ErrUnsupportedFormat)
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
	return &AudioData{
		Samples:    out,
		SampleRate: sampleRate,
	}, nil
}

func readSample(data []byte, format, bits int) float64 {
	if format == 3 {
		if bits == 32 {
			return float64(math.Float32frombits(binary.LittleEndian.Uint32(data)))
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(data))
	}

	switch bits {
	case 8:
		return (float64(data[0]) - 128) / 128
	case 16:
		return float64(int16(binary.LittleEndian.Uint16(data))) / 32768
	case 24:
		value := int32(data[0]) | int32(data[1])<<8 | int32(data[2])<<16
		if value&0x800000 != 0 {
			value |= ^0xffffff
		}
		return float64(value) / 8388608
	case 32:
		return float64(int32(binary.LittleEndian.Uint32(data))) / 2147483648
	default:
		return 0
	}
}
