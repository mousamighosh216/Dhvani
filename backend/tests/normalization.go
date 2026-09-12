package tests

import (
	"Dhvani/recognition/audio"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"
)

func Normalization() {
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

	totalSamples := len(audioData.Samples)
	durationSec := float64(totalSamples) / float64(audioData.SampleRate)

	fmt.Println("==================================================")
	fmt.Printf("Full Track Loaded: %.2f seconds (%d samples at %d Hz)\n", durationSec, totalSamples, audioData.SampleRate)
	fmt.Println("==================================================")

	// Seed random generator
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Test 5 random 3-second cuts
	clipDurationSec := 3.0
	clipSamples := int(clipDurationSec * float64(audioData.SampleRate))
	numTests := 5

	fmt.Printf("\n--- Testing %d Random %.1f-Second Cuts ---\n\n", numTests, clipDurationSec)

	for i := 1; i <= numTests; i++ {
		// Pick a random start sample index (ensuring clip fits in bounds)
		maxStart := totalSamples - clipSamples
		if maxStart <= 0 {
			fmt.Println("Audio file is too short for segment testing.")
			return
		}
		startIndex := r.Intn(maxStart)
		endIndex := startIndex + clipSamples

		// Extract slice
		slice := audioData.Samples[startIndex:endIndex]
		startTimeSec := float64(startIndex) / float64(audioData.SampleRate)
		endTimeSec := float64(endIndex) / float64(audioData.SampleRate)

		// Calculate stats for this cut
		minVal := 1.0
		maxVal := -1.0
		absMax := 0.0
		sumSquare := 0.0

		for _, sample := range slice {
			if sample < minVal {
				minVal = sample
			}
			if sample > maxVal {
				maxVal = sample
			}
			absSample := math.Abs(sample)
			if absSample > absMax {
				absMax = absSample
			}
			sumSquare += sample * sample
		}

		rms := math.Sqrt(sumSquare / float64(len(slice)))

		// Verify normalization bounds
		status := "PASS (Normalized)"
		if absMax > 1.0 {
			status = "FAIL (Clipping / Exceeds 1.0)"
		} else if absMax == 0.0 {
			status = "WARN (Complete Silence)"
		}

		fmt.Printf("Cut #%d [Time: %02d:%02d - %02d:%02d | Start Sec: %.2fs]\n",
			i,
			int(startTimeSec)/60, int(startTimeSec)%60,
			int(endTimeSec)/60, int(endTimeSec)%60,
			startTimeSec,
		)
		fmt.Printf("  Status      : %s\n", status)
		fmt.Printf("  Peak Abs Max: %.6f\n", absMax)
		fmt.Printf("  Min Value   : %.6f\n", minVal)
		fmt.Printf("  Max Value   : %.6f\n", maxVal)
		fmt.Printf("  RMS Loudness: %.6f\n", rms)
		fmt.Println("--------------------------------------------------")
	}
}
