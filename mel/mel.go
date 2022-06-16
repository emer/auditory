// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mel

import (
	"log"
	"math"

	"github.com/emer/etable/etensor"
	"gonum.org/v1/gonum/dsp/fourier"
)

// FilterBank contains mel frequency feature bank sampling parameters
type FilterBank struct {
	NFilters    int     `view:"+" def:"32,26" desc:"number of Mel frequency filters to compute"`
	LoHz        float64 `view:"+" def:"120,300" step:"10.0" desc:"low frequency end of mel frequency spectrum"`
	HiHz        float64 `view:"+" def:"10000,8000" step:"1000.0" desc:"high frequency end of mel frequency spectrum -- must be <= sample_rate / 2 (i.e., less than the Nyquist frequencY"`
	LogOff      float64 `view:"+" def:"0" desc:"on add this amount when taking the log of the Mel filter sums to produce the filter-bank output -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs"`
	LogMin      float64 `view:"+" def:"-10" desc:"minimum value a log can produce -- puts a lower limit on log output"`
	Renorm      bool    `desc:" whether to perform renormalization of the mel values"`
	RenormMin   float64 `viewif:"Renorm" step:"1.0" desc:"minimum value to use for renormalization -- you must experiment with range of inputs to determine appropriate values"`
	RenormMax   float64 `viewif:"Renorm" step:"1.0" desc:"maximum value to use for renormalization -- you must experiment with range of inputs to determine appropriate values"`
	RenormScale float64 `view:"-" desc:"1.0 / (ren_max - ren_min)"`
}

// Params
type Params struct {
	FBank  FilterBank `view:"inline"`
	BinPts []int32    `view:"-" desc:" mel scale points in fft bins"`
	HzPts  []float64  `view:"-" desc:" mel scale points in hz"`
	MFCC   bool       `view:"+" def:"false" desc:" compute cepstrum discrete cosine transform (dct) of the mel-frequency filter bank features"`
	Deltas bool       `view:"+" def:"false" desc:" compute the MFCC deltas and delta-deltas"`
	NCoefs int        `viewif:"MFCC" def:"13" desc:" number of mfcc coefficients to output -- typically 1/2 of the number of filterbank features"` // Todo: should be 12 total - 2 - 13, higher ones not useful
}

// Defaults
func (mel *Params) Defaults() {
	mel.MFCC = false
	mel.NCoefs = 13
	mel.FBank.Defaults()
}

// InitFilters computes the filter bin values
func (mel *Params) InitFilters(dftSize int, sampleRate int, filters *etensor.Float64) {
	mel.BinPts = make([]int32, mel.FBank.NFilters+2) // plus 2 because we need end points to create the right number of bins
	mel.HzPts = make([]float64, mel.FBank.NFilters+2)
	mel.FBank.Renorm = false
	if mel.FBank.Renorm == true {
		mel.FBank.RenormScale = 1.0 / (mel.FBank.RenormMax - mel.FBank.RenormMin)
	}

	hiMel := FreqToMel(mel.FBank.HiHz)
	loMel := FreqToMel(mel.FBank.LoHz)
	incr := (hiMel - loMel) / float64(mel.FBank.NFilters+1)

	for i := 0; i < len(mel.BinPts); i++ {
		ml := loMel + float64(i)*incr
		hz := MelToFreq(ml)
		mel.HzPts[i] = hz
		mel.BinPts[i] = int32(FreqToBin(hz, float64(dftSize), float64(sampleRate)))
	}

	maxBins := len(mel.BinPts)
	filters.SetShape([]int{mel.FBank.NFilters, maxBins}, nil, nil)

	for f := 0; f < mel.FBank.NFilters; f++ {
		binMin := int(mel.BinPts[f])
		binCtr := int(mel.BinPts[f+1])
		binMax := int(mel.BinPts[f+2])
		pkmin := float64(binCtr) - float64(binMin)
		pkmax := float64(binMax) - float64(binCtr)

		fi := 0
		bin := 0
		for bin = binMin; bin <= binCtr; bin, fi = bin+1, fi+1 {
			fval := (float64(bin) - float64(binMin)) / pkmin
			filters.SetFloat([]int{f, fi}, float64(fval))
		}
		for ; bin <= binMax; bin, fi = bin+1, fi+1 {
			fval := (float64(binMax) - float64(bin)) / pkmax
			filters.SetFloat([]int{f, fi}, float64(fval))
		}
	}
}

