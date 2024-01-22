// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package agabor

import (
	"fmt"
	"log"
	"math"

	"github.com/emer/etable/etable"
	"github.com/emer/etable/etensor"
)

// Filter, a struct of gabor filter parameters
type Filter struct {

	// filter on is default - set to true to exclude filter
	Off bool `desc:"filter on is default - set to true to exclude filter"`

	// wavelength of the sine waves in normalized units, 1.5 and 2 are reasonable values
	WaveLen float64 `desc:"wavelength of the sine waves in normalized units, 1.5 and 2 are reasonable values"`

	// orientation of the gabor in degrees, e.g. 0, 45, 90, 135. Multiple of the same orientation will get evenly distributed with the filter matrix
	Orientation float64 `desc:"orientation of the gabor in degrees, e.g. 0, 45, 90, 135. Multiple of the same orientation will get evenly distributed with the filter matrix"`

	// [def: 0.6] gaussian sigma for the width dimension (in the direction of the sine waves) -- normalized as a function of filter size in relevant dimension
	SigmaWidth float64 `default:"0.6" desc:"gaussian sigma for the width dimension (in the direction of the sine waves) -- normalized as a function of filter size in relevant dimension"`

	// [def: 0.3] gaussian sigma for the length dimension (elongated axis perpendicular to the sine waves) -- normalized as a function of filter size in relevant dimension
	SigmaLength float64 `default:"0.3" desc:"gaussian sigma for the length dimension (elongated axis perpendicular to the sine waves) -- normalized as a function of filter size in relevant dimension"`

	// [def: 0] offset for the sine phase -- default is an asymmetric sine wave -- can make it into a symmetric cosine gabor by using PI/2 = 1.5708
	PhaseOffset float64 `default:"0" desc:"offset for the sine phase -- default is an asymmetric sine wave -- can make it into a symmetric cosine gabor by using PI/2 = 1.5708"`

	// cut off the filter (to zero) outside a circle of diameter filter_size -- makes the filter more radially symmetric - no default - suggest using true
	CircleEdge bool `desc:"cut off the filter (to zero) outside a circle of diameter filter_size -- makes the filter more radially symmetric - no default - suggest using true"`

	// is the gabor circular? orientation, phase, sigmalength and circleedge not used for circular gabor
	Circular bool `desc:"is the gabor circular? orientation, phase, sigmalength and circleedge not used for circular gabor"`
}

// FilterSet, a struct holding a set of gabor filters stored as a tensor. Though individual filters can vary in size, when used as a set they should all have the same size.
type FilterSet struct {

	// size of each filter in X
	SizeX int `desc:"size of each filter in X"`

	// size of each filter in Y
	SizeY int `desc:"size of each filter in Y"`

	// how far to move the filter in X each step
	StrideX int `desc:"how far to move the filter in X each step"`

	// how far to move the filter in Y each step
	StrideY int `desc:"how far to move the filter in Y each step"`

	// overall gain multiplier applied after gabor filtering -- only relevant if not using renormalization (otherwize it just gets renormed away)
	Gain float64 `desc:"overall gain multiplier applied after gabor filtering -- only relevant if not using renormalization (otherwize it just gets renormed away)"`

	// if multiple horiz or vertical distribute evenly
	Distribute bool `desc:"if multiple horiz or vertical distribute evenly"`

	// [view: no-inline] actual gabor filters
	Filters etensor.Float64 `view:"no-inline" desc:"actual gabor filters"`

	// [view: -] simple gabor filter table (view only)
	Table etable.Table `view:"-" desc:"simple gabor filter table (view only)"`
}

// Defaults sets default values for any filter fields where 0 is not a reasonable value
func (f *Filter) Defaults(i int) {
	if f.WaveLen == 0 {
		f.WaveLen = 2
		fmt.Println("filter spec missing value for WaveLen: setting to 2")
	}
	if f.SigmaLength == 0 && f.Circular == false {
		f.SigmaLength = 0.5
		fmt.Println("filter spec missing value for SigmaLength: setting to 0.5")
	}
	if f.SigmaWidth == 0 {
		f.SigmaWidth = 0.5
		fmt.Println("filter spec missing value for SigmaWidth: setting to 0.5")
	}
}

