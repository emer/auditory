// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package agabor

import (
	"fmt"
	"github.com/emer/etable/etensor"
	"github.com/goki/mat32"
	"math"
)

// Filter, a struct of gabor filter parameters
type Filter struct {
	SizeX       int     `desc:"size of the filter in X, for audition this is the time (horizontal) domain, in terms of steps of the underlying DFT filtering steps"`
	SizeY       int     `desc:"size of the filter in Y, for audition this is the frequency domain, in terms of discrete frequency factors based on the FFT window and input sample rate"`
	WaveLen     float32 `desc:"wavelength of the sine waves in normalized units, 1.5 and 2 are reasonable values"`
	Orientation float32 `desc:"orientation of the gabor in degrees, e.g. 0, 45, 90, 135. Multiple of the same orientation will get evenly distributed with the filter matrix"`
	SigmaLength float32 `def:"0.3" desc:"gaussian sigma for the length dimension (elongated axis perpendicular to the sine waves) -- normalized as a function of filter size in relevant dimension"`
	SigmaWidth  float32 `def:"0.6" desc:"gaussian sigma for the width dimension (in the direction of the sine waves) -- normalized as a function of filter size in relevant dimension"`
	PhaseOffset float32 `def:"0" desc:"offset for the sine phase -- default is an asymmetric sine wave -- can make it into a symmetric cosine gabor by using PI/2 = 1.5708"`
	CircleEdge  bool    `desc:"cut off the filter (to zero) outside a circle of diameter filter_size -- makes the filter more radially symmetric - no default - suggest using true"`
}

// FilterSet, a struct holding a set of gabor filters stored as a tensor. Though individual filters can vary in size, when used as a set they should all have the same size.
type FilterSet struct {
	SizeX   int             `desc:"size of each filter in X"`
	SizeY   int             `desc:"size of each filter in Y"`
	Filters etensor.Float32 `desc:"actual gabor filters"`
}

// Defaults sets default values for any filter fields where 0 is not a reasonable value
func (f *Filter) Defaults() {
	if f.WaveLen == 0 {
		f.WaveLen = 2
		fmt.Println("filter spec missing value for WaveLen: setting to 2")
	}
	if f.SigmaLength == 0 {
		f.SigmaLength = 0.6
		fmt.Println("filter spec missing value for SigmaLength: setting to 0.6")
	}
	if f.SigmaWidth == 0 {
		f.SigmaWidth = 0.3
		fmt.Println("filter spec missing value for SigmaWidth: setting to 0.3")
	}
}

