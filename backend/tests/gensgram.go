package tests

import (
	"Dhvani/recognition/audio"
	"Dhvani/recognition/fft"
	"fmt"
	"os"
)

func GenSpecGram() {
	file, err := os.Open("testsamples/songsample.mp3")
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	audioData, err := audio.DecodeMP3(file)
	if err != nil {
		fmt.Printf("Error decoding WAV: %v\n", err)
		return
	}

	// Extract first 15 seconds for a crisp visualization
	sampleLimit := 15 * audioData.SampleRate
	if len(audioData.Samples) > sampleLimit {
		audioData.Samples = audioData.Samples[:sampleLimit]
	}

	fmt.Println("Computing STFT for visualization...")
	windowSize := 2048
	hopSize := 512

	spectrogram := fft.ComputeSTFT(audioData.Samples, audioData.SampleRate, windowSize, hopSize)

	outputImage := "spectrogramsamples/spectrogram.png"
	fmt.Printf("Saving Spectrogram Image to %s...\n", outputImage)

	err = fft.SaveSpectrogramPNG(spectrogram, outputImage)
	if err != nil {
		fmt.Printf("Failed to render spectrogram image: %v\n", err)
		return
	}

	fmt.Println("Done! Open 'spectrogram.png' in your file viewer to see the sound wave frequencies.")
}
