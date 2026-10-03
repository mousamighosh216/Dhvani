package tests

import (
	"Dhvani/recognition/audio"
	"Dhvani/recognition/fft"
	"Dhvani/recognition/fingerprint"
	"Dhvani/recognition/matcher"
	"Dhvani/recognition/peaks"
	"fmt"
	"math/rand"
	"os"
	"time"
)

func FinalTestDhvani() {
	// Step 1: Load master audio file
	filePath := "testsamples/songsample.mp3"
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening WAV file '%s': %v\n", filePath, err)
		return
	}
	defer file.Close()

	fmt.Println("==================================================")
	fmt.Println("       DHVANI BACKEND RECOGNITION TEST ENGINE     ")
	fmt.Println("==================================================")

	audioData, err := audio.DecodeAudio(file)
	if err != nil {
		fmt.Printf("Error decoding WAV: %v\n", err)
		return
	}

	trackDurationSec := float64(len(audioData.Samples)) / float64(audioData.SampleRate)
	fmt.Printf("Loaded Master File: %s (%.2f seconds | %d Hz)\n\n", filePath, trackDurationSec, audioData.SampleRate)

	// Configuration Parameters
	windowSize := 2048
	hopSize := 512
	targetZone := fingerprint.DefaultTargetZone()

	// -------------------------------------------------------------
	// STEP 2: Ingest Master Track into Memory Store (Database Simulation)
	// -------------------------------------------------------------
	fmt.Println("--> [1/3] Ingesting Master Track into Index...")
	masterSpec := fft.ComputeSTFT(audioData.Samples, audioData.SampleRate, windowSize, hopSize)
	masterPeaks := peaks.ExtractPeaks(masterSpec, 10, 10, 0.05)

	masterSongID := 101
	masterFingerprints := fingerprint.GenerateFingerprints(masterPeaks, masterSongID, targetZone)

	store := matcher.NewMemoryStore()
	store.IngestTrack(masterFingerprints)

	fmt.Printf("    Master Track Ingested successfully!\n")
	fmt.Printf("    Song ID          : %d\n", masterSongID)
	fmt.Printf("    Peaks Extracted  : %d\n", len(masterPeaks))
	fmt.Printf("    Hashes Indexed   : %d\n\n", len(masterFingerprints))

	// -------------------------------------------------------------
	// STEP 3: Extract Random 5-Second Query Slice
	// -------------------------------------------------------------
	fmt.Println("--> [2/3] Simulating User Recording (Random 5s Slice)...")
	clipDurationSec := 5.0
	clipSamplesCount := int(clipDurationSec * float64(audioData.SampleRate))

	if len(audioData.Samples) <= clipSamplesCount {
		fmt.Println("Error: Audio sample is too short to cut a 5-second slice.")
		return
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	maxStartSample := len(audioData.Samples) - clipSamplesCount
	randomStartSample := r.Intn(maxStartSample)

	actualStartSec := float64(randomStartSample) / float64(audioData.SampleRate)
	querySamples := audioData.Samples[randomStartSample : randomStartSample+clipSamplesCount]

	fmt.Printf("    Extracted 5.0s clip starting at exact timestamp: %.2f seconds\n\n", actualStartSec)

	// -------------------------------------------------------------
	// STEP 4: Run Recognition Engine on Query Clip
	// -------------------------------------------------------------
	fmt.Println("--> [3/3] Running Matcher & Time-Offset Voting...")
	querySpec := fft.ComputeSTFT(querySamples, audioData.SampleRate, windowSize, hopSize)
	queryPeaks := peaks.ExtractPeaks(querySpec, 10, 10, 0.05)

	// Query clip uses mock song ID 0
	queryFingerprints := fingerprint.GenerateFingerprints(queryPeaks, 0, targetZone)

	matchResult, err := matcher.MatchQuery(queryFingerprints, store, audioData.SampleRate, hopSize)
	if err != nil {
		fmt.Printf("Error executing recognition matcher: %v\n", err)
		return
	}

	// -------------------------------------------------------------
	// STEP 5: Output Test Results & Verification
	// -------------------------------------------------------------
	fmt.Println("==================================================")
	fmt.Println("              ENGINE RECOGNITION RESULT           ")
	fmt.Println("==================================================")
	fmt.Printf("Matched Song ID   : %d (Expected: %d)\n", matchResult.SongID, masterSongID)
	fmt.Printf("Match Confidence  : %.2f%%\n", matchResult.Confidence*100.0)
	fmt.Printf("Detected Offset   : %.2f seconds\n", matchResult.OffsetSec)
	fmt.Printf("Actual Start Time : %.2f seconds\n", actualStartSec)

	timeDiff := mathAbs(matchResult.OffsetSec - actualStartSec)
	fmt.Printf("Timestamp Delta   : %.3f seconds\n", timeDiff)
	fmt.Println("--------------------------------------------------")

	if matchResult.SongID == masterSongID && timeDiff < 0.25 {
		fmt.Println("STATUS: ✅ TEST PASSED! Dhvani successfully identified the song and offset.")
	} else {
		fmt.Println("STATUS: ❌ TEST FAILED! Song mismatch or timestamp drift detected.")
	}
	fmt.Println("==================================================")
}

func mathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
