// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package agabor

import (
	"fmt"
	"github.com/emer/etable/etable"
	"github.com/emer/etable/etensor"
	"github.com/goki/mat32"
	"log"
	"math"
)

// Filter, a struct of gabor filter parameters
type Filter struct {
	WaveLen     float32 `desc:"wavelength of the sine waves in normalized units, 1.5 and 2 are reasonable values"`
	Orientation float32 `desc:"orientation of the gabor in degrees, e.g. 0, 45, 90, 135. Multiple of the same orientation will get evenly distributed with the filter matrix"`
	SigmaWidth  float32 `def:"0.6" desc:"gaussian sigma for the width dimension (in the direction of the sine waves) -- normalized as a function of filter size in relevant dimension"`
	SigmaLength float32 `def:"0.3" desc:"gaussian sigma for the length dimension (elongated axis perpendicular to the sine waves) -- normalized as a function of filter size in relevant dimension"`
	PhaseOffset float32 `def:"0" desc:"offset for the sine phase -- default is an asymmetric sine wave -- can make it into a symmetric cosine gabor by using PI/2 = 1.5708"`
	CircleEdge  bool    `desc:"cut off the filter (to zero) outside a circle of diameter filter_size -- makes the filter more radially symmetric - no default - suggest using true"`
	Circular    bool    `desc:"is the gabor circular? orientation, phase, sigmalength and circleedge not used for circular gabor"`
}

// FilterSet, a struct holding a set of gabor filters stored as a tensor. Though individual filters can vary in size, when used as a set they should all have the same size.
type FilterSet struct {
	SizeX      int             `desc:"size of each filter in X"`
	SizeY      int             `desc:"size of each filter in Y"`
	StrideX    int             `desc:"how far to move the filter in X each step"`
	StrideY    int             `desc:"how far to move the filter in Y each step"`
	Gain       float32         `desc:"overall gain multiplier applied after gabor filtering -- only relevant if not using renormalization (otherwize it just gets renormed away)"`
	Distribute bool            `desc:"if multiple horiz or vertical distribute evenly"`
	Filters    etensor.Float64 `view:"no-inline" desc:"actual gabor filters"`
	Table      etable.Table    `view:"no-inline" desc:"simple gabor filter table (view only)"`
}

// Defaults sets default values for any filter fields where 0 is not a reasonable value
func (f *Filter) Defaults(i int) {
	if f.WaveLen == 0 {
		f.WaveLen = 2
		fmt.Println("filter spec missing value for WaveLen: setting to 2")
	}
	if f.SigmaLength == 0 && f.Circular == false {
		f.SigmaLength = 0.6
		fmt.Println("filter spec missing value for SigmaLength: setting to 0.6")
	}
	if f.SigmaWidth == 0 {
		f.SigmaWidth = 0.3
		fmt.Println("filter spec missing value for SigmaWidth: setting to 0.3")
	}
}

