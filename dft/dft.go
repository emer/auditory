package dft

import (
	"math"

	"github.com/emer/etable/etensor"
	"gonum.org/v1/gonum/fourier"
)

// Dft struct holds the variables for doing a fourier transform
type Dft struct {
	CompLogPow     bool            `def:"true" desc:"compute the log of the power and save that to a separate table -- generaly more useful for visualization of power than raw power values"`
	LogMin         float32         `viewif:"CompLogPow" def:"-100" desc:"minimum value a log can produce -- puts a lower limit on log output"`
	LogOffSet      float32         `viewif:"CompLogPow" def:"0" desc:"add this amount when taking the log of the dft power -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs"`
	PreviousSmooth float32         `def:"0" desc:"how much of the previous step's power value to include in this one -- smooths out the power spectrum which can be artificially bumpy due to discrete window samples"`
	CurrentSmooth  float32         `inactive:"+" desc:" how much of current power to include"`
	SizeFull       int             `inactive:"+" desc:" full size of fft output -- should be input.win_samples"`
	SizeHalf       int             `inactive:"+" desc:" number of dft outputs to actually use -- should be SizeFull / 2 + 1"`
	Output         []complex128    `inactive:"+" desc:" discrete fourier transform (fft) output complex representation"`
	Power          etensor.Float32 `view:"no-inline" desc:" power of the dft, up to the nyquist limit frequency (1/2 input.winSamples)"`
	LogPower       etensor.Float32 `view:"no-inline" desc:" log power of the dft, up to the nyquist liit frequency (1/2 input.winSamples)"`
}

func (dft *Dft) Initialize(winSamples int, sampleRate int) {
	dft.PreviousSmooth = 0
	dft.CurrentSmooth = 1.0 - dft.PreviousSmooth
	dft.CompLogPow = true
	dft.LogOffSet = 0
	dft.LogMin = -100
	dft.SizeFull = winSamples
	dft.Output = make([]complex128, dft.SizeFull)
	dft.SizeHalf = dft.SizeFull/2 + 1
	dft.Power.SetShape([]int{dft.SizeHalf}, nil, nil)
	dft.LogPower.SetShape([]int{dft.SizeHalf}, nil, nil)
}

// Filter filters the current window_in input data according to current settings -- called by ProcessStep, but can be called separately
func (dft *Dft) Filter(ch int, step int, windowIn etensor.Float32, firstStep bool, powerForSegment *etensor.Float32, logPowerForSegment *etensor.Float32) {
	dft.FftReal(dft.Output, windowIn)
	dft.Input(windowIn)
	dft.CompPower(ch, step, firstStep, powerForSegment, logPowerForSegment)
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
func (dft *Dft) Input(windowIn etensor.Float32) {
	dft.FftReal(dft.Output, windowIn)
	fft := fourier.NewCmplxFFT(len(dft.Output))
	dft.Output = fft.Coefficients(nil, dft.Output)
}

// PowerOfDft
func (dft *Dft) CompPower(ch, step int, firstStep bool, powerForSegment *etensor.Float32, logPowerForSegment *etensor.Float32) {
	// Mag() is absolute value   SqMag is square of it - r*r + i*i
	for k := 0; k < int(dft.SizeHalf); k++ {
		rl := real(dft.Output[k])
		im := imag(dft.Output[k])
		powr := float64(rl*rl + im*im) // why is complex converted to float here
		if firstStep == false {
			powr = float64(dft.PreviousSmooth)*dft.Power.FloatVal1D(k) + float64(dft.CurrentSmooth)*powr
		}
		dft.Power.SetFloat1D(k, powr)
		powerForSegment.SetFloat([]int{step, k, ch}, powr)

		var logp float64
		if dft.CompLogPow {
			powr += float64(dft.LogOffSet)
			if powr == 0 {
				logp = float64(dft.LogMin)
			} else {
				logp = math.Log(powr)
			}
			dft.LogPower.SetFloat1D(k, logp)
			logPowerForSegment.SetFloat([]int{step, k, ch}, logp)
		}
	}
}
