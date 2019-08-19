package audio

import (
	"fmt"
	"math"

	"github.com/chewxy/math32"
	"github.com/emer/etable/etensor"
	"gonum.org/v1/gonum/fourier"
)

// RenormSpec holds the auditory renormalization parameters
type RenormSpec struct {
	On          bool    `desc:"perform renormalization of this level of the auditory signal"`
	RenormMin   float32 `viewif:"On" desc:"minimum value to use for renormalization -- you must experiment with range of inputs to determine appropriate values"`
	RenormMax   float32 `viewif:"On" desc:"maximum value to use for renormalization -- you must experiment with range of inputs to determine appropriate values"`
	RenormScale float32 `inactive:"+" desc:"1.0 / (ren_max - ren_min)"`
}

// Initialize initializes the RenormSpec
func (rs *RenormSpec) Initialize() {
	rs.On = true
	rs.RenormMin = -5.0
	rs.RenormMax = 9.0
	rs.RenormScale = 1.0 / (rs.RenormMax - rs.RenormMin)
}

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

// MelFBank contains mel frequency feature bank sampling parameters
type MelFBank struct {
	On          bool       `desc:"perform mel-frequency filtering of the fft input"`
	NFilters    int        `viewif:"On" def:"32,26" desc:"number of Mel frequency filters to compute"`
	LoHz        float32    `viewif:"On" def:"120,300" desc:"low frequency end of mel frequency spectrum"`
	HiHz        float32    `viewif:"On" def:"10000,8000" desc:"high frequency end of mel frequency spectrum -- must be <= sample_rate / 2 (i.e., less than the Nyquist frequencY"`
	LogOff      float32    `viewif:"On" def:"0" desc:"on add this amount when taking the log of the Mel filter sums to produce the filter-bank output -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs"`
	LogMin      float32    `viewif:"On" def:"-10" desc:"minimum value a log can produce -- puts a lower limit on log output"`
	LoMel       float32    `viewif:"On" inactive:"+" desc:" low end of mel scale in mel units"`
	HiMel       float32    `viewif:"On" inactive:"+" desc:" high end of mel scale in mel units"`
	FBankRenorm RenormSpec `viewif:"MelFBank.On=true desc:"renormalization parmeters for the mel_fbank values -- performed prior to further processing"`
}

type Mel struct {
	MelPtsMel        etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [mel_n_filters_eff] scale points in mel units (mels)"`
	MelPtsHz         etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [mel_n_filters_eff] mel scale points in hz units"`
	MelPtsBin        etensor.Int32   `view:"no-inline" desc:" #NO_SAVE [mel_n_filters_eff] mel scale points in fft bins"`
	MelFilters       etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [mel_filt_max_bins][mel.n_filters] the actual filters for actual number of mel filters"`
	MelFilterMaxBins int             `inactive:"+" desc:" #NO_SAVE maximum number of bins for mel filter -- number of bins in highest filter"`
	MelNFiltersEff   int             `inactive:"+" desc:" #NO_SAVE effective number of mel filters: mel.n_filters + 2"`

	MelFBank         MelFBank
	MelFBankOut      etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [mel.n_filters] mel scale transformation of dft_power, using triangular filters, resulting in the mel filterbank output -- the natural log of this is typically applied"`
	MelFBankTrialOut etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [mel.n_filters][input.total_steps][input.channels] full trial's worth of mel feature-bank output -- only if using gabors"`
	//MelFBankTrialOutTable etable.Table    `view:"no-inline" desc:" #NO_SAVE [mel.n_filters] MelFBankOutTrial - view only"`

	Dft                 AudDftSpec      `desc:"specifications for how to compute the discrete fourier transform (DFT, using FFT)"`
	DftSize             int             `inactive:"+" desc:" #NO_SAVE full size of fft output -- should be input.win_samples"`
	DftUse              int             `inactive:"+" desc:" #NO_SAVE number of dft outputs to actually use -- should be dft_size / 2 + 1"`
	DftOut              []complex128    `inactive:"+" desc:" #NO_SAVE [dft_size] discrete fourier transform (fft) output complex representation"`
	DftPowerOut         etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [dft_use] power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPowerOut      etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [dft_use] log power of the dft, up to the nyquist liit frequency (1/2 input.win_samples)"`
	DftPowerTrialOut    etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [dft_use][input.total_steps][input.channels] full trial's worth of power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPowerTrialOut etensor.Float32 `view:"no-inline" desc:" #NO_SAVE [dft_use][input.total_steps][input.channels] full trial's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	Mfcc                MelCepstrumSpec `viewif:"MelFBank.On=true desc:"specifications of the mel cepstrum discrete cosine transform of the mel fbank filter features"`
	MfccDctOut          etensor.Float32 `view:"no-inline" desc:" #NO_SAVE discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
	MfccDctTrialOut     etensor.Float32 `view:"no-inline" desc:" #NO_SAVE full trial's worth of discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
}

