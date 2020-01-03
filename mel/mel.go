// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mel

import (
	"log"
	"math"

	"github.com/chewxy/math32"
	"github.com/emer/etable/etensor"
	"gonum.org/v1/gonum/fourier"
)

// FilterBank contains mel frequency feature bank sampling parameters
type FilterBank struct {
	NFilters    int     `viewif:"On" def:"32,26" desc:"number of Mel frequency filters to compute"`
	LoHz        float32 `viewif:"On" def:"120,300" step:"10.0" desc:"low frequency end of mel frequency spectrum"`
	HiHz        float32 `viewif:"On" def:"10000,8000" step:"1000.0" desc:"high frequency end of mel frequency spectrum -- must be <= sample_rate / 2 (i.e., less than the Nyquist frequencY"`
	LogOff      float32 `viewif:"On" def:"0" desc:"on add this amount when taking the log of the Mel filter sums to produce the filter-bank output -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs"`
	LogMin      float32 `viewif:"On" def:"-10" desc:"minimum value a log can produce -- puts a lower limit on log output"`
	Renorm      bool    `desc:" whether to perform renormalization of the mel values"`
	RenormMin   float32 `viewif:"On" step:"1.0" desc:"minimum value to use for renormalization -- you must experiment with range of inputs to determine appropriate values"`
	RenormMax   float32 `viewif:"On" step:"1.0" desc:"maximum value to use for renormalization -- you must experiment with range of inputs to determine appropriate values"`
	RenormScale float32 `inactive:"+" desc:"1.0 / (ren_max - ren_min)"`
}

// Params
type Params struct {
	FBank      FilterBank
	PtBins     etensor.Int32 `view:"no-inline" desc:" mel scale points in fft bins"`
	CompMfcc   bool          `desc:" compute cepstrum discrete cosine transform (dct) of the mel-frequency filter bank features"`
	MfccNCoefs int           `def:"13" desc:" number of mfcc coefficients to output -- typically 1/2 of the number of filterbank features"` // Todo: should be 12 total - 2 - 13, higher ones not useful
}

// Defaults
func (mel *Params) Defaults() {
	mel.CompMfcc = false
	mel.MfccNCoefs = 13
	mel.FBank.Defaults()
}

// InitFilters computes the filter bin values
func (mel *Params) InitFilters(dftSize int, sampleRate int, filters *etensor.Float32) {
	mel.FBank.RenormScale = 1.0 / (mel.FBank.RenormMax - mel.FBank.RenormMin)

	hiMel := FreqToMel(mel.FBank.HiHz)
	loMel := FreqToMel(mel.FBank.LoHz)
	nFiltersEff := mel.FBank.NFilters + 2
	mel.PtBins.SetShape([]int{nFiltersEff}, nil, nil)
	melIncr := (hiMel - loMel) / float32(mel.FBank.NFilters+1)

	for i := 0; i < nFiltersEff; i++ {
		ml := loMel + float32(i)*melIncr
		hz := MelToFreq(ml)
		bin := FreqToBin(hz, float32(dftSize), float32(sampleRate))
		mel.PtBins.SetFloat1D(i, float64(bin))
	}

	maxBins := int(mel.PtBins.Value1D(nFiltersEff-1)) - int(mel.PtBins.Value1D(nFiltersEff-3)) + 1
	filters.SetShape([]int{mel.FBank.NFilters, maxBins}, nil, nil)

	for flt := 0; flt < mel.FBank.NFilters; flt++ {
		mnbin := int(mel.PtBins.Value1D(flt))
		pkbin := int(mel.PtBins.Value1D(flt + 1))
		mxbin := int(mel.PtBins.Value1D(flt + 2))
		pkmin := float32(pkbin) - float32(mnbin)
		pkmax := float32(mxbin) - float32(pkbin)

		fi := 0
		bin := 0
		for bin = mnbin; bin <= pkbin; bin, fi = bin+1, fi+1 {
			fval := (float32(bin) - float32(mnbin)) / pkmin
			filters.SetFloat([]int{flt, fi}, float64(fval))
		}
		for ; bin <= mxbin; bin, fi = bin+1, fi+1 {
			fval := (float32(mxbin) - float32(bin)) / pkmax
			filters.SetFloat([]int{flt, fi}, float64(fval))
		}
	}
}