// FilterDft applies the mel filters to power of dft
func (mel *Params) FilterDft(ch, step int, dftPowerOut *etensor.Float64, segmentData *etensor.Float64, fBankData *etensor.Float64, filters *etensor.Float64) {
	mi := 0
	for flt := 0; flt < int(mel.FBank.NFilters); flt, mi = flt+1, mi+1 {
		minBin := mel.BinPts[flt]
		maxBin := mel.BinPts[flt+2]

		sum := 0.0
		fi := 0
		for bin := minBin; bin <= maxBin; bin, fi = bin+1, fi+1 {
			fVal := filters.Value([]int{mi, fi})
			pVal := dftPowerOut.FloatVal1D(int(bin))
			sum += fVal * pVal
		}
		sum += mel.FBank.LogOff
		var val float64
		if sum == 0 {
			val = mel.FBank.LogMin
		} else {
			val = math.Log(sum)
		}
		if mel.FBank.Renorm {
			val -= mel.FBank.RenormMin
			if val < 0.0 {
				val = 0.0
			}
			val *= mel.FBank.RenormScale
			if val > 1.0 {
				val = 1.0
			}
		}
		fBankData.SetFloat1D(mi, val)
		segmentData.Set([]int{step, mi, ch}, val)
	}
}

// FreqToMel converts frequency to mel scale
func FreqToMel(freq float64) float64 {
	return 1127.0 * math.Log(1.0+freq/700.0) // 1127 because we are using natural log
}

// FreqToMel converts mel scale to frequency
func MelToFreq(mel float64) float64 {
	return 700.0 * (math.Exp(mel/1127.0) - 1.0)
}

// FreqToBin converts frequency into FFT bin number, using parameters of number of FFT bins and sample rate
func FreqToBin(freq, nFft, sampleRate float64) int {
	return int(math.Floor(((nFft + 1) * freq) / sampleRate))
}

//Defaults initializes FBank values - these are the ones you most likely need to adjust for your particular signals
func (mfb *FilterBank) Defaults() {
	mfb.LoHz = 0
	mfb.HiHz = 8000.0
	mfb.NFilters = 32
	mfb.LogOff = 0.0
	mfb.LogMin = -10.0
	mfb.Renorm = true
	mfb.RenormMin = -6.0
	mfb.RenormMax = 4.0
}

// FftReal
func (mel *Params) FftReal(out []complex128, in *etensor.Float64) {
	var c complex128
	for i := 0; i < len(out); i++ {
		c = complex(in.FloatVal1D(i), 0)
		out[i] = c
	}
}

// CepstrumDct applies a discrete cosine transform (DCT) to get the cepstrum coefficients on the mel filterbank values
func (mel *Params) CepstrumDct(ch, step int, fBankData *etensor.Float64, mfccSegment *etensor.Float64, mfccDct *etensor.Float64) {
	sz := copy(mfccDct.Values, fBankData.Values)
	if sz != len(mfccDct.Values) {
		log.Printf("mel.CepstrumDctMel: memory copy size wrong")
	}

	dct := fourier.NewDCT(len(mfccDct.Values))
	var mfccOut []float64
	src := []float64{}
	mfccDct.Floats(&src)
	mfccOut = dct.Transform(mfccOut, src)
	el0 := mfccOut[0]
	mfccOut[0] = math.Log(1.0 + el0*el0) // replace with log energy instead..

	// copy only NCoefs
	for i := 0; i < mel.NCoefs; i++ {
		mfccSegment.SetFloat([]int{step, i, ch}, mfccOut[i])
	}

	// calculate deltas
}
