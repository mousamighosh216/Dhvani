package peaks

import (
	"math"
	"testing"

	"dhwani/recognition/fft"
)

func flatSpec(frames, bins int, mag float64) *fft.Spectrogram {
	m := make([][]float64, frames)
	for i := range m {
		m[i] = make([]float64, bins)
		for k := range m[i] {
			m[i][k] = mag
		}
	}
	return &fft.Spectrogram{Magnitudes: m, SampleRate: 44100, WindowSize: bins * 2, HopSize: 512}
}

func TestFindsIsolatedPeaks(t *testing.T) {
	s := flatSpec(100, 1024, 0.001)
	s.Magnitudes[20][100] = 10
	s.Magnitudes[70][300] = 8
	got := ExtractPeaks(s, 40)
	if len(got) != 2 {
		t.Fatalf("want 2 peaks, got %+v", got)
	}
	if got[0].TimeFrame != 20 || got[0].FreqBin != 100 || got[1].TimeFrame != 70 || got[1].FreqBin != 300 {
		t.Errorf("wrong peaks: %+v", got)
	}
	if math.Abs(got[0].Magnitude-20) > 1e-6 { // 20*log10(10)
		t.Errorf("magnitude dB = %v", got[0].Magnitude)
	}
}

func TestThresholdDropsQuietPeaks(t *testing.T) {
	s := flatSpec(100, 1024, 1e-6)
	s.Magnitudes[20][100] = 100  // 40 dB
	s.Magnitudes[60][100] = 0.01 // -40 dB: 80 dB below the loud one
	if got := ExtractPeaks(s, 40); len(got) != 1 || got[0].TimeFrame != 20 {
		t.Fatalf("expected only the loud peak, got %+v", got)
	}
}

func TestNeighbourhoodSuppression(t *testing.T) {
	s := flatSpec(100, 1024, 0.001)
	s.Magnitudes[50][200] = 10
	s.Magnitudes[52][203] = 9 // inside the neighbourhood of the bigger peak
	if got := ExtractPeaks(s, 40); len(got) != 1 || got[0].FreqBin != 200 {
		t.Fatalf("got %+v", got)
	}
}

func TestSilenceAndEmpty(t *testing.T) {
	if got := ExtractPeaks(flatSpec(50, 1024, 0), 40); len(got) != 0 {
		t.Errorf("silence gave %d peaks", len(got))
	}
	if ExtractPeaks(nil, 40) != nil || ExtractPeaks(&fft.Spectrogram{}, 40) != nil {
		t.Error("empty input should give nil")
	}
}

func TestBandFilterKeepsLoudestInEachBand(t *testing.T) {
	in := []Peak{
		{0, 5, 10}, {1, 6, 30}, // band [4,12): mean 20 -> keep the 30 dB one
		{2, 100, 50}, {3, 101, 50}, // band [96,192): all at mean -> kept
	}
	got := filterFrequencyBands(in, []int{4, 12, 24, 48, 96, 192, 512})
	if len(got) != 3 || got[0].FreqBin != 6 {
		t.Fatalf("got %+v", got)
	}
}