// Initialize
func (mel *Mel) Initialize(winSamples int, sampleRate int) {
	mel.Dft.Initialize()
	mel.DftSize = winSamples
	mel.DftOut = make([]complex128, mel.DftSize)
	mel.DftUse = mel.DftSize/2 + 1
	mel.MelFBank.Initialize()
	mel.InitFilters(mel.DftUse, sampleRate)
}

// InitFilters
func (mel *Mel) InitFilters(dftUse int, sampleRate int) bool {
	mel.DftUse = dftUse
	mel.MelNFiltersEff = mel.MelFBank.NFilters + 2
	mel.MelPtsMel.SetShape([]int{mel.MelNFiltersEff}, nil, nil)
	mel.MelPtsHz.SetShape([]int{mel.MelNFiltersEff}, nil, nil)
	mel.MelPtsBin.SetShape([]int{mel.MelNFiltersEff}, nil, nil)
	melIncr := (mel.MelFBank.HiMel - mel.MelFBank.LoMel) / float32(mel.MelFBank.NFilters+1)

	for idx := 0; idx < mel.MelNFiltersEff; idx++ {
		ml := mel.MelFBank.LoMel + float32(idx)*melIncr
		hz := MelToFreq(ml)
		bin := FreqToBin(hz, float32(mel.DftUse), float32(sampleRate))
		mel.MelPtsMel.SetFloat1D(idx, float64(ml))
		mel.MelPtsHz.SetFloat1D(idx, float64(hz))
		mel.MelPtsBin.SetFloat1D(idx, float64(bin))
	}

	mel.MelFilterMaxBins = int(mel.MelPtsBin.Value1D(mel.MelNFiltersEff-1)) - int(mel.MelPtsBin.Value1D(mel.MelNFiltersEff-3)) + 1
	mel.MelFilters.SetShape([]int{mel.MelFBank.NFilters, mel.MelFilterMaxBins}, nil, nil)

	for f := 0; f < mel.MelFBank.NFilters; f++ {
		mnbin := int(mel.MelPtsBin.Value1D(f))
		pkbin := int(mel.MelPtsBin.Value1D(f + 1))
		mxbin := int(mel.MelPtsBin.Value1D(f + 2))
		pkmin := pkbin - mnbin
		pkmax := mxbin - pkbin

		fi := 0
		bin := 0
		for bin = mnbin; bin <= pkbin; bin, fi = bin+1, fi+1 {
			fval := float32((bin - mnbin) / pkmin)
			mel.MelFilters.SetFloat([]int{f, fi}, float64(fval))
		}
		for ; bin <= mxbin; bin, fi = bin+1, fi+1 {
			fval := float32((mxbin - bin) / pkmax)
			mel.MelFilters.SetFloat([]int{f, fi}, float64(fval))
		}
	}
	return true
}