// FilterDft applies the mel filters to power of dft
func (mel *Params) FilterDft(ch, step int, dftPowerOut etensor.Float32, segmentData *etensor.Float32, fBankData *etensor.Float32, filters *etensor.Float32) {
	mi := 0
	for flt := 0; flt < int(mel.FBank.NFilters); flt, mi = flt+1, mi+1 {
		minBin := mel.PtBins.Value1D(flt)
		maxBin := mel.PtBins.Value1D(flt + 2)

		sum := float32(0)
		fi := 0
		for bin := minBin; bin <= maxBin; bin, fi = bin+1, fi+1 {
			fVal := filters.Value([]int{mi, fi})
			pVal := float32(dftPowerOut.FloatVal1D(int(bin)))
			sum += fVal * pVal
		}
		sum += mel.FBank.LogOff
		var val float32
		if sum == 0 {
			val = mel.FBank.LogMin
		} else {
			val = math32.Log(sum)
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
		fBankData.SetFloat1D(mi, float64(val))
		segmentData.Set([]int{step, mi, ch}, val)
	}
}

// FreqToMel converts frequency to mel scale
func FreqToMel(freq float32) float32 {
	return 1127.0 * math32.Log(1.0+freq/700.0)
}

// FreqToMel converts mel scale to frequency
func MelToFreq(mel float32) float32 {
	return 700.0 * (math32.Exp(mel/1127.0) - 1.0)
}

// FreqToBin converts frequency into FFT bin number, using parameters of number of FFT bins and sample rate
func FreqToBin(freq, nFft, sampleRate float32) int {
	return int(math32.Floor(((nFft + 1) * freq) / sampleRate))
}

//Defaults initializes FBank values - these are the ones you most likely need to adjust for your particular signals
func (mfb *FilterBank) Defaults() {
	mfb.LoHz = 300
	mfb.HiHz = 8000.0
	mfb.NFilters = 32
	mfb.LogOff = 0.0
	mfb.LogMin = -10.0
	mfb.Renorm = true
	mfb.RenormMin = -5.0
	mfb.RenormMax = 9.0
}

// Filter filters the current window_in input data according to current settings -- called by ProcessStep, but can be called separately
func (mel *Params) Filter(ch int, step int, windowIn *etensor.Float32, filters *etensor.Float32, dftPower *etensor.Float32, segmentData *etensor.Float32, fBankData *etensor.Float32, mfccSegmentData *etensor.Float32, mfccDct *etensor.Float32) {
	mel.FilterDft(ch, step, *dftPower, segmentData, fBankData, filters)
	if mel.CompMfcc {
		mel.CepstrumDct(ch, step, fBankData, mfccSegmentData, mfccDct)
	}
}

// FftReal
func (mel *Params) FftReal(out []complex128, in *etensor.Float32) {
	var c complex128
	for i := 0; i < len(out); i++ {
		c = complex(in.FloatVal1D(i), 0)
		out[i] = c
	}
}

// CepstrumDct applies a discrete cosine transform (DCT) to get the cepstrum coefficients on the mel filterbank values
func (mel *Params) CepstrumDct(ch, step int, fBankData *etensor.Float32, mfccSegmentData *etensor.Float32, mfccDct *etensor.Float32) {
	sz := copy(mfccDct.Values, fBankData.Values)
	if sz != len(mfccDct.Values) {
		log.Printf("mel.CepstrumDctMel: memory copy size wrong")
	}

	dct := fourier.NewDCT(len(mfccDct.Values))
	var mfccDctOut []float64
	src := []float64{}
	mfccDct.Floats(&src)
	mfccDctOut = dct.Transform(mfccDctOut, src)
	el0 := mfccDctOut[0]
	mfccDctOut[0] = math.Log(1.0 + el0*el0) // replace with log energy instead..
	for i := 0; i < mel.FBank.NFilters; i++ {
		mfccSegmentData.SetFloat([]int{step, i, ch}, mfccDctOut[i])
	}
}
