// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package agabor

import (
	"github.com/emer/etable/etensor"
	"github.com/goki/mat32"
)

// Gabor params for auditory gabor filters: 2d Gaussian envelope times a sinusoidal plane wave --
// by default produces 2 phase asymmetric edge detector filters -- horizontal tuning is different from V1 version --
// has elongated frequency-band specific tuning, not a parallel horizontal tuning -- and has multiple of these
type Params struct {
	On              bool    `desc:"use this gabor filtering of the time-frequency space filtered input (time in terms of steps of the DFT transform, and discrete frequency factors based on the FFT window and input sample rate)"`
	TimeSize        int     `viewif:"On" def:"6,8,12,16,24" desc:" size of the filter in the time (horizontal) domain, in terms of steps of the underlying DFT filtering steps"`
	FreqSize        int     `viewif:"On" def:"6,8,12,16,24" desc:" size of the filter in the frequency domain, in terms of discrete frequency factors based on the FFT window and input sample rate"`
	TimeStride      int     `viewif:"On" desc:" spacing in the time (horizontal) domain, in terms of steps"`
	FreqStride      int     `viewif:"On" desc:" spacing in the frequency (vertical) domain"`
	WaveLen         float32 `viewif:"On" def:"1.5,2" desc:"wavelength of the sine waves in normalized units"`
	SigmaLen        float32 `viewif:"On" def:"0.6" desc:"gaussian sigma for the length dimension (elongated axis perpendicular to the sine waves) -- normalized as a function of filter size in relevant dimension"`
	SigmaWidth      float32 `viewif:"On" def:"0.3" desc:"gaussian sigma for the width dimension (in the direction of the sine waves) -- normalized as a function of filter size in relevant dimension"`
	HorizSigmaLen   float32 `viewif:"On" def:"0.3" desc:"gaussian sigma for the length of special horizontal narrow-band filters -- normalized as a function of filter size in relevant dimension"`
	HorizSigmaWidth float32 `viewif:"On" def:"0.1" desc:"gaussian sigma for the horizontal dimension for special horizontal narrow-band filters -- normalized as a function of filter size in relevant dimension"`
	Gain            float32 `viewif:"On" def:"2" desc:"overall gain multiplier applied after gabor filtering -- only relevant if not using renormalization (otherwize it just gets renormed awaY"`
	NHoriz          int     `viewif:"On" def:"4" desc:"number of horizontally-elongated,  pure time-domain, frequency-band specific filters to include, evenly spaced over the available frequency space for this filter set -- in addition to these, there are two diagonals (45, 135) and a vertically-elongated (wide frequency band) filter"`
	NAng            int     `viewif:"On" def:"3" desc:"number of time-domain / frequency-band filters, typically there are two diagonals (45, 135) and a vertically-elongated (wide frequency band) filter"`
	PhaseOffset     float32 `viewif:"On" def:"0,1.5708" desc:"offset for the sine phase -- default is an asymmetric sine wave -- can make it into a symmetric cosine gabor by using PI/2 = 1.5708"`
	CircleEdge      bool    `viewif:"On" def:"true" desc:"cut off the filter (to zero) outside a circle of diameter filter_size -- makes the filter more radially symmetric"`
	NFilters        int     `viewif:"On" desc:" total number of filters = 3 + NHoriz"`
}

//Initialize initializes the Gabor
func (ga *Params) Defaults() {
	ga.On = true
	ga.Gain = 2.0
	ga.NHoriz = 4
	ga.NAng = 3
	ga.TimeSize = 6.0
	ga.FreqSize = 6.0
	ga.TimeStride = 2.0
	ga.FreqStride = 2.0
	ga.WaveLen = 6.0
	ga.SigmaLen = 0.6
	ga.SigmaWidth = 0.3
	ga.HorizSigmaLen = 0.3
	ga.HorizSigmaWidth = 0.2
	ga.PhaseOffset = 0.0
	ga.CircleEdge = true
	ga.NFilters = ga.NAng + ga.NHoriz // 3 is number of angle filters
}

