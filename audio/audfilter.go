// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package audio

import (
	"fmt"

	"github.com/chewxy/math32"
	"github.com/emer/etable/etensor"
)

// Conv processes input using filters that operate over an entire trial of samples
func Conv(ch int, spec Gabor, input Input, raw etensor.Float32, filters int, melOut etensor.Float32) {
	tHalfSz := spec.SizeTime / 2
	tOff := tHalfSz - input.BorderSteps
	tMin := tOff
	if tMin < 0 {
		tMin = 0
	}
	tMax := input.TrialSteps - tMin

	fMin := int(0)
	//fMax := ap.Mel.MelFBank.NFilters - spec.SizeFreq
	fMax := filters - spec.SizeFreq

	tIdx := 0
	for s := tMin; s < tMax; s, tIdx = s+spec.SpaceTime, tIdx+1 {
		inSt := s - tOff
		if tIdx > raw.Dim(4) {
			fmt.Printf("GaborFilter: time index %v out of range: %v", tIdx, raw.Dim(3))
			break
		}

		fIdx := 0
		for flt := fMin; flt < fMax; flt, fIdx = flt+spec.SpaceFreq, fIdx+1 {
			if fIdx > raw.Dim(3) {
				fmt.Printf("GaborFilter: freq index %v out of range: %v", tIdx, raw.Dim(2))
				break
			}
			nf := spec.NFilters
			for fi := int(0); fi < nf; fi++ {
				fSum := float32(0.0)
				for ff := int(0); ff < spec.SizeFreq; ff++ {
					for ft := int(0); ft < spec.SizeTime; ft++ {
						fVal := spec.Filters.Value([]int{ft, ff, fi})
						//iVal := ap.Mel.MelFBankTrialOut.Value([]int{flt + ff, inSt + ft, ch})
						iVal := melOut.Value([]int{flt + ff, inSt + ft, ch})
						fSum += fVal * iVal
					}
				}
				pos := fSum >= 0.0
				act := spec.Gain * math32.Abs(fSum)
				if pos {
					raw.SetFloat([]int{ch, fi, 0, fIdx, tIdx}, float64(act))
					raw.SetFloat([]int{ch, fi, 1, fIdx, tIdx}, 0)
				} else {
					raw.SetFloat([]int{ch, fi, 0, fIdx, tIdx}, 0)
					raw.SetFloat([]int{ch, fi, 1, fIdx, tIdx}, float64(act))
				}
			}
		}
	}
}
