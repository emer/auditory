// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dft

import (
	"math"

	"github.com/emer/etable/etensor"
	"gonum.org/v1/gonum/dsp/fourier"
)

// Dft struct holds the variables for doing a fourier transform
type Params struct {

	// [def: true] compute the log of the power and save that to a separate table -- generaly more useful for visualization of power than raw power values
	CompLogPow bool `def:"true" desc:"compute the log of the power and save that to a separate table -- generaly more useful for visualization of power than raw power values"`

	// [def: -100] [viewif: CompLogPow] minimum value a log can produce -- puts a lower limit on log output
	LogMin float64 `viewif:"CompLogPow" def:"-100" desc:"minimum value a log can produce -- puts a lower limit on log output"`

	// [def: 0] [viewif: CompLogPow] add this amount when taking the log of the dft power -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs
	LogOffSet float64 `viewif:"CompLogPow" def:"0" desc:"add this amount when taking the log of the dft power -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs"`

	// [def: 0] how much of the previous step's power value to include in this one -- smooths out the power spectrum which can be artificially bumpy due to discrete window samples
	PrevSmooth float64 `def:"0" desc:"how much of the previous step's power value to include in this one -- smooths out the power spectrum which can be artificially bumpy due to discrete window samples"`

	//  how much of current power to include
	CurSmooth float64 `inactive:"+" desc:" how much of current power to include"`
}

func (dft *Params) Defaults() {
	dft.PrevSmooth = 0
	dft.CurSmooth = 1.0 - dft.PrevSmooth
	dft.CompLogPow = true
	dft.LogOffSet = 1.0
	dft.LogMin = -100
}

// Filter filters the current window_in input data according to current settings -- called by ProcessStep, but can be called separately
func (dft *Params) Filter(step int, windowIn *etensor.Float64, winSamples int, power *etensor.Float64, logPower *etensor.Float64, powerForSegment *etensor.Float64, logPowerForSegment *etensor.Float64) {
	fftCoefs := make([]complex128, winSamples)
	dft.FftReal(fftCoefs, windowIn)
	fft := fourier.NewCmplxFFT(len(fftCoefs))
	fftCoefs = fft.Coefficients(nil, fftCoefs)
	dft.Power(step, winSamples, fftCoefs, power, logPower, powerForSegment, logPowerForSegment)
	fft = nil
	fftCoefs = nil
}

// FftReal
func (dft *Params) FftReal(fftCoefs []complex128, in *etensor.Float64) {
	var c complex128
	for i := 0; i < len(fftCoefs); i++ {
		c = complex(in.FloatVal1D(i), 0)
		fftCoefs[i] = c
	}
}

// Power
func (dft *Params) Power(step int, winSamples int, fftCoefs []complex128, power *etensor.Float64, logPower *etensor.Float64, powerForSegment *etensor.Float64, logPowerForSegment *etensor.Float64) {
	for k := 0; k < winSamples/2+1; k++ {
		rl := real(fftCoefs[k])
		im := imag(fftCoefs[k])
		powr := float64(rl*rl + im*im)
		if step > 0 {
			powr = dft.PrevSmooth*power.FloatVal1D(k) + dft.CurSmooth*powr
		}
		power.SetFloat1D(k, powr)
		powerForSegment.SetFloat([]int{k, step}, powr)

		var logp float64
		if dft.CompLogPow {
			powr += dft.LogOffSet
			if powr == 0 {
				logp = dft.LogMin
			} else {
				logp = math.Log(powr)
			}
			logPower.SetFloat1D(k, logp)
			logPowerForSegment.SetFloat([]int{k, step}, logp)
		}
	}
}
