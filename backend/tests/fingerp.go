package tests

import (
	"fmt"
	"os"

	"Dhvani/recognition/audio"
	"Dhvani/recognition/fft"
	"Dhvani/recognition/fingerprint"
	"Dhvani/recognition/peaks"
)

func FingerPrintTest() {
	file, err := os.Open("testsamples/sample1.wav")
	if err != nil {
		fmt.Printf("Error opening WAV file: %v\n", err)
		return
	}
	defer file.Close()

	audioData, err := audio.DecodeWAV(file)
	if err != nil {
		fmt.Printf("Error decoding WAV: %v\n", err)
		return
	}

	fmt.Println("Processing pipeline: Audio -> STFT -> Peaks -> Fingerprints...")

	// 1. STFT
	windowSize := 2048
	hopSize := 512
	spectrogram := fft.ComputeSTFT(audioData.Samples, audioData.SampleRate, windowSize, hopSize)

	// 2. Peak Extraction
	neighborhoodTime := 10
	neighborhoodFreq := 10
	minMagnitude := 0.05
	extractedPeaks := peaks.ExtractPeaks(spectrogram, neighborhoodTime, neighborhoodFreq, minMagnitude)

	// 3. Fingerprint Generation
	songID := 101 // Mock Song ID
	targetZone := fingerprint.DefaultTargetZone()
	generatedFingerprints := fingerprint.GenerateFingerprints(extractedPeaks, songID, targetZone)

	fmt.Println("\n==================================================")
	fmt.Printf("Extracted Peak Landmarks : %d peaks\n", len(extractedPeaks))
	fmt.Printf("Generated Fingerprints   : %d packed 32-bit hashes\n", len(generatedFingerprints))
	if len(extractedPeaks) > 0 {
		fmt.Printf("Fingerprints per Peak    : %.2f hashes/peak\n", float64(len(generatedFingerprints))/float64(len(extractedPeaks)))
	}
	fmt.Println("==================================================")

	if len(generatedFingerprints) > 5 {
		fmt.Println("\nFirst 5 Fingerprints:")
		for i := 0; i < 5; i++ {
			fp := generatedFingerprints[i]
			// Unpack hash for verification
			f1 := (fp.Hash >> 16) & 0xFFFF
			f2 := (fp.Hash >> 8) & 0xFF
			dt := fp.Hash & 0xFF

			fmt.Printf("  FP #%d -> Raw Hash: %10d | Packed (f1: %4d, f2: %3d, dt: %2d) | Anchor TimeFrame: %d\n",
				i+1, fp.Hash, f1, f2, dt, fp.AbsoluteTime)
		}
	}
}