// InitMatrices sets the shape of all output matrices
func (mel *Mel) InitMatrices(input Input) {
	mel.DftPowerOut.SetShape([]int{mel.DftUse}, nil, nil)
	mel.DftPowerTrialOut.SetShape([]int{mel.DftUse, input.TotalSteps, input.Channels}, nil, nil)

	if mel.Dft.LogPow {
		mel.DftLogPowerOut.SetShape([]int{mel.DftUse}, nil, nil)
		mel.DftLogPowerTrialOut.SetShape([]int{mel.DftUse, input.TotalSteps, input.Channels}, nil, nil)
	}

	if mel.MelFBank.On {
		mel.MelFBankOut.SetShape([]int{mel.MelFBank.NFilters}, nil, nil)
		mel.MelFBankTrialOut.SetShape([]int{mel.MelFBank.NFilters, input.TotalSteps, input.Channels}, nil, nil)
		if mel.Mfcc.On {
			mel.MfccDctOut.SetShape([]int{mel.MelFBank.NFilters}, nil, nil)
			mel.MfccDctTrialOut.SetShape([]int{mel.MelFBank.NFilters, input.TotalSteps, input.Channels}, nil, nil)
		}
	}
}

// NeedsInit checks to see if we need to reinitialize AuditoryProc
func (mel *Mel) NeedsInit(winSamples int) bool {
	if mel.DftSize != winSamples || mel.MelNFiltersEff != mel.MelFBank.NFilters+2 {
		return true
	}
	return false
}

