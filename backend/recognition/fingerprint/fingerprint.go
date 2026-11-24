// Package fingerprint pairs anchor and target peaks into 32-bit hashes.
package fingerprint

import (
	"sort"

	"dhwani/recognition/peaks"
)

const (
	MaxDT  = 96 // max frames between anchor and target
	FanOut = 10 // targets per anchor
)

type Fingerprint struct {
	Hash uint32 // f1 (10 bits) | f2 (10 bits) | dt (12 bits)
	T    int    // anchor frame
}

func Pack(f1, f2, dt int) uint32 {
	return uint32(f1&0x3FF)<<22 | uint32(f2&0x3FF)<<12 | uint32(dt&0xFFF)
}

func Generate(ps []peaks.Peak) []Fingerprint {
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].T != ps[j].T {
			return ps[i].T < ps[j].T
		}
		return ps[i].F < ps[j].F
	})
	var out []Fingerprint
	for i, a := range ps {
		n := 0
		for j := i + 1; j < len(ps) && n < FanOut; j++ {
			dt := ps[j].T - a.T
			if dt > MaxDT {
				break
			}
			if dt < 1 {
				continue
			}
			out = append(out, Fingerprint{Hash: Pack(a.F, ps[j].F, dt), T: a.T})
			n++
		}
	}
	return out
}
