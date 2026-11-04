package main

import (
	"dhwani/recognition/audio"
	"fmt"
	"log"
	"os"
)

func main() {
	// addr := flag.String("addr", ":8080", "HTTP network address")
	// flag.Parse()

	// log.Printf("Server started on %s", *addr)
	// router := api.NewRouter()
	// http.ListenAndServe(*addr, router)

	// Example usage of the audio decoder
	filePath := "testaudio/wavwav.wav"
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file '%s': %v\n", filePath, err)
		return
	}
	defer file.Close()

	audioData, err := audio.Decoder(file)
	if err != nil {
		log.Fatalf("Decode failed: %v", err)
	}

	// Print metadata
	fmt.Printf("--- WAV Metadata ---\n")
	fmt.Printf("Sample Rate: %d Hz\n", audioData.SampleRate)
	fmt.Printf("Total Samples Extracted: %d\n\n", len(audioData.Samples))

	// Print the first 50 non-silent samples to see the wave coordinates
	fmt.Printf("--- First 50 Normalized Float Samples ---\n")
	printedCount := 0
	for i, sample := range audioData.Samples {
		// Skip absolute digital silence at the very beginning of the track
		if sample == 0.0 {
			continue
		}
		fmt.Printf("Index [%d]: %.6f\n", i, sample)
		printedCount++
		if printedCount >= 50 {
			break
		}
	}
}
