package audio

import (
	"math"

	"github.com/emer/etable/etensor"
	"gonum.org/v1/gonum/fourier"
)

// AudDftSpec discrete fourier transform (dft) specifications
type AudDftSpec struct {
	LogPow         bool    `"def:"true" desc:"compute the log of the power and save that to a separate table -- generaly more useful for visualization of power than raw power values"`
	LogOff         float32 `viewif:"LogPow" def:"0" desc:"add this amount when taking the log of the dft power -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs"`
	LogMin         float32 `viewif:"LogPow" def:"-100" desc:"minimum value a log can produce -- puts a lower limit on log output"`
	PreviousSmooth float32 `def:"0" desc:"how much of the previous step's power value to include in this one -- smooths out the power spectrum which can be artificially bumpy due to discrete window samples"`
	CurrentSmooth  float32 `inactive:"+" desc:"how much of current power to include"`
}

//Initialize initializes the AudDftSpec
func (ad *AudDftSpec) Initialize() {
	ad.PreviousSmooth = 0
	ad.CurrentSmooth = 1.0 - ad.PreviousSmooth
	ad.LogPow = true
	ad.LogOff = 0
	ad.LogMin = -100
}

type Dft struct {
	DftSpec             AudDftSpec      `desc:"specifications for how to compute the discrete fourier transform (DFT, using FFT)"`
	DftSize             int             `inactive:"+" desc:" #NO_SAVE full size of fft output -- should be input.win_samples"`
	DftUse              int             `inactive:"+" desc:" #NO_SAVE number of dft outputs to actually use -- should be dft_size / 2 + 1"`
	DftOut              []complex128    `inactive:"+" desc:" #NO_SAVE [dft_size] discrete fourier transform (fft) output complex representation"`
	DftPowerOut         etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [dft_use] power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPowerOut      etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [dft_use] log power of the dft, up to the nyquist liit frequency (1/2 input.win_samples)"`
	DftPowerTrialOut    etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [dft_use][input.total_steps][input.channels] full trial's worth of power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPowerTrialOut etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [dft_use][input.total_steps][input.channels] full trial's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
}

func (dft *Dft) Initialize(winSamples int, sampleRate int) {
	dft.DftSpec.Initialize()
	dft.DftSize = winSamples
	dft.DftOut = make([]complex128, dft.DftSize)
	dft.DftUse = dft.DftSize/2 + 1
}

// InitMatrices sets the shape of all output matrices
func (dft *Dft) InitMatrices(input Input) {
	dft.DftPowerOut.SetShape([]int{dft.DftUse}, nil, nil)
	dft.DftPowerTrialOut.SetShape([]int{dft.DftUse, input.TotalSteps, input.Channels}, nil, nil)

	if dft.DftSpec.LogPow {
		dft.DftLogPowerOut.SetShape([]int{dft.DftUse}, nil, nil)
		dft.DftLogPowerTrialOut.SetShape([]int{dft.DftUse, input.TotalSteps, input.Channels}, nil, nil)
	}
}

// NeedsInit checks to see if we need to reinitialize AuditoryProc
func (dft *Dft) NeedsInit(winSamples int) bool {
	if dft.DftSize != winSamples {
		return true
	}
	return false
}

// FilterWindow filters the current window_in input data according to current settings -- called by ProcessStep, but can be called separately
func (dft *Dft) FilterWindow(ch int, step int, windowIn etensor.Float32, firstStep bool) {
	dft.FftReal(dft.DftOut, windowIn)
	dft.DftInput(windowIn.Floats(), windowIn)
	dft.PowerOfDft(ch, step, firstStep)
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
func (dft *Dft) DftInput(windowInVals []float64, windowIn etensor.Float32) {
	dft.FftReal(dft.DftOut, windowIn)
	fft := fourier.NewCmplxFFT(len(dft.DftOut))
	dft.DftOut = fft.Coefficients(nil, dft.DftOut)
}

// PowerOfDft
func (dft *Dft) PowerOfDft(ch, step int, firstStep bool) {
	// Mag() is absolute value   SqMag is square of it - r*r + i*i
	for k := 0; k < int(dft.DftUse); k++ {
		rl := real(dft.DftOut[k])
		im := imag(dft.DftOut[k])
		powr := float64(rl*rl + im*im) // why is complex converted to float here
		if firstStep == false {
			powr = float64(dft.DftSpec.PreviousSmooth)*dft.DftPowerOut.FloatVal1D(k) + float64(dft.DftSpec.CurrentSmooth)*powr
		}
		dft.DftPowerOut.SetFloat1D(k, powr)
		dft.DftPowerTrialOut.SetFloat([]int{k, step, ch}, powr)

		var logp float64
		if dft.DftSpec.LogPow {
			powr += float64(dft.DftSpec.LogOff)
			if powr == 0 {
				logp = float64(dft.DftSpec.LogMin)
			} else {
				logp = math.Log(powr)
			}
			dft.DftLogPowerOut.SetFloat1D(k, logp)
			dft.DftLogPowerTrialOut.SetFloat([]int{k, step, ch}, logp)
		}
	}
}

// CopyStepFromStep
func (dft *Dft) CopyStepFromStep(toStep, fmStep, ch int) {
	for i := 0; i < int(dft.DftUse); i++ {
		val := dft.DftPowerTrialOut.Value([]int{i, fmStep, ch})
		dft.DftPowerTrialOut.Set([]int{i, toStep, ch}, val)
		if dft.DftSpec.LogPow {
			val := dft.DftLogPowerTrialOut.Value([]int{i, fmStep, ch})
			dft.DftLogPowerTrialOut.Set([]int{i, toStep, ch}, val)
		}
	}
}
