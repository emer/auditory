// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dft

import (
	"math"

	"github.com/emer/etable/etensor"
	"gonum.org/v1/gonum/fourier"
)

// Dft struct holds the variables for doing a fourier transform
type Params struct {
	CompLogPow bool         `def:"true" desc:"compute the log of the power and save that to a separate table -- generaly more useful for visualization of power than raw power values"`
	LogMin     float32      `viewif:"CompLogPow" def:"-100" desc:"minimum value a log can produce -- puts a lower limit on log output"`
	LogOffSet  float32      `viewif:"CompLogPow" def:"0" desc:"add this amount when taking the log of the dft power -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs"`
	PrevSmooth float32      `def:"0" desc:"how much of the previous step's power value to include in this one -- smooths out the power spectrum which can be artificially bumpy due to discrete window samples"`
	CurSmooth  float32      `inactive:"+" desc:" how much of current power to include"`
	Fft        []complex128 `inactive:"+" desc:" discrete fourier transform (fft) output complex representation"`
}

func (dft *Params) Initialize(winSamples int, sampleRate int) {
	dft.PrevSmooth = 0
	dft.CurSmooth = 1.0 - dft.PrevSmooth
	dft.CompLogPow = true
	dft.LogOffSet = 0
	dft.LogMin = -100
	dft.Fft = make([]complex128, winSamples)
}

// Filter filters the current window_in input data according to current settings -- called by ProcessStep, but can be called separately
func (dft *Params) Filter(ch int, step int, windowIn *etensor.Float32, firstStep bool, winSamples int, power *etensor.Float32, logPower *etensor.Float32, powerForSegment *etensor.Float32, logPowerForSegment *etensor.Float32) {
	dft.FftReal(windowIn)
	dft.Input(windowIn)
	dft.Power(ch, step, firstStep, winSamples, power, logPower, powerForSegment, logPowerForSegment)
}

// FftReal
func (dft *Params) FftReal(in *etensor.Float32) {
	var c complex128
	for i := 0; i < len(dft.Fft); i++ {
		c = complex(in.FloatVal1D(i), 0)
		dft.Fft[i] = c
	}
}

// DftInput applies dft (fft) to input
func (dft *Params) Input(windowIn *etensor.Float32) {
	dft.FftReal(windowIn)
	fft := fourier.NewCmplxFFT(len(dft.Fft))
	dft.Fft = fft.Coefficients(nil, dft.Fft)
}

// PowerOfDft
func (dft *Params) Power(ch, step int, firstStep bool, winSamples int, power *etensor.Float32, logPower *etensor.Float32, powerForSegment *etensor.Float32, logPowerForSegment *etensor.Float32) {
	// Mag() is absolute value   SqMag is square of it - r*r + i*i
	for k := 0; k < winSamples/2+1; k++ {
		rl := real(dft.Fft[k])
		im := imag(dft.Fft[k])
		powr := float64(rl*rl + im*im) // why is complex converted to float here
		if firstStep == false {
			powr = float64(dft.PrevSmooth)*power.FloatVal1D(k) + float64(dft.CurSmooth)*powr
		}
		power.SetFloat1D(k, powr)
		powerForSegment.SetFloat([]int{step, k, ch}, powr)

		var logp float64
		if dft.CompLogPow {
			powr += float64(dft.LogOffSet)
			if powr == 0 {
				logp = float64(dft.LogMin)
			} else {
				logp = math.Log(powr)
			}
			logPower.SetFloat1D(k, logp)
			logPowerForSegment.SetFloat([]int{step, k, ch}, logp)
		}
	}
}