// ToTensor generates filters into the tensor passed by caller
func ToTensor(specs []Filter, set *FilterSet) { // i is filter index in
	nhf := 0 // number of horizontal filters
	nvf := 0 // number of vertical filters
	if set.Distribute == true {
		for _, f := range specs {
			if f.Orientation == 0 {
				nhf++
			} else if f.Orientation == 90 {
				nvf++
			}
		}
	} else {
		nhf = 1
		nvf = 1
	}

	sx := set.SizeX
	sy := set.SizeY
	radiusX := float32(sx) / 2.0
	radiusY := float32(sy) / 2.0

	// need filter center and count of horizontal and vertical filters to properly distribute them
	ctrX := float32(sx-1) / 2.0
	ctrY := float32(sy-1) / 2.0
	hCtrInc := float32(sy-1) / float32(nhf+1)
	vCtrInc := float32(sx-1) / float32(nvf+1)
	//
	hCnt := 0 // the current count of 0 degree filters generated
	vCnt := 0 // the current count of 90 degree filters generated
	for i, f := range specs {
		f.Defaults(i)
		twoPiNorm := (2.0 * mat32.Pi) / f.WaveLen
		var lNorm float32
		var wNorm float32
		if f.Orientation == 0 {
			lNorm = 1.0 / (2.0 * f.SigmaLength * f.SigmaLength)
			wNorm = 1.0 / (2.0 * f.SigmaWidth * f.SigmaWidth)
		} else {
			l := f.SigmaLength * float32(sx)
			w := f.SigmaWidth * float32(sy)
			lNorm = 1.0 / (2.0 * l * l)
			wNorm = 1.0 / (2.0 * w * w)
		}

		hPos := float32(0)
		vPos := float32(0)
		if set.Distribute == true {
			if f.Orientation == 0 {
				hPos = hCtrInc * float32(hCnt+1)
				hCnt++
			}
			if f.Orientation == 90 {
				vPos = vCtrInc * float32(vCnt+1)
				vCnt++
			}
		} else {
			hPos = hCtrInc * float32(hCnt+1)
			vPos = vCtrInc * float32(vCnt+1)
		}

		if f.Circular == false {
			for y := 0; y < sy; y++ {
				var xf, yf, xfn, yfn float32
				for x := 0; x < sx; x++ {
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
					val := float64(0)
					if !(f.CircleEdge && dist > 1.0) {
						radians := f.Orientation * math.Pi / 180
						nx := xfn*mat32.Cos(radians) - yfn*mat32.Sin(radians)
						ny := yfn*mat32.Cos(radians) + xfn*mat32.Sin(radians)
						gauss := mat32.Exp(-(wNorm*(nx*nx) + lNorm*(ny*ny)))
						sinVal := mat32.Sin(twoPiNorm*ny + f.PhaseOffset)
						val = float64(gauss * sinVal)
					}
					set.Filters.Set([]int{i, y, x}, val)
				}
			}
		} else { // circular
			norm := 1.0 / (2.0 * f.SigmaWidth * f.SigmaWidth)
			for y := 0; y < sy; y++ {
				var xf, yf, xfn, yfn float32
				for x := 0; x < sx; x++ {
					xf = float32(x) - ctrX
					yf = float32(y) - ctrY
					xfn = xf / radiusX
					yfn = yf / radiusY

					val := float64(0)
					nx := xfn * xfn * norm
					ny := yfn * yfn * norm
					gauss := mat32.Sqrt(float32(nx) + float32(ny))
					sinVal := mat32.Sin(twoPiNorm * nx * ny)
					val = float64(-gauss * sinVal)
					set.Filters.Set([]int{i, y, x}, val)
				}
			}
		}
	}

	// renorm each half
	for i := 0; i < set.Filters.Dim(0); i++ {
		posSum := float64(0)
		negSum := float64(0)
		for y := 0; y < sy; y++ {
			for x := 0; x < sx; x++ {
				val := float32(set.Filters.Value([]int{i, y, x}))
				if val > 0 {
					posSum += float64(val)
				} else if val < 0 {
					negSum += float64(val)
				}
			}
		}
		posNorm := 1.0 / posSum
		negNorm := -1.0 / negSum
		for y := 0; y < sy; y++ {
			for x := 0; x < sx; x++ {
				val := set.Filters.Value([]int{i, y, x})
				if val > 0.0 {
					val *= posNorm
				} else if val < 0.0 {
					val *= negNorm
				}
				set.Filters.Set([]int{i, y, x}, val)
			}
		}
	}
}

