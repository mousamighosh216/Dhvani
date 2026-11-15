package fingerprint

import (
	"testing"

	"dhwani/recognition/peaks"
)

func TestHashPackingRoundTrip(t *testing.T) {
	cases := [][3]int{{0, 0, 0}, {1023, 1023, 4095}, {1, 2, 3}, {512, 100, 96}, {1023, 0, 1}}
	for _, c := range cases {
		h := CreateHash(c[0], c[1], c[2])
		f1, f2, dt := UnpackHash(h)
		if f1 != c[0] || f2 != c[1] || dt != c[2] {
			t.Errorf("%v -> %#x -> %d %d %d", c, h, f1, f2, dt)
		}
	}
	if CreateHash(1023, 1023, 4095) != 0xFFFFFFFF {
		t.Error("max fields should fill all 32 bits")
	}
	if CreateHash(1, 0, 0) != 1<<22 || CreateHash(0, 1, 0) != 1<<12 || CreateHash(0, 0, 1) != 1 {
		t.Error("bit positions must be 10|10|12")
	}
}

func TestDistinctPairsGiveDistinctHashes(t *testing.T) {
	seen := map[uint32][3]int{}
	for f1 := 0; f1 < 1024; f1 += 37 {
		for f2 := 0; f2 < 1024; f2 += 41 {
			for dt := 1; dt < 100; dt += 7 {
				h := CreateHash(f1, f2, dt)
				if prev, ok := seen[h]; ok {
					t.Fatalf("collision %v %v", prev, [3]int{f1, f2, dt})
				}
				seen[h] = [3]int{f1, f2, dt}
			}
		}
	}
}

func TestGenerateFingerprints(t *testing.T) {
	pk := []peaks.Peak{
		{TimeFrame: 10, FreqBin: 100}, {TimeFrame: 14, FreqBin: 200},
		{TimeFrame: 14, FreqBin: 50}, {TimeFrame: 400, FreqBin: 70}, // far outside any target zone
	}
	fps := GenerateFingerprints(pk, 7)
	// anchor(10,100): targets (14,50),(14,200); anchor (14,50): (14,200) has dt 0 -> skipped
	if len(fps) != 2 {
		t.Fatalf("want 2 fingerprints, got %d: %+v", len(fps), fps)
	}
	for _, fp := range fps {
		if fp.SongID != 7 || fp.SongOffset != 10 {
			t.Errorf("bad fp %+v", fp)
		}
	}
	want := map[uint32]bool{CreateHash(100, 50, 4): true, CreateHash(100, 200, 4): true}
	for _, fp := range fps {
		if !want[fp.Hash] {
			t.Errorf("unexpected hash %#x", fp.Hash)
		}
	}
}

func TestFanOutAndDeterminism(t *testing.T) {
	var pk []peaks.Peak
	for i := 0; i < 50; i++ {
		pk = append(pk, peaks.Peak{TimeFrame: i, FreqBin: (i * 37) % 500})
	}
	a := GenerateWith(pk, 1, Params{FanOut: 3, MinDeltaT: 1, MaxDeltaT: 20})
	b := GenerateWith(pk, 1, Params{FanOut: 3, MinDeltaT: 1, MaxDeltaT: 20})
	if len(a) != 3*49-3-2*0 && len(a) > 3*50 {
		t.Errorf("fan-out exceeded: %d", len(a))
	}
	if len(a) != len(b) {
		t.Fatal("non-deterministic length")
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatal("non-deterministic output")
		}
	}
	if GenerateFingerprints(nil, 1) != nil {
		t.Error("no peaks -> no fingerprints")
	}
}
