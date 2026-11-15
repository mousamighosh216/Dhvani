// Command gensamples writes synthetic songs and degraded query clips so the
// app can be tried without any copyrighted audio.
//
//	go run ./cmd/gensamples -out ../samples
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"dhwani/internal/synth"
)

const sr = 44100

func main() {
	out := flag.String("out", "../samples", "output directory")
	count := flag.Int("songs", 5, "number of songs")
	seconds := flag.Float64("seconds", 60, "song length in seconds")
	flag.Parse()

	songs := filepath.Join(*out, "songs")
	clips := filepath.Join(*out, "clips")
	for _, d := range []string{songs, clips} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			log.Fatal(err)
		}
	}
	for i := 1; i <= *count; i++ {
		x := synth.Song(int64(i), *seconds, sr)
		write(filepath.Join(songs, fmt.Sprintf("song%d.wav", i)), x)
		// Query clips: 6 s taken at 12 s, clean / noisy (10 dB) / quiet (0.2x).
		clip := synth.Clip(x, sr, 12, 6)
		write(filepath.Join(clips, fmt.Sprintf("song%d_clean.wav", i)), clip)
		write(filepath.Join(clips, fmt.Sprintf("song%d_noisy10dB.wav", i)), synth.AddNoise(clip, 10, int64(i)))
		write(filepath.Join(clips, fmt.Sprintf("song%d_quiet.wav", i)), synth.Gain(clip, 0.2))
	}
	// A song that is never ingested: should come back as "no match".
	write(filepath.Join(clips, "unknown_clip.wav"), synth.Clip(synth.Song(9999, 30, sr), sr, 5, 6))
	fmt.Printf("wrote %d songs and %d clips under %s\n", *count, *count*3+1, *out)
}

func write(path string, x []float64) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := synth.WriteWAV(f, x, sr); err != nil {
		log.Fatal(err)
	}
}
