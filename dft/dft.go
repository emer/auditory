package dft

import (
	"math"

	"github.com/emer/etable/etensor"
	"gonum.org/v1/gonum/fourier"
)

// Dft struct holds the variables for doing a fourier transform
type Dft struct {
	CompLogPow     bool            `"def:"true" desc:"compute the log of the power and save that to a separate table -- generaly more useful for visualization of power than raw power values"`
	LogMin         float32         `viewif:"LogPow" def:"-100" desc:"minimum value a log can produce -- puts a lower limit on log output"`
	LogOffSet      float32         `viewif:"LogPow" def:"0" desc:"add this amount when taking the log of the dft power -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs"`
	PreviousSmooth float32         `def:"0" desc:"how much of the previous step's power value to include in this one -- smooths out the power spectrum which can be artificially bumpy due to discrete window samples"`
	CurrentSmooth  float32         `inactive:"+" desc:"how much of current power to include"`
	SizeFull       int             `inactive:"+" desc:" #NO_SAVE full size of fft output -- should be input.win_samples"`
	SizeHalf       int             `inactive:"+" desc:" #NO_SAVE number of dft outputs to actually use -- should be dft_size / 2 + 1"`
	DftOut         []complex128    `inactive:"+" desc:" #NO_SAVE [dft_size] discrete fourier transform (fft) output complex representation"`
	DftPower       etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [dft_use] power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPower    etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [dft_use] log power of the dft, up to the nyquist liit frequency (1/2 input.win_samples)"`
}

func (dft *Dft) Initialize(winSamples int, sampleRate int) {
	dft.PreviousSmooth = 0
	dft.CurrentSmooth = 1.0 - dft.PreviousSmooth
	dft.CompLogPow = true
	dft.LogOffSet = 0
	dft.LogMin = -100
	dft.SizeFull = winSamples
	dft.DftOut = make([]complex128, dft.SizeFull)
	dft.SizeHalf = dft.SizeFull/2 + 1
	dft.DftPower.SetShape([]int{dft.SizeHalf}, nil, nil)
	dft.DftLogPower.SetShape([]int{dft.SizeHalf}, nil, nil)
}

// Filter filters the current window_in input data according to current settings -- called by ProcessStep, but can be called separately
func (dft *Dft) Filter(ch int, step int, windowIn etensor.Float32, firstStep bool, powerForTrial *etensor.Float32, logPowerForTrial *etensor.Float32) {
	dft.FftReal(dft.DftOut, windowIn)
	dft.Input(windowIn.Floats(), windowIn)
	dft.Power(ch, step, firstStep, powerForTrial, logPowerForTrial)
}

// FftReal
func (dft *Dft) FftReal(out []complex128, in etensor.Float32) {
	var c complex128
	for i := 0; i < len(out); i++ {
		c = complex(in.FloatVal1D(i), 0)
		out[i] = c
	}
}

// DftInput applies dft (fft) to input
func (dft *Dft) Input(windowInVals []float64, windowIn etensor.Float32) {
	dft.FftReal(dft.DftOut, windowIn)
	fft := fourier.NewCmplxFFT(len(dft.DftOut))
	dft.DftOut = fft.Coefficients(nil, dft.DftOut)
}

// PowerOfDft
func (dft *Dft) Power(ch, step int, firstStep bool, powerForTrial *etensor.Float32, logPowerForTrial *etensor.Float32) {
	// Mag() is absolute value   SqMag is square of it - r*r + i*i
	for k := 0; k < int(dft.SizeHalf); k++ {
		rl := real(dft.DftOut[k])
		im := imag(dft.DftOut[k])
		powr := float64(rl*rl + im*im) // why is complex converted to float here
		if firstStep == false {
			powr = float64(dft.PreviousSmooth)*dft.DftPower.FloatVal1D(k) + float64(dft.CurrentSmooth)*powr
		}
		dft.DftPower.SetFloat1D(k, powr)
		powerForTrial.SetFloat([]int{k, step, ch}, powr)

		var logp float64
		if dft.CompLogPow {
			powr += float64(dft.LogOffSet)
			if powr == 0 {
				logp = float64(dft.LogMin)
			} else {
				logp = math.Log(powr)
			}
			dft.DftLogPower.SetFloat1D(k, logp)
			logPowerForTrial.SetFloat([]int{k, step, ch}, logp)
		}
	}
}

// CopyStepFromStep
func (dft *Dft) CopyStepFromStep(toStep, fmStep, ch int, powerForTrial *etensor.Float32, logPowerForTrial *etensor.Float32) {
	for i := 0; i < int(dft.SizeHalf); i++ {
		val := powerForTrial.Value([]int{i, fmStep, ch})
		powerForTrial.Set([]int{i, toStep, ch}, val)
		if dft.CompLogPow {
			val := logPowerForTrial.Value([]int{i, fmStep, ch})
			logPowerForTrial.Set([]int{i, toStep, ch}, val)
		}
	}
}
