package mel

import (
	"fmt"
	"math"

	"github.com/chewxy/math32"
	"github.com/emer/etable/etensor"
	"gonum.org/v1/gonum/fourier"
)

// FilterBank contains mel frequency feature bank sampling parameters
type FilterBank struct {
	NFilters    int     `viewif:"On" def:"32,26" desc:"number of Mel frequency filters to compute"`
	LoHz        float32 `viewif:"On" def:"120,300" desc:"low frequency end of mel frequency spectrum"`
	HiHz        float32 `viewif:"On" def:"10000,8000" desc:"high frequency end of mel frequency spectrum -- must be <= sample_rate / 2 (i.e., less than the Nyquist frequencY"`
	LogOff      float32 `viewif:"On" def:"0" desc:"on add this amount when taking the log of the Mel filter sums to produce the filter-bank output -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs"`
	LogMin      float32 `viewif:"On" def:"-10" desc:"minimum value a log can produce -- puts a lower limit on log output"`
	LoMel       float32 `viewif:"On" inactive:"+" desc:" low end of mel scale in mel units"`
	HiMel       float32 `viewif:"On" inactive:"+" desc:" high end of mel scale in mel units"`
	Renorm      bool    `desc:"whether to perform renormalization of the mel values"`
	RenormMin   float32 `viewif:"On" desc:"minimum value to use for renormalization -- you must experiment with range of inputs to determine appropriate values"`
	RenormMax   float32 `viewif:"On" desc:"maximum value to use for renormalization -- you must experiment with range of inputs to determine appropriate values"`
	RenormScale float32 `inactive:"+" desc:"1.0 / (ren_max - ren_min)"`
}