// ToTensor generates filters into the tensor passed by caller
func ToTensor(specs []Filter, set *FilterSet) { // i is filter index in
	active := Active(specs)
	nhf := 0 // number of horizontal filters
	nvf := 0 // number of vertical filters
	if set.Distribute == true {
		for _, f := range active {
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
	radiusX := float64(sx) / 2.0
	radiusY := float64(sy) / 2.0

	// need filter center and count of horizontal and vertical filters to properly distribute them
	ctrX := float64(sx-1) / 2.0
	ctrY := float64(sy-1) / 2.0
	hCtrInc := float64(sy-1) / float64(nhf+1)
	vCtrInc := float64(sx-1) / float64(nvf+1)
	//
	hCnt := 0 // the current count of 0 degree filters generated
	vCnt := 0 // the current count of 90 degree filters generated

	for i, f := range active {
		f.Defaults(i)
		twoPiNorm := (2.0 * math.Pi) / f.WaveLen
		var lNorm float64
		var wNorm float64
		lNorm = 1.0 / (2.0 * f.SigmaLength * f.SigmaLength)
		wNorm = 1.0 / (2.0 * f.SigmaWidth * f.SigmaWidth)

		hPos := float64(0)
		vPos := float64(0)
		if set.Distribute == true {
			if f.Orientation == 0 {
				hPos = hCtrInc * float64(hCnt+1)
				hCnt++
			}
			if f.Orientation == 90 {
				vPos = vCtrInc * float64(vCnt+1)
				vCnt++
			}
		} else {
			hPos = hCtrInc * float64(hCnt+1)
			vPos = vCtrInc * float64(vCnt+1)
		}

		if f.Circular == false {
			for y := 0; y < sy; y++ {
				var xf, yf, xfn, yfn float64
				for x := 0; x < sx; x++ {
					xf = float64(x) - ctrX
					yf = float64(y) - ctrY
					if f.Orientation == 0 {
						yf = float64(y) - float64(hPos)
					}
					if f.Orientation == 90 {
						xf = float64(x) - float64(vPos)
					}
					xfn = xf / radiusX
					yfn = yf / radiusY

					dist := math.Hypot(xfn, yfn)
					val := float64(0)
					if !(f.CircleEdge && dist > 1.0) {
						radians := f.Orientation * math.Pi / 180
						nx := xfn*math.Cos(radians) - yfn*math.Sin(radians)
						ny := yfn*math.Cos(radians) + xfn*math.Sin(radians)
						gauss := math.Exp(-(wNorm*(nx*nx) + lNorm*(ny*ny)))
						sinVal := math.Sin(twoPiNorm*ny + f.PhaseOffset)
						val = gauss * sinVal
					}
					set.Filters.Set([]int{i, y, x}, val)
				}
			}
		} else { // circular
			norm := 1.0 / (2.0 * f.SigmaWidth * f.SigmaWidth)
			for y := 0; y < sy; y++ {
				var xf, yf, xfn, yfn float64
				for x := 0; x < sx; x++ {
					xf = float64(x) - ctrX
					yf = float64(y) - ctrY
					xfn = xf / radiusX
					yfn = yf / radiusY

					val := float64(0)
					nx := xfn * xfn * norm
					ny := yfn * yfn * norm
					gauss := math.Sqrt(float64(nx) + float64(ny))
					sinVal := math.Sin(twoPiNorm * nx * ny)
					val = float64(-gauss * sinVal)
					set.Filters.Set([]int{i, y, x}, val)
				}
			}
		}
	}

	// renorm each half
	for i := 0; i < set.Filters.Dim(0); i++ {
		posSum := 0.0
		negSum := 0.0
		for y := 0; y < sy; y++ {
			for x := 0; x < sx; x++ {
				val := set.Filters.Value([]int{i, y, x})
				if val > 0 {
					posSum += val
				} else if val < 0 {
					negSum += val
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
func Convolve(melData *etensor.Float64, filters FilterSet, rawOut *etensor.Float32, byTime bool) {
	if melData.Dim(1) < filters.SizeX {
		log.Println("Gabor filter width can not be larger than the width of the mel matrix")
		return
	}

	tMax := 1
	fMax := 1
	tMaxStrides := 1
	if rawOut.NumDims() == 2 {
		x := melData.Dim(1) - filters.SizeX
		if x == 0 || x < filters.StrideX {
			// leave tMax equal to 1
		} else {
			tMax = x + 1
		}

		z := melData.Dim(1) - filters.SizeX
		tMaxStrides = z/filters.StrideX + 1

		y := melData.Dim(0) - filters.SizeY
		if y == 0 || y < filters.StrideY {
			// leave fMax equal to 1
		} else {
			fMax = y + 1
		}
	} else if rawOut.NumDims() == 4 {
		tMax1 := rawOut.Shp[1] * filters.StrideX
		tMax2 := melData.Shp[1] - filters.StrideX
		tMax = int(math.Min(float64(tMax1), float64(tMax2)))

		fMax1 := rawOut.Shp[0] * filters.StrideY  // limit frequency strides so we don't overrun the output tensor
		fMax2 := melData.Shp[0] - filters.StrideY // limit strides based on melData in frequency dimension
		fMax = int(math.Min(float64(fMax1), float64(fMax2)))
	} else {
		log.Println("The output tensor should have 2 or 4 dimensions")
		return
	}

	// ToDo: I think this looping could be done with one var instead of two if tMax was tMaxStrides
	// and the filter size was added in the x/y calculation
	// ditto for y
	tIdx := 0
	for t := 0; t < tMax; t, tIdx = t+filters.StrideX, tIdx+1 { // t for time
		fIdx := 0
		for f := 0; f < fMax; f, fIdx = f+filters.StrideY, fIdx+1 { // f for frequency
			nf := filters.Filters.Dim(0)         // number of filters
			for flt := int(0); flt < nf; flt++ { // which filter
				fSum := 0.0
				for ff := int(0); ff < filters.SizeY; ff++ { // size of gabor filter in Y (frequency)
					for ft := int(0); ft < filters.SizeX; ft++ { // size of gabor filter in X (time)
						fVal := filters.Filters.Value([]int{flt, ff, ft})
						iVal := melData.Value([]int{f + ff, t + ft})
						if math.IsNaN(iVal) {
							iVal = .5
						}
						fSum += fVal * iVal
					}
				}
				pos := fSum >= 0.0
				act := filters.Gain * math.Abs(fSum)
				if rawOut.NumDims() == 2 {
					y := fIdx * 2 // we are populating 2 rows, off-center and on-center, thus we need to jump by 2 when populating the output tensor
					x := 0
					if byTime {
						x = tIdx + tMaxStrides*flt
					} else { // default
						x = flt + tIdx*filters.Filters.Dim(0) // tIdx increments for each stride, flt increments stepping through the filters
					}
					if pos {
						rawOut.SetFloat([]int{y, x}, act)
						rawOut.SetFloat([]int{y + 1, x}, 0)
					} else {
						rawOut.SetFloat([]int{y, x}, 0)
						rawOut.SetFloat([]int{y + 1, x}, act)
					}
				} else if rawOut.NumDims() == 4 { // in the 4D case we have pools no need for the multiplication we have in the 2D setting of the output tensor
					if pos {
						rawOut.SetFloat([]int{fIdx, tIdx, 0, flt}, act)
						rawOut.SetFloat([]int{fIdx, tIdx, 1, flt}, 0)
					} else {
						rawOut.SetFloat([]int{fIdx, tIdx, 0, flt}, 0)
						rawOut.SetFloat([]int{fIdx, tIdx, 1, flt}, act)
					}
				} else {
					log.Println("The output tensor should have 2 or 4 dimensions")
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
		{"Filter", etensor.FLOAT64, []int{1, fs.SizeX, fs.SizeY}, []string{"Filter", "Y", "X"}},
	}, n)
	tab.Cols[0].SetFloats(set.Filters.Values)
}

// create reduced set of just the active specs
func Active(specs []Filter) (active []Filter) {
	for _, spec := range specs {
		if spec.Off == false {
			active = append(active, spec)
		}
	}
	return active
}
