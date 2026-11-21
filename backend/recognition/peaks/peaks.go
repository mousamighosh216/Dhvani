// Package peaks picks 2D local maxima from a dB spectrogram.
package peaks

type Peak struct{ T, F int }

// Find returns points that are the maximum of their (2*dT+1) x (2*dF+1) neighbourhood
// and within rangeDB of the loudest point in the spectrogram.
func Find(spec [][]float32, dT, dF int, rangeDB float32) []Peak {
	if len(spec) == 0 {
		return nil
	}
	nT, nF := len(spec), len(spec[0])
	top := spec[0][0]
	for _, row := range spec {
		for _, v := range row {
			if v > top {
				top = v
			}
		}
	}
	floor := top - rangeDB

	// pass 1: max along frequency
	tmp := make([][]float32, nT)
	for t := range spec {
		row := make([]float32, nF)
		for f := range row {
			lo, hi := max(0, f-dF), min(nF-1, f+dF)
			m := spec[t][lo]
			for k := lo + 1; k <= hi; k++ {
				if spec[t][k] > m {
					m = spec[t][k]
				}
			}
			row[f] = m
		}
		tmp[t] = row
	}

	// pass 2: max along time; a peak equals its own neighbourhood max
	var out []Peak
	for t := 0; t < nT; t++ {
		lo, hi := max(0, t-dT), min(nT-1, t+dT)
		for f := 1; f < nF; f++ { // skip DC
			v := spec[t][f]
			if v < floor || v != tmp[t][f] {
				continue
			}
			isMax := true
			for k := lo; k <= hi; k++ {
				if tmp[k][f] > v {
					isMax = false
					break
				}
			}
			if isMax {
				out = append(out, Peak{T: t, F: f})
			}
		}
	}
	return out
}