// Convolve processes input using filters that operate over an entire segment of samples
func Convolve(ch int, melData *etensor.Float32, filters FilterSet, rawOut *etensor.Float32) {
	if melData.Dim(0) < filters.SizeX {
		log.Println("Gabor filter width can not be larger than the width of the mel matrix")
		return
	}

	tMax := 1
	fMax := 1
	if rawOut.NumDims() == 3 {
		x := melData.Dim(0) - filters.SizeX
		if x == 0 || x < filters.StrideX {
			// leave tMax equal to 1
		} else {
			tMax = x + 1
		}
		y := melData.Dim(1) - filters.SizeY
		if y == 0 || y < filters.StrideY {
			// leave fMax equal to 1
		} else {
			fMax = y + 1
		}
	} else if rawOut.NumDims() == 5 {
		tMax1 := rawOut.Shp[2] * filters.StrideX
		tMax2 := melData.Shp[0] - filters.StrideX
		tMax = int(mat32.Min32i(int32(tMax1), int32(tMax2)))

		fMax1 := rawOut.Shp[1] * filters.StrideY  // limit frequency strides so we don't overrun the output tensor
		fMax2 := melData.Shp[1] - filters.StrideY // limit strides based on melData in frequency dimension
		fMax = int(mat32.Min32i(int32(fMax1), int32(fMax2)))
	} else {
		log.Println("The output tensor should have 3 or 5 dimensions (1 for number of channels plus 2 or 4 for 2D or 4D result")
		return
	}

	tIdx := 0
	for t := 0; t < tMax; t, tIdx = t+filters.StrideX, tIdx+1 { // t for time
		fIdx := 0
		for f := 0; f < fMax; f, fIdx = f+filters.StrideY, fIdx+1 { // f for frequency
			nf := filters.Filters.Dim(0)         // number of filters
			for flt := int(0); flt < nf; flt++ { // which filter
				fSum := float32(0.0)
				for ff := int(0); ff < filters.SizeY; ff++ { // size of gabor filter in Y (frequency)
					for ft := int(0); ft < filters.SizeX; ft++ { // size of gabor filter in X (time)
						fVal := filters.Filters.Value([]int{flt, ff, ft})
						iVal := float64(melData.Value([]int{t + ft, f + ff, ch}))
						if math.IsNaN(iVal) {
							iVal = .5
						}
						fSum += float32(fVal * iVal)
					}
				}
				pos := fSum >= 0.0
				act := filters.Gain * mat32.Abs(fSum)
				if rawOut.NumDims() == 3 {
					y := fIdx * 2 // we are populating 2 rows, off-center and on-center, thus we need to jump by 2 when populating the output tensor
					if pos {
						rawOut.SetFloat([]int{ch, y, flt}, float64(act))
						rawOut.SetFloat([]int{ch, y + 1, flt}, 0)
					} else {
						rawOut.SetFloat([]int{ch, y, flt}, 0)
						rawOut.SetFloat([]int{ch, y + 1, flt}, float64(act))
					}
				} else if rawOut.NumDims() == 5 { // in the 4D case we have pools no need for the multiplication we have in the 2D setting of the output tensor
					if pos {
						rawOut.SetFloat([]int{ch, fIdx, tIdx, 0, flt}, float64(act))
						rawOut.SetFloat([]int{ch, fIdx, tIdx, 1, flt}, 0)
					} else {
						rawOut.SetFloat([]int{ch, fIdx, tIdx, 0, flt}, 0)
						rawOut.SetFloat([]int{ch, fIdx, tIdx, 1, flt}, float64(act))
					}
				} else {
					log.Println("The output tensor should have 3 or 5 dimensions (1 for number of channels plus 2 or 4 for 2D or 4D result")
				}
			}
		}
	}
}

// ToDo: don't renorm
// ToTable renders filters into the given etable.Table
// This is useful for display and validation purposes.
func (fs *FilterSet) ToTable(set FilterSet, tab *etable.Table) {
	n := fs.Filters.Dim(0)
	tab.SetFromSchema(etable.Schema{
		{"Filter", etensor.FLOAT32, []int{1, fs.SizeX, fs.SizeY}, []string{"Filter", "Y", "X"}},
	}, n)
	tab.Cols[0].SetFloats(set.Filters.Values)
}