// ToTensor generates filters into the tensor passed by caller
func ToTensor(specs []Filter, filters *etensor.Float32) { // i is filter index in tensor
	nhf := 0 // number of horizontal filters
	nvf := 0 // number of vertical filters
	for _, f := range specs {
		if f.Orientation == 0 {
			nhf++
		} else if f.Orientation == 90 {
			nvf++
		}
	}

	sizeX := specs[0].SizeX
	sizeY := specs[0].SizeY
	radiusX := float32(sizeX / 2.0)
	radiusY := float32(sizeY / 2.0)

	// need filter center and count of horizontal and vertical filters to properly distribute them
	ctrX := (float32(sizeX - 1)) / 2.0
	ctrY := (float32(sizeY - 1)) / 2.0
	hCtrInc := (sizeY - 1) / (nhf + 1)
	vCtrInc := (sizeX - 1) / (nvf + 1)

	hCnt := 0 // the current count of 0 degree filters generated
	vCnt := 0 // the current count of 90 degree filters generated
	for i, f := range specs {
		f.Defaults()
		twoPiNorm := (2.0 * mat32.Pi) / f.WaveLen
		var lNorm float32
		var wNorm float32
		if f.Orientation == 0 {
			lNorm = 1.0 / (2.0 * f.SigmaLength * f.SigmaLength)
			wNorm = 1.0 / (2.0 * f.SigmaWidth * f.SigmaWidth)
		} else {
			l := f.SigmaLength * float32(f.SizeX)
			w := f.SigmaWidth * float32(f.SizeY)
			lNorm = 1.0 / (2.0 * l * l)
			wNorm = 1.0 / (2.0 * w * w)
		}

		hPos := 0
		vPos := 0
		if f.Orientation == 0 {
			hPos = hCtrInc * (hCnt + 1)
			hCnt++
		}
		if f.Orientation == 90 {
			vPos = vCtrInc * (vCnt + 1)
			vCnt++
		}

		for y := 0; y < f.SizeY; y++ {
			var xf, yf, xfn, yfn float32
			for x := 0; x < f.SizeX; x++ {
				xf = float32(x) - ctrX
				yf = float32(y) - ctrY
				if f.Orientation == 0 {
					yf = float32(y) - float32(hPos)
				}
				if f.Orientation == 90 {
					xf = float32(x) - float32(vPos)
				}
				xfn = xf / radiusX
				yfn = yf / radiusY

				dist := mat32.Hypot(xfn, yfn)
				val := float32(0)
				if !(f.CircleEdge && dist > 1.0) {
					radians := f.Orientation * math.Pi / 180
					nx := xfn*mat32.Cos(radians) - yfn*mat32.Sin(radians)
					ny := yfn*mat32.Cos(radians) + xfn*mat32.Sin(radians)
					gauss := mat32.Exp(-(wNorm*(nx*nx) + lNorm*(ny*ny)))
					sinVal := mat32.Sin(twoPiNorm*ny + f.PhaseOffset)
					val = gauss * sinVal
				}
				filters.Set([]int{i, y, x}, val)
			}
		}
	}

	// renorm each half
	for i := 0; i < filters.Dim(0); i++ {
		posSum := float32(0)
		negSum := float32(0)
		for y := 0; y < sizeY; y++ {
			for x := 0; x < sizeX; x++ {
				val := float32(filters.Value([]int{i, y, x}))
				if val > 0 {
					posSum += val
				} else if val < 0 {
					negSum += val
				}
			}
		}
		posNorm := 1.0 / posSum
		negNorm := -1.0 / negSum
		for y := 0; y < sizeY; y++ {
			for x := 0; x < sizeX; x++ {
				val := filters.Value([]int{i, y, x})
				if val > 0.0 {
					val *= posNorm
				} else if val < 0.0 {
					val *= negNorm
				}
				filters.Set([]int{i, y, x}, val)
			}
		}
	}
}

// Convolve processes input using filters that operate over an entire segment of samples
func Convolve(ch int, segmentSteps int, borderSteps int, melFilterCount int, melData *etensor.Float32, filters FilterSet, strideX int, strideY int, gain float32, rawOut *etensor.Float32) {
	// just set tMin to zero - any offset should be handled by the calling code
	tMin := 0
	tMax1 := rawOut.Shp[2] * strideX
	tMax2 := melData.Shp[0] - strideX - 1
	tMax := int(mat32.Min32i(int32(tMax1), int32(tMax2)))

	fMin := 0
	fMax1 := rawOut.Shp[1] * strideY      // limit frequency strides so we don't overrun the output tensor
	fMax2 := melData.Shp[1] - strideY - 1 // limit strides based on melData in frequency dimension
	fMax := int(mat32.Min32i(int32(fMax1), int32(fMax2)))

	tIdx := 0
	for s := tMin; s < tMax; s, tIdx = s+strideX, tIdx+1 {
		fIdx := 0
		for flt := fMin; flt < fMax; flt, fIdx = flt+strideY, fIdx+1 {
			nf := filters.Filters.Dim(0)
			for fi := int(0); fi < nf; fi++ {
				fSum := float32(0.0)
				for ff := int(0); ff < filters.SizeY; ff++ {
					for ft := int(0); ft < filters.SizeX; ft++ {
						fVal := filters.Filters.Value([]int{fi, ff, ft})
						iVal := melData.Value([]int{s + ft, flt + ff, ch})
						if mat32.IsNaN(iVal) {
							iVal = .5
						}
						fSum += fVal * iVal
					}
				}
				pos := fSum >= 0.0
				act := gain * mat32.Abs(fSum)
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
