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

func Match() {
	// Load full master track
	file, err := os.Open("testsamples/songsample.mp3")
	if err != nil {
		fmt.Printf("Error opening MP3 file: %v\n", err)
		return
	}
	defer file.Close()

	audioData, err := audio.DecodeMP3(file)
	if err != nil {
		fmt.Printf("Error decoding MP3: %v\n", err)
		return
	}

	windowSize := 2048
	hopSize := 512
	targetZone := fingerprint.DefaultTargetZone()

	fmt.Println("=== 1. INGESTING MASTER TRACK INTO RECOGNITION INDEX ===")
	masterSpec := fft.ComputeSTFT(audioData.Samples, audioData.SampleRate, windowSize, hopSize)
	masterPeaks := peaks.ExtractPeaks(masterSpec, 10, 10, 0.05)

	masterSongID := 101
	masterFingerprints := fingerprint.GenerateFingerprints(masterPeaks, masterSongID, targetZone)

	store := matcher.NewMemoryStore()
	store.IngestTrack(masterFingerprints)

	fmt.Printf("Master Track Ingested -> Song ID: %d | Fingerprints Stored: %d\n", masterSongID, len(masterFingerprints))

	fmt.Println("\n=== 2. EXTRACTING A RANDOM 5-SECOND QUERY CLIP ===")
	clipDurationSec := 5.0
	clipSamplesCount := int(clipDurationSec * float64(audioData.SampleRate))

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	maxStart := len(audioData.Samples) - clipSamplesCount
	randomStartSample := r.Intn(maxStart)

	actualStartSec := float64(randomStartSample) / float64(audioData.SampleRate)
	querySamples := audioData.Samples[randomStartSample : randomStartSample+clipSamplesCount]

	fmt.Printf("Sliced 5-Second Clip starting at exact timestamp: %.2f seconds\n", actualStartSec)

	fmt.Println("\n=== 3. RUNNING RECOGNITION ENGINE ON QUERY CLIP ===")
	querySpec := fft.ComputeSTFT(querySamples, audioData.SampleRate, windowSize, hopSize)
	queryPeaks := peaks.ExtractPeaks(querySpec, 10, 10, 0.05)
	queryFingerprints := fingerprint.GenerateFingerprints(queryPeaks, 0, targetZone)

	fmt.Printf("Query Clip Generated: %d peaks | %d fingerprints\n", len(queryPeaks), len(queryFingerprints))

	// Execute time-offset voting matcher
	matchResult, err := matcher.MatchQuery(queryFingerprints, store, audioData.SampleRate, hopSize)
	if err != nil {
		fmt.Printf("Match error: %v\n", err)
		return
	}

	fmt.Println("\n==================================================")
	fmt.Println("              RECOGNITION MATCH RESULT            ")
	fmt.Println("==================================================")
	fmt.Printf("Matched Song ID   : %d (Expected: %d)\n", matchResult.SongID, masterSongID)
	fmt.Printf("Match Confidence  : %.2f%%\n", matchResult.Confidence*100.0)
	fmt.Printf("Detected Offset   : %.2f seconds (Actual Start: %.2f seconds)\n", matchResult.OffsetSec, actualStartSec)
	fmt.Println("==================================================")

	if matchResult.SongID == masterSongID {
		fmt.Println("SUCCESS: Phase 1 Recognition Engine correctly identified the song and offset!")
	} else {
		fmt.Println("FAIL: Song mismatch or low confidence.")
	}
}
