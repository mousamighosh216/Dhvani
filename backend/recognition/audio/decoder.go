package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net/http"

	// The decoder uses wav's PCM buffer directly; no separate audio package import is needed.
	"github.com/go-audio/wav"
	"github.com/hajimehoshi/go-mp3"
)

// AudioData holds the nromalized mono PCM samples and metadata
type AudioData struct {
	Samples    []float64 // amplitude values normalized between -1.0 to 1.0
	SampleRate int       // no. of samples per sec
}

// DecodeWAV reads a WAV stream -> downmix to mono -> normalize amplitude
func DecodeWAV(r io.ReadSeeker) (*AudioData, error) {
	decoder := wav.NewDecoder(r)

	if !decoder.IsValidFile() {
		return nil, fmt.Errorf("invalid WAV file header")
	}

	// read full audio buffer
	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to read PCM Buffer: %w", err)
	}

	numChannels := buf.Format.NumChannels
	sampleRate := buf.Format.SampleRate
	totalFrames := len(buf.Data) / numChannels

	monoSamples := make([]float64, totalFrames)

	// determine normalization factor based on bit depth
	// Fetch BitDepth from the decoder directly (or buffer source)
	bitDepth := int(decoder.BitDepth)
	if bitDepth == 0 {
		bitDepth = buf.SourceBitDepth
	}
	if bitDepth == 0 {
		bitDepth = 16 // Safe default fallback for standard WAV files
	}
	maxVal := math.Pow(2, float64(bitDepth-1))

	// Downmix channels to mono and normalize to [-1.0, 1.0] range
	for i := range totalFrames {
		var sum float64
		for ch := range numChannels {
			sampleIndex := i*numChannels + ch
			sum += float64(buf.Data[sampleIndex])
		}
		// Average the channels and scale amplitude
		avgSample := sum / float64(numChannels)
		monoSamples[i] = avgSample / maxVal
	}

	return &AudioData{
		Samples:    monoSamples,
		SampleRate: sampleRate,
	}, nil
}

func DecodeMP3(r io.Reader) (*AudioData, error) {
	decoder, err := mp3.NewDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MP3 decoder: %w", err)
	}

	sampleRate := decoder.SampleRate()
	// go-mp3 always outputs 16-bit, 2 channel (stereo) little endian pcm data
	numChannels := 2
	maxVal := math.Pow(2, 15) // 16 bit signed int max = 32768

	//read all decoded raw pcm bytes from the decoder
	pcmBytes, err := io.ReadAll(decoder)
	if err != nil {
		return nil, fmt.Errorf("failed to read decoded MP3 bytes: %w", err)
	}

	// Each frame consists of 2 channels * 2 bytes per sample (16-bit) = 4 bytes per stereo frame
	bytesPerFrame := numChannels * 2
	totalFrames := len(pcmBytes) / bytesPerFrame

	monoSamples := make([]float64, totalFrames)

	// Convert 16-bit integer bytes into normalized mono float64 samples
	for i := range totalFrames {
		offset := i * bytesPerFrame

		// Read Left Channel (16-bit Little Endian)
		leftInt := int16(binary.LittleEndian.Uint16(pcmBytes[offset : offset+2]))
		// Read Right Channel (16-bit Little Endian)
		rightInt := int16(binary.LittleEndian.Uint16(pcmBytes[offset+2 : offset+4]))

		// Downmix channels to mono and scale to [-1.0, 1.0]
		avgInt := (float64(leftInt) + float64(rightInt)) / 2.0
		monoSamples[i] = avgInt / maxVal
	}

	return &AudioData{
		Samples:    monoSamples,
		SampleRate: sampleRate,
	}, nil
}

// DecodeAudio detects MIME type and routes to the correct decoder
func DecodeAudio(r io.ReadSeeker) (*AudioData, error) {
	// read first 512 bytes to sniff the MIME type
	buf := make([]byte, 0)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to inspect audio file header: %w", err)
	}

	// reset reader to init position
	_, err = r.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to reset stream offset: %w", err)
	}

	mimeType := http.DetectContentType(buf[:n])
	switch {
	case mimeType == "audio/x-wav" || mimeType == "audio/wav":
		return DecodeWAV(r)
	case mimeType == "audio/mpeg" || mimeType == "audio/mp3":
		return DecodeMP3(r)
	default:
		// mp3 decoding as a fallback if sniffing returns octet stream
		if data, mp3Err := DecodeMP3(r); mp3Err == nil {
			return data, nil
		}
		return nil, fmt.Errorf("unsupported audio format: %s", mimeType)
	}
}