// RenderFilters generates filters into the given matrix, which is formatted as: [ga.TimeSize_steps][ga.FreqSize][n_filters]
func (ga *Params) RenderFilters(filters *etensor.Float32) {
	ctrTime := (float32(ga.TimeSize) - 1) / 2.0
	ctrFreq := (float32(ga.FreqSize) - 1) / 2.0
	angInc := mat32.Pi / 4.0
	radiusTime := float32(ga.TimeSize / 2.0)
	radiusFreq := float32(ga.FreqSize / 2.0)

	//gs_len_eff := ga.SigmaLen * float32(ga.TimeSize)
	//gs_wd_eff := ga.SigmaWidth * float32(ga.FreqSize)
	//lenNorm := 1.0 / (2.0 * gs_len_eff * gs_len_eff)
	//widthNorm := 1.0 / (2.0 * gs_wd_eff * gs_wd_eff)
	lenNorm := 1.0 / (2.0 * ga.SigmaLen * ga.SigmaLen)
	widthNorm := 1.0 / (2.0 * ga.SigmaWidth * ga.SigmaWidth)

	lenHorizNorm := 1.0 / (2.0 * ga.HorizSigmaLen * ga.HorizSigmaLen)
	widthHorizNorm := 1.0 / (2.0 * ga.HorizSigmaWidth * ga.HorizSigmaWidth)

	twoPiNorm := (2.0 * mat32.Pi) / ga.WaveLen
	hCtrInc := (ga.FreqSize - 1) / (ga.NHoriz + 1)

	fli := 0
	for hi := 0; hi < ga.NHoriz; hi, fli = hi+1, fli+1 {
		hCtrFreq := hCtrInc * (hi + 1)
		angF := float32(-2.0 * angInc)
		for y := 0; y < ga.FreqSize; y++ {
			var xf, yf, xfn, yfn float32
			for x := 0; x < ga.TimeSize; x++ {
				xf = float32(x) - ctrTime
				yf = float32(y) - float32(hCtrFreq)
				xfn = xf / radiusTime
				yfn = yf / radiusFreq

				dist := mat32.Hypot(xfn, yfn)
				val := float32(0)
				if !(ga.CircleEdge && dist > 1.0) {
					nx := xfn*mat32.Cos(angF) - yfn*mat32.Sin(angF)
					ny := yfn*mat32.Cos(angF) + xfn*mat32.Sin(angF)
					gauss := mat32.Exp(-(widthHorizNorm*(nx*nx) + lenHorizNorm*(ny*ny)))
					sinVal := mat32.Sin(twoPiNorm*ny + ga.PhaseOffset)
					val = gauss * sinVal
				}
				filters.Set([]int{fli, y, x}, val)
			}
		}
	}

	// fli should be ga.Horiz - 1 at this point
	for ang := 1; ang < ga.NAng+1; ang, fli = ang+1, fli+1 {
		angF := float32(-ang) * float32(angInc)
		var xf, yf, xfn, yfn float32
		for y := 0; y < ga.FreqSize; y++ {
			for x := 0; x < ga.TimeSize; x++ {
				xf = float32(x) - ctrTime
				yf = float32(y) - ctrFreq
				xfn = xf / radiusTime
				yfn = yf / radiusFreq

				dist := mat32.Hypot(xfn, yfn)
				val := float32(0)
				if !(ga.CircleEdge && dist > 1.0) {
					nx := xfn*mat32.Cos(angF) - yfn*mat32.Sin(angF)
					ny := yfn*mat32.Cos(angF) + xfn*mat32.Sin(angF)
					gauss := mat32.Exp(-(lenNorm*(nx*nx) + widthNorm*(ny*ny)))
					sinVal := mat32.Sin(twoPiNorm*ny + ga.PhaseOffset)
					val = gauss * sinVal
				}
				filters.Set([]int{fli, y, x}, val)
			}
		}
	}

	// renorm each half
	for fli := 0; fli < ga.NFilters; fli++ {
		posSum := float32(0)
		negSum := float32(0)
		for y := 0; y < ga.FreqSize; y++ {
			for x := 0; x < ga.TimeSize; x++ {
				val := float32(filters.Value([]int{fli, y, x}))
				if val > 0 {
					posSum += val
				} else if val < 0 {
					negSum += val
				}
			}
		}
		posNorm := 1.0 / posSum
		negNorm := -1.0 / negSum
		for y := 0; y < ga.FreqSize; y++ {
			for x := 0; x < ga.TimeSize; x++ {
				val := filters.Value([]int{fli, y, x})
				if val > 0.0 {
					val *= posNorm
				} else if val < 0.0 {
					val *= negNorm
				}
				filters.Set([]int{fli, y, x}, val)
			}
		}
	}
}

// Conv processes input using filters that operate over an entire segment of samples
func Conv(ch int, gbor Params, segmentSteps int, borderSteps int, rawOut *etensor.Float32, melFilterCount int, gborFilters *etensor.Float32, melData *etensor.Float32) {
	//tOffset := gbor.TimeSize/2 - borderSteps
	//tMin := tOffset
	//if tMin < 0 {
	//	tMin = 0
	//}

	// just set tMin to zero - any offset is handled by the calling code
	tMin := 0

	tMax := rawOut.Shp[2] * gbor.TimeStride

	fMin := 0
	fMax := rawOut.Shp[1] * gbor.FreqStride

	tIdx := 0
	for s := tMin; s < tMax; s, tIdx = s+gbor.TimeStride, tIdx+1 {
		fIdx := 0
		for flt := fMin; flt < fMax; flt, fIdx = flt+gbor.FreqStride, fIdx+1 {
			nf := gbor.NFilters
			for fi := int(0); fi < nf; fi++ {
				fSum := float32(0.0)
				for ff := int(0); ff < gbor.FreqSize; ff++ {
					for ft := int(0); ft < gbor.TimeSize; ft++ {
						fVal := gborFilters.Value([]int{fi, ff, ft})
						iVal := melData.Value([]int{s + ft, flt + ff, ch})
						if mat32.IsNaN(iVal) {
							iVal = .5
						}
						fSum += fVal * iVal
					}
				}
				pos := fSum >= 0.0
				act := gbor.Gain * mat32.Abs(fSum)
				if pos {
					rawOut.SetFloat([]int{ch, fIdx, tIdx, 0, fi}, float64(act))
					rawOut.SetFloat([]int{ch, fIdx, tIdx, 1, fi}, 0)
				} else {
					rawOut.SetFloat([]int{ch, fIdx, tIdx, 0, fi}, 0)
					rawOut.SetFloat([]int{ch, fIdx, tIdx, 1, fi}, float64(act))
				}
			}
		}
	}
}