// MelFilterDft
func (mel *Mel) MelFilterDft(ch, step int, dftPowerOut *etensor.Float32) {
	mi := 0
	for f := 0; f < int(mel.MelFBank.NFilters); f, mi = f+1, mi+1 { // f is filter
		minBin := mel.MelPtsBin.Value1D(f)
		maxBin := mel.MelPtsBin.Value1D(f + 2)

		sum := float32(0)
		fi := 0
		for bin := minBin; bin < maxBin; bin, fi = bin+1, fi+1 {
			fVal := mel.MelFilters.Value([]int{mi, fi})
			pVal := dftPowerOut.FloatVal1D(int(bin))
			sum += fVal * float32(pVal)
		}
		sum += mel.MelFBank.LogOff
		var val float32
		if sum == 0 {
			val = mel.MelFBank.LogMin
		} else {
			val = math32.Log(sum)
		}
		if mel.MelFBank.FBankRenorm.On {
			val -= mel.MelFBank.FBankRenorm.RenormMin
		}
		if val < 0 {
			val = 0
		}
		val *= mel.MelFBank.FBankRenorm.RenormScale
		if val > 1.0 {
			val = 1.0
		}
		mel.MelFBankOut.SetFloat1D(mi, float64(val))
		mel.MelFBankTrialOut.Set([]int{mi, step, ch}, val)
	}
	//
	//mel.MelFBankTrialOutTable.SetFromSchema(etable.Schema{
	//	{"MelFBank", etensor.FLOAT32, []int{mel.MelFBankTrialOut.Dim(0), mel.MelFBankTrialOut.Dim(1), mel.MelFBankTrialOut.Dim(2)}, []string{"Row", "Y", "X"}},
	//}, mel.MelFBankTrialOut.Dim(2))
	//mel.MelFBankTrialOutTable.Col(0).CopyFrom(&mel.MelFBankTrialOut)
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

//Initialize initializes the MelFBankSpec
func (mfb *MelFBank) Initialize() {
	mfb.On = true
	mfb.LoHz = 120.0
	mfb.HiHz = 10000.0
	mfb.NFilters = 32
	mfb.LogOff = 0.0
	mfb.LogMin = -10.0
	mfb.LoMel = FreqToMel(mfb.LoHz)
	mfb.HiMel = FreqToMel(mfb.HiHz)
	mfb.FBankRenorm.Initialize()
}

// MelCepstrumSpec holds the mel frequency sampling parameters
type MelCepstrumSpec struct {
	On     bool `desc:"perform cepstrum discrete cosine transform (dct) of the mel-frequency filter bank features"`
	NCoeff int  `def:"13" desc:"number of mfcc coefficients to output -- typically 1/2 of the number of filterbank features"`
}

//Initialize initializes the MelCepstrumSpec
func (mc *MelCepstrumSpec) Initialize() {
	mc.On = false
	mc.NCoeff = 13
}

// FilterWindow filters the current window_in input data according to current settings -- called by ProcessStep, but can be called separately
func (mel *Mel) FilterWindow(ch int, step int, windowIn etensor.Float32, firstStep bool) bool {
	mel.FftReal(mel.DftOut, windowIn)
	mel.DftInput(windowIn.Floats(), windowIn)
	if mel.MelFBank.On {
		mel.PowerOfDft(ch, step, firstStep)
		mel.MelFilterDft(ch, step, &mel.DftPowerOut)
		if mel.Mfcc.On {
			mel.CepstrumDctMel(ch, step)
		}
	}
	return true
}

// FftReal
func (mel *Mel) FftReal(out []complex128, in etensor.Float32) {
	var c complex128
	for i := 0; i < len(out); i++ {
		c = complex(in.FloatVal1D(i), 0)
		out[i] = c
	}
}

// DftInput applies dft (fft) to input
func (mel *Mel) DftInput(windowInVals []float64, windowIn etensor.Float32) {
	mel.FftReal(mel.DftOut, windowIn)
	fft := fourier.NewCmplxFFT(len(mel.DftOut))
	mel.DftOut = fft.Coefficients(nil, mel.DftOut)
}

// PowerOfDft
func (mel *Mel) PowerOfDft(ch, step int, firstStep bool) {
	// Mag() is absolute value   SqMag is square of it - r*r + i*i
	for k := 0; k < int(mel.DftUse); k++ {
		rl := real(mel.DftOut[k])
		im := imag(mel.DftOut[k])
		powr := float64(rl*rl + im*im) // why is complex converted to float here
		if firstStep == false {
			powr = float64(mel.Dft.PreviousSmooth)*mel.DftPowerOut.FloatVal1D(k) + float64(mel.Dft.CurrentSmooth)*powr
		}
		mel.DftPowerOut.SetFloat1D(k, powr)
		mel.DftPowerTrialOut.SetFloat([]int{k, step, ch}, powr)

		var logp float64
		if mel.Dft.LogPow {
			powr += float64(mel.Dft.LogOff)
			if powr == 0 {
				logp = float64(mel.Dft.LogMin)
			} else {
				logp = math.Log(powr)
			}
			mel.DftLogPowerOut.SetFloat1D(k, logp)
			mel.DftLogPowerTrialOut.SetFloat([]int{k, step, ch}, logp)
		}
	}
}

// CopyStepFromStep
func (mel *Mel) CopyStepFromStep(toStep, fmStep, ch int) bool {
	for i := 0; i < int(mel.DftUse); i++ {
		val := mel.DftPowerTrialOut.Value([]int{i, fmStep, ch})
		mel.DftPowerTrialOut.Set([]int{i, toStep, ch}, val)
		if mel.Dft.LogPow {
			val := mel.DftLogPowerTrialOut.Value([]int{i, fmStep, ch})
			mel.DftLogPowerTrialOut.Set([]int{i, toStep, ch}, val)
		}
	}
	if mel.MelFBank.On {
		for i := 0; i < int(mel.MelFBank.NFilters); i++ {
			val := mel.MelFBankTrialOut.Value([]int{i, fmStep, ch})
			mel.MelFBankTrialOut.Set([]int{i, toStep, ch}, val)
		}
		if mel.Mfcc.On {
			for i := 0; i < int(mel.MelFBank.NFilters); i++ {
				val := mel.MfccDctTrialOut.Value([]int{i, fmStep, ch})
				mel.MfccDctTrialOut.Set([]int{i, toStep, ch}, val)
			}
		}
	}
	return true
}

// CepstrumDctMel
func (mel *Mel) CepstrumDctMel(ch, step int) {
	sz := copy(mel.MfccDctOut.Values, mel.MelFBankOut.Values)
	if sz != len(mel.MfccDctOut.Values) {
		fmt.Printf("CepstrumDctMel: memory copy size wrong")
	}

	dct := fourier.NewDCT(len(mel.MfccDctOut.Values))
	var mfccDctOut []float64
	mfccDctOut = dct.Transform(mfccDctOut, mel.MfccDctOut.Floats())
	el0 := mfccDctOut[0]
	mfccDctOut[0] = math.Log(1.0 + el0*el0) // replace with log energy instead..
	for i := 0; i < mel.MelFBank.NFilters; i++ {
		mel.MfccDctTrialOut.SetFloat([]int{i, step, ch}, mfccDctOut[i])
	}
}