type Mel struct {
	PtsBin      etensor.Int32   `view:"no-inline" desc:" [MelNFiltersEff] mel scale points in fft bins"`
	Filters     etensor.Float32 `view:"no-inline" desc:" [MelNFiltersEff][NFilters] the actual filters for actual number of mel filters"`
	MaxBins     int             `inactive:"+" desc:" maximum number of bins for mel filter -- number of bins in highest filter"`
	NFiltersEff int             `inactive:"+" desc:" effective number of mel filters: mel.n_filters + 2"`

	FBank     FilterBank
	FBankData etensor.Float32 `view:"no-inline" desc:" [mel.n_filters] mel scale transformation of dft_power, using triangular filters, resulting in the mel filterbank output -- the natural log of this is typically applied"`

	CompMfcc   bool            `desc:"compute cepstrum discrete cosine transform (dct) of the mel-frequency filter bank features"`
	MfccNCoefs int             `def:"13" desc:"number of mfcc coefficients to output -- typically 1/2 of the number of filterbank features"` // Todo: should be 12 total - 2 - 13, higher ones not useful
	MfccDctOut etensor.Float32 `view:"no-inline" desc:" discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
}

// Initialize
func (mel *Mel) Initialize(dftSize int, winSamples int, sampleRate int, compMfcc bool) {
	mel.CompMfcc = compMfcc
	mel.MfccNCoefs = 13
	mel.FBank.Initialize()
	mel.InitFilters(dftSize, sampleRate)
	mel.FBankData.SetShape([]int{mel.FBank.NFilters}, nil, nil)
	if mel.CompMfcc {
		mel.MfccDctOut.SetShape([]int{mel.FBank.NFilters}, nil, nil)
	}
}

// InitFilters
func (mel *Mel) InitFilters(dftSize int, sampleRate int) {
	mel.NFiltersEff = mel.FBank.NFilters + 2
	mel.PtsBin.SetShape([]int{mel.NFiltersEff}, nil, nil)
	melIncr := (mel.FBank.HiMel - mel.FBank.LoMel) / float32(mel.FBank.NFilters+1)

	for idx := 0; idx < mel.NFiltersEff; idx++ {
		ml := mel.FBank.LoMel + float32(idx)*melIncr
		hz := MelToFreq(ml)
		bin := FreqToBin(hz, float32(dftSize), float32(sampleRate))
		mel.PtsBin.SetFloat1D(idx, float64(bin))
	}

	mel.MaxBins = int(mel.PtsBin.Value1D(mel.NFiltersEff-1)) - int(mel.PtsBin.Value1D(mel.NFiltersEff-3)) + 1
	mel.Filters.SetShape([]int{mel.FBank.NFilters, mel.MaxBins}, nil, nil)

	for f := 0; f < mel.FBank.NFilters; f++ {
		mnbin := int(mel.PtsBin.Value1D(f))
		pkbin := int(mel.PtsBin.Value1D(f + 1))
		mxbin := int(mel.PtsBin.Value1D(f + 2))
		pkmin := float32(pkbin) - float32(mnbin)
		pkmax := float32(mxbin) - float32(pkbin)

		fi := 0
		bin := 0
		for bin = mnbin; bin <= pkbin; bin, fi = bin+1, fi+1 {
			fval := (float32(bin) - float32(mnbin)) / pkmin
			mel.Filters.SetFloat([]int{f, fi}, float64(fval))
		}
		for ; bin <= mxbin; bin, fi = bin+1, fi+1 {
			fval := (float32(mxbin) - float32(bin)) / pkmax
			mel.Filters.SetFloat([]int{f, fi}, float64(fval))
		}
	}
}

// FilterDft
func (mel *Mel) FilterDft(ch, step int, dftPowerOut etensor.Float32, trialData *etensor.Float32) {
	mi := 0
	for f := 0; f < int(mel.FBank.NFilters); f, mi = f+1, mi+1 { // f is filter
		minBin := mel.PtsBin.Value1D(f)
		maxBin := mel.PtsBin.Value1D(f + 2)

		sum := float32(0)
		fi := 0
		for bin := minBin; bin <= maxBin; bin, fi = bin+1, fi+1 {
			fVal := mel.Filters.Value([]int{mi, fi})
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
		mel.FBankData.SetFloat1D(mi, float64(val))
		trialData.Set([]int{step, mi, ch}, val)
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

//Initialize initializes the MelFBankSpec
func (mfb *FilterBank) Initialize() {
	mfb.LoHz = 120.0
	mfb.HiHz = 10000.0
	mfb.NFilters = 32
	mfb.LogOff = 0.0
	mfb.LogMin = -10.0
	mfb.LoMel = FreqToMel(mfb.LoHz)
	mfb.HiMel = FreqToMel(mfb.HiHz)
	mfb.Renorm = true
	mfb.RenormMin = -5.0
	mfb.RenormMax = 9.0
	mfb.RenormScale = 1.0 / (mfb.RenormMax - mfb.RenormMin)
}

// Filter filters the current window_in input data according to current settings -- called by ProcessStep, but can be called separately
func (mel *Mel) Filter(ch int, step int, windowIn etensor.Float32, dftPower *etensor.Float32, trialData *etensor.Float32, mfccTrialData *etensor.Float32) {
	mel.FilterDft(ch, step, *dftPower, trialData)
	if mel.CompMfcc {
		mel.CepstrumDctMel(ch, step, mfccTrialData)
	}
}

// FftReal
func (mel *Mel) FftReal(out []complex128, in etensor.Float32) {
	var c complex128
	for i := 0; i < len(out); i++ {
		c = complex(in.FloatVal1D(i), 0)
		out[i] = c
	}
}

// CopyStepFromStep
func (mel *Mel) CopyStepFromStep(toStep, fmStep, ch int, trialData *etensor.Float32, mfccTrialData *etensor.Float32) {
	for i := 0; i < int(mel.FBank.NFilters); i++ {
		val := trialData.Value([]int{fmStep, i, ch})
		trialData.Set([]int{toStep, i, ch}, val)
		if mel.CompMfcc {
			for i := 0; i < int(mel.FBank.NFilters); i++ {
				val := mfccTrialData.Value([]int{fmStep, i, ch})
				mfccTrialData.Set([]int{toStep, i, ch}, val)
			}
		}
	}
}

// CepstrumDctMel
func (mel *Mel) CepstrumDctMel(ch, step int, mfccTrialData *etensor.Float32) {
	sz := copy(mel.MfccDctOut.Values, mel.FBankData.Values)
	if sz != len(mel.MfccDctOut.Values) {
		fmt.Printf("CepstrumDctMel: memory copy size wrong")
	}

	dct := fourier.NewDCT(len(mel.MfccDctOut.Values))
	var mfccDctOut []float64
	src := []float64{}
	mel.MfccDctOut.Floats(&src)
	mfccDctOut = dct.Transform(mfccDctOut, src)
	el0 := mfccDctOut[0]
	mfccDctOut[0] = math.Log(1.0 + el0*el0) // replace with log energy instead..
	for i := 0; i < mel.FBank.NFilters; i++ {
		mfccTrialData.SetFloat([]int{step, i, ch}, mfccDctOut[i])
	}
}
