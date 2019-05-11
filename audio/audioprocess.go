// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package audio

import (
	"fmt"
	"image"
	"math"
	"strconv"

	"github.com/chewxy/math32"
	"github.com/emer/etable/etable"
	"github.com/emer/etable/etensor"
	"gonum.org/v1/gonum/fourier"
)

// AudInputSpec defines the sound input parameters for auditory processing
type AudInputSpec struct {
	WinMsec      float32 `desc:"#DEF_25 input window -- number of milliseconds worth of sound to filter at a time"`
	StepMsec     float32 `desc:"#DEF_5;10;12.5 input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample"`
	TrialMsec    float32 `desc:"#DEF_100 length of a full trial's worth of input -- total number of milliseconds to accumulate into a complete trial of activations to present to a network -- must be a multiple of step_msec -- input will be trial_msec / step_msec = trial_steps wide in the X axis, and number of filters in the Y axis"`
	SampleRate   int     `desc:"rate of sampling in our sound input (e.g., 16000 = 16Khz) -- can initialize this from a taSound object using InitFromSound method"`
	BorderSteps  int     `desc:"number of steps before and after the trial window to preserve -- this is important when applying temporal filters that have greater temporal extent"`
	Channels     int     `desc:"total number of channels to process"`
	Channel      int     `desc:"#CONDSHOW_ON_channels:1 specific channel to process, if input has multiple channels, and we only process one of them (-1 = process all)"`
	WinSamples   int     `desc:"#READ_ONLY #SHOW total number of samples to process (win_msec * .001 * sample_rate)"`
	StepSamples  int     `desc:"#READ_ONLY #SHOW total number of samples to step input by (step_msec * .001 * sample_rate)"`
	TrialSamples int     `desc:"#READ_ONLY #SHOW total number of samples in a trial  (trail_msec * .001 * sample_rate)"`
	TrialSteps   int     `desc:"#READ_ONLY #SHOW total number of steps in a trial  (trail_msec / step_msec)"`
	TotalSteps   int     `desc:"#READ_ONLY #SHOW 2*border_steps + trial_steps -- total in full window"`
}

//Initialize initializes the AudInputSpec
func (ais *AudInputSpec) Initialize() {
	ais.WinMsec = 25.0
	ais.StepMsec = 5.0
	ais.TrialMsec = 100.0
	ais.BorderSteps = 2
	ais.SampleRate = 44100
	ais.Channels = 1
	ais.Channel = 0
	ais.ComputeSamples()
}

// ComputeSamples computes the sample counts based on time and sample rate
func (ais *AudInputSpec) ComputeSamples() {
	ais.WinSamples = MSecToSamples(ais.WinMsec, ais.SampleRate)
	ais.StepSamples = MSecToSamples(ais.StepMsec, ais.SampleRate)
	ais.TrialSamples = MSecToSamples(ais.TrialMsec, ais.SampleRate)
	ais.TrialSteps = int(math.Round(float64(ais.TrialMsec / ais.StepMsec)))
	ais.TotalSteps = 2*ais.BorderSteps + ais.TrialSteps
}

// MSecToSamples converts milliseconds to samples, in terms of sample_rate
func MSecToSamples(msec float32, rate int) int {
	return int(math.Round(float64(msec) * 0.001 * float64(rate)))
}

// SamplesToMSec converts samples to milliseconds, in terms of sample_rate
func SamplesToMSec(samples int, rate int) float32 {
	return 1000.0 * float32(samples) / float32(rate)
}

// InitSound
func (ap *AuditoryProc) InitSound() bool {
	ap.InputPos = 0
	//ap.SoundFull = nil
	return true
}

// InitFromSound loads a sound and sets the AudInputSpec channel vars and sample rate
func (ais *AudInputSpec) InitFromSound(snd *Sound, nChannels int, channel int) {
	if snd == nil {
		fmt.Printf("InitFromSound: sound nil")
		return
	}

	ais.SampleRate = int(snd.SampleRate())
	ais.ComputeSamples()
	if nChannels < 1 {
		ais.Channels = int(snd.Channels())
	} else {
		ais.Channels = int(math32.Min(float32(nChannels), float32(ais.Channels)))
	}
	if ais.Channels > 1 {
		ais.Channel = channel
	} else {
		ais.Channel = 0
	}
}

// AudDftSpec discrete fourier transform (dft) specifications
type AudDftSpec struct {
	LogPow         bool    `desc:"#DEF_true compute the log of the power and save that to a separate table -- generaly more useful for visualization of power than raw power values"`
	LogOff         float32 `desc:"#CONDSHOW_ON_log_pow #DEF_0 add this amount when taking the log of the dft power -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs"`
	LogMin         float32 `desc:"#CONDSHOW_ON_log_pow #DEF_-100 minimum value a log can produce -- puts a lower limit on log output"`
	PreviousSmooth float32 `desc:"#DEF_0 how much of the previous step's power value to include in this one -- smooths out the power spectrum which can be artificially bumpy due to discrete window samples"`
	CurrentSmooth  float32 `desc:"#READ_ONLY #EXPERT 1 - prv_smooth -- how much of current power to include"`
}

//Initialize initializes the AudDftSpec
func (ad *AudDftSpec) Initialize() {
	ad.PreviousSmooth = 0
	ad.CurrentSmooth = 1.0 - ad.PreviousSmooth
	ad.LogPow = true
	ad.LogOff = 0
	ad.LogMin = -100
}

// AudRenormSpec holds the auditory renormalization parameters
type AudRenormSpec struct {
	On          bool    `desc:"perform renormalization of this level of the auditory signal"`
	RenormMin   float32 `desc:"#CONDSHOW_ON_on minimum value to use for renormalization -- you must experiment with range of inputs to determine appropriate values"`
	RenormMax   float32 `desc:"#CONDSHOW_ON_on maximum value to use for renormalization -- you must experiment with range of inputs to determine appropriate values"`
	RenormScale float32 `desc:"#READ_ONLY 1.0 / (ren_max - ren_min)"`
}

//Initialize initializes the AudRenormSpec
func (ar *AudRenormSpec) Initialize() {
	ar.On = true
	ar.RenormMin = -10.0
	ar.RenormMax = 7.0
	ar.RenormScale = 1.0 / (ar.RenormMax - ar.RenormMin)
}

// AudGaborSpec params for auditory gabor filters: 2d Gaussian envelope times a sinusoidal plane wave --
// by default produces 2 phase asymmetric edge detector filters -- horizontal tuning is different from V1 version --
// has elongated frequency-band specific tuning, not a parallel horizontal tuning -- and has multiple of these
type AudGaborSpec struct {
	On              bool    `desc:"use this gabor filtering of the time-frequency space filtered input (time in terms of steps of the DFT transform, and discrete frequency factors based on the FFT window and input sample rate)"`
	SizeTime        int     `desc:"#CONDSHOW_ON_on #DEF_6;8;12;16;24 size of the filter in the time (horizontal) domain, in terms of steps of the underlying DFT filtering steps"`
	SizeFreq        int     `desc:"#CONDSHOW_ON_on #DEF_6;8;12;16;24 size of the filter in the frequency domain, in terms of discrete frequency factors based on the FFT window and input sample rate"`
	SpaceTime       int     `desc:"#CONDSHOW_ON_on spacing in the time (horizontal) domain, in terms of steps"`
	SpaceFreq       int     `desc:"#CONDSHOW_ON_on spacing in the frequency (vertical) domain"`
	WaveLen         float32 `desc:"#CONDSHOW_ON_on #DEF_1.5;2 wavelength of the sine waves in normalized units"`
	SigmaLen        float32 `desc:"#CONDSHOW_ON_on #DEF_0.6 gaussian sigma for the length dimension (elongated axis perpendicular to the sine waves) -- normalized as a function of filter size in relevant dimension"`
	SigmaWidth      float32 `desc:"#CONDSHOW_ON_on #DEF_0.3 gaussian sigma for the width dimension (in the direction of the sine waves) -- normalized as a function of filter size in relevant dimension"`
	HorizSigmaLen   float32 `desc:"#CONDSHOW_ON_on #DEF_0.3 gaussian sigma for the length of special horizontal narrow-band filters -- normalized as a function of filter size in relevant dimension"`
	HorizSigmaWidth float32 `desc:"#CONDSHOW_ON_on #DEF_0.1 gaussian sigma for the horizontal dimension for special horizontal narrow-band filters -- normalized as a function of filter size in relevant dimension"`
	Gain            float32 `desc:"#CONDSHOW_ON_on #DEF_2 overall gain multiplier applied after gabor filtering -- only relevant if not using renormalization (otherwize it just gets renormed awaY"`
	NHoriz          int     `desc:"#CONDSHOW_ON_on #DEF_4 number of horizontally-elongated,  pure time-domain, frequency-band specific filters to include, evenly spaced over the available frequency space for this filter set -- in addition to these, there are two diagonals (45, 135) and a vertically-elongated (wide frequency band) filter"`
	PhaseOffset     float32 `desc:"#CONDSHOW_ON_on #DEF_0;1.5708 offset for the sine phase -- default is an asymmetric sine wave -- can make it into a symmetric cosine gabor by using PI/2 = 1.5708"`
	CircleEdge      bool    `desc:"#CONDSHOW_ON_on #DEF_true cut off the filter (to zero) outside a circle of diameter filter_size -- makes the filter more radially symmetric"`
	NFilters        int     `desc:"#CONDSHOW_ON_on #READ_ONLY #SHOW total number of filters = 3 + n_horiz"`
}

//Initialize initializes the AudGaborSpec
func (ag *AudGaborSpec) Initialize() {
	ag.On = true
	ag.Gain = 2.0
	ag.NHoriz = 4
	ag.SizeTime = 6.0
	ag.SizeFreq = 6.0
	ag.SpaceTime = 2.0
	ag.SpaceFreq = 2.0
	ag.WaveLen = 1.5
	ag.SigmaLen = 0.6
	ag.SigmaWidth = 0.3
	ag.HorizSigmaLen = 0.3
	ag.HorizSigmaWidth = 0.1
	ag.PhaseOffset = 0.0
	ag.CircleEdge = true
	ag.NFilters = 3 + ag.NHoriz
}

// RenderFilters generates filters into the given matrix, which is formatted as: [ag.SizeTime_steps][ag.SizeFreq][n_filters]
func (ag *AudGaborSpec) RenderFilters(filters *etensor.Float32) {
	filters.SetShape([]int{ag.SizeTime, ag.SizeFreq, ag.NFilters}, nil, nil)

	ctrTime := (ag.SizeTime - 1) / 2.0
	ctrFreq := (ag.SizeFreq - 1) / 2.0
	angInc := math32.Pi / 4.0

	radiusTime := ag.SizeTime / 2.0
	radiusFreq := ag.SizeFreq / 2.0

	lenNorm := 1.0 / (2.0 * ag.SigmaLen * ag.SigmaLen)
	widthNorm := 1.0 / (2.0 * ag.SigmaWidth * ag.SigmaWidth)
	lenHorizNorm := 1.0 / (2.0 * ag.HorizSigmaLen * ag.HorizSigmaLen)
	widthHorizNorm := 1.0 / (2.0 * ag.HorizSigmaWidth * ag.HorizSigmaWidth)

	twoPiNorm := (2.0 * math32.Pi) / ag.WaveLen
	hCtrInc := (ag.SizeFreq - 1) / (ag.NHoriz + 1)

	fli := 0
	for hi := 0; hi < ag.NHoriz; hi, fli = hi+1, fli+1 {
		hCtrFreq := hCtrInc * (hi + 1)
		angF := -2.0 * angInc
		for y := 0; y < ag.SizeFreq; y++ {
			for x := 0; x < ag.SizeTime; x++ {
				xf := x - ctrTime
				yf := y - hCtrFreq
				xfn := float32(xf / radiusTime)
				yfn := float32(yf / radiusFreq)

				dist := math32.Hypot(xfn, yfn)
				val := float32(0)
				if !(ag.CircleEdge && dist > 1.0) {
					nx := xfn*math32.Cos(angF) - yfn*math32.Sin(angF)
					ny := yfn*math32.Cos(angF) + xfn*math32.Sin(angF)
					gauss := math32.Exp(-(widthHorizNorm*(nx*nx) + lenHorizNorm*(ny*ny)))
					sinVal := math32.Sin(twoPiNorm*ny + ag.PhaseOffset)
					val = gauss * sinVal
				}
				filters.Set([]int{x, y, fli}, val)
			}
		}
	}

	for ang := 1; ang < 4; ang, fli = ang+1, fli+1 {
		angF := float32(-ang) * angInc

		for y := 0; y < ag.SizeFreq; y++ {
			for x := 0; x < ag.SizeTime; x++ {
				xf := x - ctrTime
				yf := y - ctrFreq
				xfn := float32(xf / radiusTime)
				yfn := float32(yf / radiusFreq)

				dist := math32.Hypot(xfn, yfn)
				val := float32(0)
				if !(ag.CircleEdge && dist > 1.0) {
					nx := xfn*math32.Cos(angF) - yfn*math32.Sin(angF)
					ny := yfn*math32.Cos(angF) + xfn*math32.Sin(angF)
					gauss := math32.Exp(-(lenNorm*(nx*nx) + widthNorm*(ny*ny)))
					sinVal := math32.Sin(twoPiNorm*ny + ag.PhaseOffset)
					val = gauss * sinVal
				}
				filters.Set([]int{x, y, fli}, val)
			}
		}
	}

	// renorm each half
	for fli := 0; fli < ag.NFilters; fli++ {
		posSum := float32(0)
		negSum := float32(0)
		for y := 0; y < ag.SizeFreq; y++ {
			for x := 0; x < ag.SizeTime; x++ {
				val := float32(filters.Value([]int{x, y, fli}))
				if val > 0 {
					posSum += val
				} else if val < 0 {
					negSum += val
				}
			}
		}
		posNorm := 1.0 / posSum
		negNorm := -1.0 / negSum
		for y := 0; y < ag.SizeFreq; y++ {
			for x := 0; x < ag.SizeTime; x++ {
				val := filters.Value([]int{x, y, fli})
				if val > 0.0 {
					val *= posNorm
					filters.Set([]int{x, y, fli}, val)
				} else if val < 0.0 {
					val *= negNorm
					filters.Set([]int{x, y, fli}, val)
				}
			}
		}
	}
}

// MelFBankSpec contains mel frequency feature bank sampling parameters
type MelFBankSpec struct {
	On       bool    `desc:"perform mel-frequency filtering of the fft input"`
	NFilters int     `desc:"#DEF_32;26 #CONDSHOW_ON_on number of Mel frequency filters to compute"`
	LoHz     float32 `desc:"#DEF_120;300 #CONDSHOW_ON_on low frequency end of mel frequency spectrum"`
	HiHz     float32 `desc:"#DEF_10000;8000 #CONDSHOW_ON_on high frequency end of mel frequency spectrum -- must be <= sample_rate / 2 (i.e., less than the Nyquist frequencY"`
	LogOff   float32 `desc:"#CONDSHOW_ON_on #DEF_0 on add this amount when taking the log of the Mel filter sums to produce the filter-bank output -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs"`
	LogMin   float32 `desc:"#CONDSHOW_ON_on #DEF_-10 minimum value a log can produce -- puts a lower limit on log output"`
	LoMel    float32 `desc:"#READ_ONLY #SHOW #CONDSHOW_ON_on low end of mel scale in mel units"`
	HiMel    float32 `desc:"#READ_ONLY #SHOW #CONDSHOW_ON_on high end of mel scale in mel units"`
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
func (mfb *MelFBankSpec) Initialize() {
	mfb.On = true
	mfb.LoHz = 120.0
	mfb.HiHz = 10000.0
	mfb.NFilters = 32
	mfb.LogOff = 0.0
	mfb.LogMin = -10.0
	mfb.LoMel = FreqToMel(mfb.LoHz)
	mfb.HiMel = FreqToMel(mfb.HiHz)
}

// MelCepstrumSpec holds the mel frequency sampling parameters
type MelCepstrumSpec struct {
	On     bool `desc:"perform cepstrum discrete cosine transform (dct) of the mel-frequency filter bank features"`
	NCoeff int  `desc:"#CONDSHOW_ON_on #DEF_13 number of mfcc coefficients to output -- typically 1/2 of the number of filterbank features"`
}

//Initialize initializes the MelCepstrumSpec
func (mc *MelCepstrumSpec) Initialize() {
	mc.On = true
	mc.NCoeff = 13
}

type AuditoryProc struct {
	// From C++ version not ported
	//	enum SaveMode {               // how to add new data to the data table
	//	NONE_SAVE,                  // don't save anything at all -- overrides any more specific save guys and prevents any addition or modification to the data table
	//	FIRST_ROW,                  // always overwrite the first row -- does EnforceRows(1) if rows = 0
	//	ADD_ROW,                    // always add a new row and write to that, preserving a history of inputs over time -- should be reset at some interval!
	//};
	//	SaveMode      save_mode;      // how to add new data to the data table
	//	V1KwtaSpec    gabor_kwta;     // #CONDSHOW_ON_gabor1.on k-winner-take-all inhibitory dynamics for the time-gabor output

	Data        *etable.Table   `desc:"data table for saving filter results for viewing and applying to networks etc"`
	Input       AudInputSpec    `desc:"specifications of the raw auditory input"`
	Dft         AudDftSpec      `desc:"specifications for how to compute the discrete fourier transform (DFT, using FFT)"`
	MelFBank    MelFBankSpec    `desc:"specifications of the mel feature bank frequency sampling of the DFT (FFT) of the input sound"`
	FBankRenorm AudRenormSpec   `desc:"#CONDSHOW_ON_mel_fbank.on renormalization parmeters for the mel_fbank values -- performed prior to further processing"`
	Gabor1      AudGaborSpec    `desc:"#CONDSHOW_ON_mel_fbank.on full set of frequency / time gabor filters -- first size"`
	Gabor2      AudGaborSpec    `desc:"#CONDSHOW_ON_mel_fbank.on full set of frequency / time gabor filters -- second size"`
	Gabor3      AudGaborSpec    `desc:"#CONDSHOW_ON_mel_fbank.on full set of frequency / time gabor filters -- third size"`
	Mfcc        MelCepstrumSpec `desc:"#CONDSHOW_ON_mel_fbank.on specifications of the mel cepstrum discrete cosine transform of the mel fbank filter features"`

	// Filters
	DftSize        int `desc:"#READ_ONLY #NO_SAVE full size of fft output -- should be input.win_samples"`
	DftUse         int `desc:"#READ_ONLY #NO_SAVE number of dft outputs to actually use -- should be dft_size / 2 + 1"`
	MelNFiltersEff int `desc:"#READ_ONLY #NO_SAVE effective number of mel filters: mel.n_filters + 2"`

	MelPtsMel        etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel_n_filters_eff] scale points in mel units (mels)"`
	MelPtsHz         etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel_n_filters_eff] mel scale points in hz units"`
	MelPtsBin        etensor.Int32   `desc:"#READ_ONLY #NO_SAVE [mel_n_filters_eff] mel scale points in fft bins"`
	MelFilters       etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel_filt_max_bins][mel.n_filters] the actual filters for actual number of mel filters"`
	MelFilterMaxBins int             `desc:"#READ_ONLY #NO_SAVE maximum number of bins for mel filter -- number of bins in highest filter"`

	Gabor1Filters etensor.Float32 `desc:"#READ_ONLY #NO_SAVE full gabor filters"`
	Gabor2Filters etensor.Float32 `desc:"#READ_ONLY #NO_SAVE full gabor filters"`
	Gabor3Filters etensor.Float32 `desc:"#READ_ONLY #NO_SAVE full gabor filters"`

	// Outputs
	FirstStep     bool        `desc:"#READ_ONLY #NO_SAVE #SHOW is this the first step of processing -- turns of prv smoothing of dft power"`
	InputPos      int         `desc:"#READ_ONLY #NO_SAVE #SHOW current position in the sound_full input -- in terms of sample number"`
	TrialStartPos int         `desc:"#READ_ONLY #NO_SAVE #SHOW starting position of the current trial -- in terms of sample number"`
	TrialEndPos   int         `desc:"#READ_ONLY #NO_SAVE #SHOW ending position of the current trial -- in terms of sample number"`
	Gabor1Shape   image.Point `desc:"#CONDSHOW_ON_gabor1.on #READ_ONLY #SHOW overall geometry of gabor1 output (group-level geometry -- feature / unit level geometry is n_features, 2)"`
	Gabor2Shape   image.Point `desc:"#CONDSHOW_ON_gabor2.on #READ_ONLY #SHOW overall geometry of gabor2 output (group-level geometry -- feature / unit level geometry is n_features, 2)"`
	Gabor3Shape   image.Point `desc:"#CONDSHOW_ON_gabor3.on #READ_ONLY #SHOW overall geometry of gabor3 output (group-level geometry -- feature / unit level geometry is n_features, 2)"`

	SoundFull           etensor.Float32    `desc:"#READ_ONLY #NO_SAVE the full sound input obtained from the sound input"`
	WindowIn            etensor.Float32    `desc:"#READ_ONLY #NO_SAVE [input.win_samples] the raw sound input, one channel at a time"`
	DftOut              etensor.Complex128 `desc:"#READ_ONLY #NO_SAVE [dft_size] discrete fourier transform (fft) output complex representation"`
	DftPowerOut         etensor.Float32    `desc:"#READ_ONLY #NO_SAVE [dft_use] power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPowerOut      etensor.Float32    `desc:"#READ_ONLY #NO_SAVE [dft_use] log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftPowerTrialOut    etensor.Float32    `desc:"#READ_ONLY #NO_SAVE [dft_use][input.total_steps][input.channels] full trial's worth of power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPowerTrialOut etensor.Float32    `desc:"#READ_ONLY #NO_SAVE [dft_use][input.total_steps][input.channels] full trial's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`

	MelFBankOut      etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel.n_filters] mel scale transformation of dft_power, using triangular filters, resulting in the mel filterbank output -- the natural log of this is typically applied"`
	MelFBankTrialOut etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel.n_filters][input.total_steps][input.channels] full trial's worth of mel feature-bank output -- only if using gabors"`

	GaborGci  etensor.Float32 `desc:"#READ_ONLY #NO_SAVE inhibitory conductances, for computing kwta"`
	Gabor1Raw etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] raw output of gabor1 -- full trial's worth of gabor steps"`
	Gabor1Out etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] post-kwta output of full trial's worth of gabor steps"`
	Gabor2Raw etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] raw output of gabor1 -- full trial's worth of gabor steps"`
	Gabor2Out etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] post-kwta output of full trial's worth of gabor steps"`
	Gabor3Raw etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] raw output of gabor1 -- full trial's worth of gabor steps"`
	Gabor3Out etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] post-kwta output of full trial's worth of gabor steps"`

	MfccDctOut      etensor.Float32 `desc:"#READ_ONLY #NO_SAVE discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
	MfccDctTrialOut etensor.Float32 `desc:"#READ_ONLY #NO_SAVE full trial's worth of discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
}

// InitFilters
func (ap *AuditoryProc) InitFilters() bool {
	ap.DftSize = ap.Input.WinSamples
	ap.DftUse = ap.DftSize/2 + 1
	ap.InitFiltersMel()
	if ap.Gabor1.On {
		ap.Gabor1Filters.SetShape([]int{ap.Gabor1.SizeTime, ap.Gabor1.SizeFreq, ap.Gabor1.NFilters}, nil, nil)
		ap.Gabor1.RenderFilters(&ap.Gabor1Filters)
	}
	if ap.Gabor2.On {
		ap.Gabor2Filters.SetShape([]int{ap.Gabor2.SizeTime, ap.Gabor2.SizeFreq, ap.Gabor2.NFilters}, nil, nil)
		ap.Gabor2.RenderFilters(&ap.Gabor2Filters)
	}
	if ap.Gabor3.On {
		ap.Gabor3Filters.SetShape([]int{ap.Gabor3.SizeTime, ap.Gabor3.SizeFreq, ap.Gabor3.NFilters}, nil, nil)
		ap.Gabor3.RenderFilters(&ap.Gabor3Filters)
	}
	return true
}

// InitFiltersMel
func (ap *AuditoryProc) InitFiltersMel() bool {
	ap.MelNFiltersEff = ap.MelFBank.NFilters + 2
	ap.MelPtsMel.SetShape([]int{1, ap.MelNFiltersEff}, nil, nil)
	ap.MelPtsHz.SetShape([]int{1, ap.MelNFiltersEff}, nil, nil)
	ap.MelPtsBin.SetShape([]int{1, ap.MelNFiltersEff}, nil, nil)

	melIncr := (ap.MelFBank.HiMel - ap.MelFBank.LoMel) / float32(ap.MelFBank.NFilters+1)

	for idx := 0; idx < ap.MelNFiltersEff; idx++ {
		ml := ap.MelFBank.LoMel + float32(idx)*melIncr
		hz := MelToFreq(ml)
		bin := FreqToBin(hz, float32(ap.DftUse), float32(ap.Input.SampleRate))
		ap.MelPtsMel.SetFloat1D(idx, float64(ml))
		ap.MelPtsHz.SetFloat1D(idx, float64(hz))
		ap.MelPtsBin.SetFloat1D(idx, float64(bin))
	}

	ap.MelFilterMaxBins = int(ap.MelPtsBin.Value1D(ap.MelNFiltersEff-1)) - int(ap.MelPtsBin.Value1D(ap.MelNFiltersEff-3)) + 1
	ap.MelFilters.SetShape([]int{ap.MelFilterMaxBins, ap.MelFBank.NFilters}, nil, nil)

	for f := 0; f < ap.MelFBank.NFilters; f++ {
		mnbin := int(ap.MelPtsBin.Value1D(f))
		pkbin := int(ap.MelPtsBin.Value1D(f + 1))
		mxbin := int(ap.MelPtsBin.Value1D(f + 2))
		pkmin := pkbin - mnbin
		pkmax := mxbin - pkbin

		fi := 0
		bin := 0
		for bin = mnbin; bin < pkbin; bin, fi = bin+1, fi+1 {
			fval := float32((bin - mnbin) / pkmin)
			ap.MelFilters.SetFloat([]int{fi, f}, float64(fval))
		}
		for ; bin < mxbin; bin, fi = bin+1, fi+1 {
			fval := float32((mxbin - bin) / pkmax)
			ap.MelFilters.SetFloat([]int{fi, f}, float64(fval))
		}
	}
	return true
}

// InitOutMatrix
func (ap *AuditoryProc) InitOutMatrix() bool {
	ap.WindowIn.SetShape([]int{ap.Input.WinSamples}, nil, nil)
	ap.DftOut.SetShape([]int{ap.DftSize}, nil, nil)
	ap.DftPowerOut.SetShape([]int{ap.DftUse}, nil, nil)
	ap.DftPowerTrialOut.SetShape([]int{ap.DftUse, ap.Input.TotalSteps, ap.Input.Channels}, nil, nil)

	if ap.Dft.LogPow {
		ap.DftLogPowerOut.SetShape([]int{ap.DftUse}, nil, nil)
		ap.DftLogPowerTrialOut.SetShape([]int{ap.DftUse, ap.Input.TotalSteps, ap.Input.Channels}, nil, nil)
	}

	if ap.MelFBank.On {
		ap.MelFBankOut.SetShape([]int{ap.MelFBank.NFilters}, nil, nil)
		ap.MelFBankTrialOut.SetShape([]int{ap.MelFBank.NFilters, ap.Input.TotalSteps, ap.Input.Channels}, nil, nil)
		if ap.Gabor1.On {
			ap.Gabor1Raw.SetShape([]int{ap.Gabor1.NFilters, 2, ap.Gabor1Shape.Y, ap.Gabor1Shape.X, ap.Input.Channels}, nil, nil)
			ap.Gabor1Out.SetShape([]int{ap.Gabor1.NFilters, 2, ap.Gabor1Shape.Y, ap.Gabor1Shape.X, ap.Input.Channels}, nil, nil)
		}
		if ap.Gabor2.On {
			ap.Gabor2Raw.SetShape([]int{ap.Gabor2.NFilters, 2, ap.Gabor2Shape.Y, ap.Gabor2Shape.X, ap.Input.Channels}, nil, nil)
			ap.Gabor2Out.SetShape([]int{ap.Gabor2.NFilters, 2, ap.Gabor2Shape.Y, ap.Gabor2Shape.X, ap.Input.Channels}, nil, nil)
		}
		if ap.Gabor3.On {
			ap.Gabor3Raw.SetShape([]int{ap.Gabor3.NFilters, 2, ap.Gabor3Shape.Y, ap.Gabor3Shape.X, ap.Input.Channels}, nil, nil)
			ap.Gabor3Out.SetShape([]int{ap.Gabor3.NFilters, 2, ap.Gabor3Shape.Y, ap.Gabor3Shape.X, ap.Input.Channels}, nil, nil)
		}
		if ap.Mfcc.On {
			ap.MfccDctOut.SetShape([]int{ap.MelFBank.NFilters}, nil, nil)
			ap.MfccDctTrialOut.SetShape([]int{ap.MelFBank.NFilters, ap.Input.TotalSteps, ap.Input.Channels}, nil, nil)
		}
	}
	return true
}

func (ap *AuditoryProc) LoadSound(snd *Sound) bool {
	var needsInit = false
	if ap.NeedsInit() {
		needsInit = true
	}

	if snd == nil || !snd.IsValid() {
		fmt.Printf("LoadSound: sound nil or invalid")
		return false
	}

	if int(snd.SampleRate()) != ap.Input.SampleRate {
		fmt.Printf("LoadSound: sample rate does not match sound -- re-initializing with new rate of: %v", strconv.Itoa(int(snd.SampleRate())))
		ap.Input.SampleRate = int(snd.SampleRate())
		needsInit = true
	}

	if needsInit {
		ap.Init()
	}

	if ap.Input.Channels > 1 {
		snd.SoundToMatrix(&ap.SoundFull, -1)
	} else {
		snd.SoundToMatrix(&ap.SoundFull, ap.Input.Channel)
	}
	ap.StartNewSound()
	return true
}

func (ap *AuditoryProc) StartNewSound() bool {
	ap.FirstStep = true
	ap.InputPos = 0
	ap.TrialStartPos = 0
	ap.TrialEndPos = int(ap.TrialStartPos) + ap.Input.TrialSamples
	return true
}

func (ap *AuditoryProc) NeedsInit() bool {
	if int(ap.DftSize) != ap.Input.WinSamples || ap.MelNFiltersEff != ap.MelFBank.NFilters+2 {
		return true
	}
	return false

}

// UpdateConfig
func (ap *AuditoryProc) UpdateConfig() {
	ap.Gabor1Shape.X = ((ap.Input.TrialSteps - 1) / ap.Gabor1.SpaceTime) + 1
	ap.Gabor1Shape.Y = ((ap.MelFBank.NFilters - ap.Gabor1.SizeFreq - 1) / ap.Gabor1.SpaceFreq) + 1

	ap.Gabor2Shape.X = ((ap.Input.TrialSteps - 1) / ap.Gabor2.SpaceTime) + 1
	ap.Gabor2Shape.Y = ((ap.MelFBank.NFilters - ap.Gabor2.SizeFreq - 1) / ap.Gabor2.SpaceFreq) + 1

	ap.Gabor3Shape.X = ((ap.Input.TrialSteps - 1) / ap.Gabor3.SpaceTime) + 1
	ap.Gabor3Shape.Y = ((ap.MelFBank.NFilters - ap.Gabor3.SizeFreq - 1) / ap.Gabor3.SpaceFreq) + 1
}

func (ap *AuditoryProc) Init() bool {
	ap.Input.Initialize()
	ap.Dft.Initialize()
	ap.MelFBank.Initialize()
	ap.FBankRenorm.Initialize()
	ap.Gabor1.Initialize()
	ap.Gabor2.Initialize()
	ap.Gabor3.Initialize()
	ap.UpdateConfig()
	ap.Mfcc.Initialize()

	ap.InitFilters()
	ap.InitOutMatrix()
	ap.Data = &etable.Table{}
	ap.InitDataTable()
	ap.InitSound()
	return true
}

func (ap *AuditoryProc) InitDataTable() bool {
	if ap.Data == nil {
		fmt.Printf("InitDataTable: ap.Data is nil")
		return false
	}
	if ap.Input.Channels > 1 {
		for ch := 0; ch < int(ap.Input.Channels); ch++ {
			ap.InitDataTableChan(ch)
		}
	} else {
		ap.InitDataTableChan(int(ap.Input.Channel))
	}
	return true
}

func (ap *AuditoryProc) InitDataTableChan(ch int) bool {
	if ap.MelFBank.On {
		ap.MelOutputToTable(ap.Data, ch, true)
	}
	return true
}

// InputStepsLeft returns the number of steps left to process in the current input sound
func (ap *AuditoryProc) InputStepsLeft() int {
	samplesLeft := len(ap.SoundFull.Values) - ap.InputPos
	//samplesLeft = ap.SoundFull.Frames() - ap.InputPos
	return samplesLeft / ap.Input.StepSamples
}

// ProcessTrial processes a full trial worth of sound -- iterates over steps to fill a trial's worth of sound data
func (ap *AuditoryProc) ProcessTrial() bool {
	if ap.NeedsInit() {
		ap.Init()
	}
	ap.Data.AddRows(1)

	if ap.InputStepsLeft() < 1 {
		fmt.Printf("ProcessTrial: no steps worth of input sound available -- load a new sound")
		return false
	}

	startPos := ap.InputPos
	if ap.InputPos == 0 { // just starting out -- fill whole buffer..
		border := 2 * ap.Input.BorderSteps // full amount to wrap
		ap.TrialStartPos = ap.InputPos
		ap.TrialEndPos = ap.TrialStartPos + ap.Input.TrialSamples + 2*border*ap.Input.StepSamples

		for ch := int(0); ch < ap.Input.Channels; ch++ {
			ap.InputPos = startPos // always start at same place per channel
			for s := 0; s < int(ap.Input.TotalSteps); s++ {
				ap.ProcessStep(ch, s)
			}
			ap.FilterTrial(ch)
			ap.OutputToTable(ch)
		}
	} else {
		border := 2 * ap.Input.BorderSteps // full amount to wrap
		ap.TrialStartPos = ap.InputPos - ap.Input.TrialSamples*ap.Input.BorderSteps
		ap.TrialEndPos = ap.TrialStartPos + ap.Input.TrialSamples

		for ch := 0; ch < int(ap.Input.Channels); ch++ {
			ap.InputPos = startPos // always start at same place per channel
			ap.WrapBorder(ch)
			for s := border; s < ap.Input.TotalSteps; s++ {
				ap.ProcessStep(ch, s)
			}
			ap.FilterTrial(ch)
			ap.OutputToTable(ch)
		}
	}
	return true
}

// SoundToWindow gets sound from sound_full at given position and channel, into window_in -- pads with zeros for any amount not available in the sound_full input
func (ap *AuditoryProc) SoundToWindow(inPos int, ch int) bool {
	samplesAvail := len(ap.SoundFull.Values) - inPos
	samplesCopy := int(math32.Min(float32(samplesAvail), float32(ap.Input.WinSamples)))
	if samplesCopy > 0 {
		if ap.SoundFull.NumDims() == 1 {
			copy(ap.WindowIn.Values, ap.SoundFull.Values[inPos:samplesCopy+inPos])
		} else {
			// todo: comment from c++ version - this is not right
			//memcpy(window_in.el, (void*)&(sound_full.FastEl2d(chan, in_pos)), sz);
			fmt.Printf("SoundToWindow: else case not implemented - please report this issue")
		}
	}
	samplesCopy = int(math32.Max(float32(samplesCopy), 0)) // prevent negatives here -- otherwise overflows
	// pad remainder with zero
	zeroN := int(ap.Input.WinSamples) - int(samplesCopy)
	if zeroN > 0 {
		//sz := zeroN * int(unsafe.Sizeof(Float))
		sz := zeroN * 4
		copy(ap.WindowIn.Values[samplesCopy:], make([]float32, sz))
	}
	return true

}

// WrapBorder
func (ap *AuditoryProc) WrapBorder(ch int) bool {
	if ap.Input.BorderSteps == 0 {
		return true
	}
	borderEff := 2 * ap.Input.BorderSteps
	srcStStep := ap.Input.TotalSteps - borderEff
	for s := 0; s < int(borderEff); s++ {
		ap.CopyStepFromStep(s, int(srcStStep)+s, ch)
	}
	return true
}

// StepForward
func (ap *AuditoryProc) StepForward(ch int) bool {
	totalM1 := ap.Input.TotalSteps - 1
	for s := 0; s < int(totalM1); s++ {
		ap.CopyStepFromStep(s, s+1, ch)
	}
	return true
}

// CopyStepFromStep
func (ap *AuditoryProc) CopyStepFromStep(toStep, fmStep, ch int) bool {
	for i := 0; i < int(ap.DftUse); i++ {
		val := ap.DftPowerTrialOut.Value([]int{i, fmStep, ch})
		ap.DftPowerTrialOut.Set([]int{i, toStep, ch}, val)
		if ap.Dft.LogPow {
			val := ap.DftLogPowerTrialOut.Value([]int{i, fmStep, ch})
			ap.DftLogPowerTrialOut.Set([]int{i, toStep, ch}, val)
		}
	}
	if ap.MelFBank.On {
		for i := 0; i < int(ap.MelFBank.NFilters); i++ {
			val := ap.MelFBankTrialOut.Value([]int{i, fmStep, ch})
			ap.MelFBankTrialOut.Set([]int{i, toStep, ch}, val)
		}
		if ap.Mfcc.On {
			for i := 0; i < int(ap.MelFBank.NFilters); i++ {
				val := ap.MfccDctTrialOut.Value([]int{i, fmStep, ch})
				ap.MfccDctTrialOut.Set([]int{i, toStep, ch}, val)
			}
		}
	}
	return true
}

// ProcessStep process a step worth of sound input from current input_pos, and increment input_pos by input.step_samples
func (ap *AuditoryProc) ProcessStep(ch int, step int) bool {
	ap.SoundToWindow(ap.InputPos, ch)
	ap.FilterWindow(int(ch), int(step))
	ap.InputPos = ap.InputPos + ap.Input.StepSamples
	ap.FirstStep = false
	return true
}

// DftInput applies dft (fft) to input
func (ap *AuditoryProc) DftInput(windowInVals []float64) {
	//taMath_float::fft_real(&dft_out, &window_in); // old emergent
	fft := fourier.NewFFT(len(windowInVals))
	fftOut := fft.Coefficients(nil, windowInVals)
	var i int
	var v complex128
	for i, v = range fftOut {
		ap.DftOut.Set([]int{i}, v)
	}
}

// PowerOfDft
func (ap *AuditoryProc) PowerOfDft(ch, step int) {
	// Mag() is absolute value   SqMag is square of it - r*r + i*i
	for k := 0; k < int(ap.DftUse); k++ {
		rl := ap.DftOut.FloatVal1D(k)
		im := ap.DftOut.FloatVal1DImag(k)
		powr := float64(rl*rl + im*im) // why is complex converted to float here
		if ap.FirstStep == false {
			powr = float64(ap.Dft.PreviousSmooth)*ap.DftPowerOut.FloatVal1D(k) + float64(ap.Dft.CurrentSmooth)*powr
		}
		ap.DftPowerOut.SetFloat1D(k, powr)
		ap.DftPowerTrialOut.SetFloat([]int{k, step, ch}, powr)
		var logp float64
		if ap.Dft.LogPow {
			powr += float64(ap.Dft.LogOff)
			if powr == 0 {
				logp = float64(ap.Dft.LogMin)
			} else {
				logp = math.Log(powr)
			}
			ap.DftLogPowerOut.SetFloat1D(k, logp)
			ap.DftLogPowerTrialOut.SetFloat([]int{k, step, ch}, logp)
		}
	}
}

// MelFilterDft
func (ap *AuditoryProc) MelFilterDft(ch, step int) {
	mi := 0
	for flt := 0; flt < int(ap.MelFBank.NFilters); flt, mi = flt+1, mi+1 {
		minBin := ap.MelPtsBin.Value1D(flt)
		maxBin := ap.MelPtsBin.Value1D(flt + 2)

		sum := float32(0)
		fi := 0
		for bin := minBin; bin < maxBin; bin, fi = bin+1, fi+1 {
			fVal := ap.MelFilters.Value([]int{fi, mi})
			pVal := ap.DftPowerOut.FloatVal1D(int(bin))
			sum += fVal * float32(pVal)
		}
		sum += ap.MelFBank.LogOff
		var val float32
		if sum == 0 {
			val = ap.MelFBank.LogMin
		} else {
			val = math32.Log(sum)
		}
		if ap.FBankRenorm.On {
			val -= ap.FBankRenorm.RenormMin
		}
		if val < 0 {
			val = 0
		}
		val *= ap.FBankRenorm.RenormScale
		if val > 1.0 {
			val = 1.0
		}
	}
}

// FilterTrial process filters that operate over an entire trial at a time
func (ap *AuditoryProc) FilterTrial(ch int) bool {
	if ap.Gabor1.On {
		ap.GaborFilter(ch, &ap.Gabor1, &ap.Gabor1Filters, &ap.Gabor1Raw, &ap.Gabor1Out)
	}
	if ap.Gabor2.On {
		ap.GaborFilter(ch, &ap.Gabor2, &ap.Gabor2Filters, &ap.Gabor2Raw, &ap.Gabor2Out)
	}
	if ap.Gabor3.On {
		ap.GaborFilter(ch, &ap.Gabor3, &ap.Gabor3Filters, &ap.Gabor3Raw, &ap.Gabor3Out)
	}
	return true
}

// CepstrumDctMel
func (ap *AuditoryProc) CepstrumDctMel(ch, step int) {
	sz := copy(ap.MfccDctOut.Values, ap.MelFBankOut.Values)
	if sz != len(ap.MfccDctOut.Values) {
		fmt.Printf("CepstrumDctMel: memory copy size wrong")
	}

	dct := fourier.NewDCT(len(ap.MfccDctOut.Values))
	var mfccDctOut []float64
	mfccDctOut = dct.Transform(mfccDctOut, ap.MfccDctOut.Floats1D())
	el0 := mfccDctOut[0]
	mfccDctOut[0] = math.Log(1.0 + el0*el0) // replace with log energy instead..
	for i := 0; i < ap.MelFBank.NFilters; i++ {
		ap.MfccDctTrialOut.SetFloat([]int{i, step, ch}, mfccDctOut[i])
	}
}

// GaborFilter process filters that operate over an entire trial at a time
func (ap *AuditoryProc) GaborFilter(ch int, spec *AudGaborSpec, filters *etensor.Float32, outRaw *etensor.Float32, out *etensor.Float32) {
	tHalfSz := spec.SizeTime / 2
	tOff := tHalfSz - ap.Input.BorderSteps
	tMin := tOff
	if tMin < 0 {
		tMin = 0
	}
	tMax := ap.Input.TrialSteps - tMin

	fMin := int(0)
	fMax := ap.MelFBank.NFilters - spec.SizeFreq

	tIdx := 0
	for s := tMin; s < tMax; s, tIdx = s+spec.SpaceTime, tIdx+1 {
		//inSt := s - tOff
		if tIdx > outRaw.Dim(3) {
			fmt.Printf("GaborFilter: time index %v out of range: %v", tIdx, outRaw.Dim(3))
			break
		}

		fIdx := 0
		for flt := fMin; flt < fMax; flt, fIdx = flt+spec.SpaceFreq, fIdx+1 {
			if fIdx > outRaw.Dim(2) {
				fmt.Printf("GaborFilter: freq index %v out of range: %v", tIdx, outRaw.Dim(2))
				break
			}
			nf := spec.NFilters
			for fi := int(0); fi < nf; fi++ {
				fSum := float32(0.0)
				for ff := int(0); ff < spec.SizeFreq; ff++ {
					for ft := int(0); ft < spec.SizeTime; ft++ {
						fVal := filters.Value([]int{ft, ff, fi})
						//iVal := ap.MelFBankTrialOut.Value([]int{flt + ff, inSt + ft, ch})
						//fSum += fVal * iVal
						fSum += fVal
					}
				}
				pos := fSum >= 0.0
				act := spec.Gain * math32.Abs(fSum)
				if pos {
					outRaw.SetFloat([]int{ch, fi, 0, fIdx, tIdx}, float64(act))
					outRaw.SetFloat([]int{ch, fi, 1, fIdx, tIdx}, 0)
				} else {
					outRaw.SetFloat([]int{ch, fi, 0, fIdx, tIdx}, 0)
					outRaw.SetFloat([]int{ch, fi, 1, fIdx, tIdx}, float64(act))
				}
			}
		}
	}
	rawFrame, err := outRaw.SubSpace(outRaw.NumDims()-1, []int{ch})
	if err != nil {
		fmt.Printf("GaborFilter: SubSpace error: %v", err)
	}

	outFrame, err := out.SubSpace(outRaw.NumDims()-1, []int{ch})
	if err != nil {
		fmt.Printf("GaborFilter: SubSpace error: %v", err)
	}

	// todo: V1KwtaSpec not yet implemented
	//if (gabor_kwta.On()) {
	//	gabor_kwta.Compute_Inhib(*raw_frm, *out_frm, gabor_gci);
	//} else {
	//	memcpy(out_frm->el, raw_frm->el, raw_frm->size * sizeof(float));
	copy(outFrame.Values, rawFrame.Values)
	//}

}

// FilterWindow filters the current window_in input data according to current settings -- called by ProcessStep, but can be called separately
func (ap *AuditoryProc) FilterWindow(ch int, step int) bool {
	//ap.DftInput(ch, step)
	ap.DftInput(ap.WindowIn.Floats1D())
	if ap.MelFBank.On {
		ap.PowerOfDft(ch, step)
		ap.MelFilterDft(ch, step)
		if ap.Mfcc.On {
			ap.CepstrumDctMel(ch, step)
		}
	}
	return true
}

// OutputToTable
func (ap *AuditoryProc) OutputToTable(ch int) bool {
	if ap.Data == nil {
		return false
	}
	if ap.MelFBank.On {
		ap.MelOutputToTable(ap.Data, ch, false) // not fmt_only
	}
	return true
}

// MelOutputToTable mel filter bank to output table
func (ap *AuditoryProc) MelOutputToTable(dt *etable.Table, ch int, fmtOnly bool) bool { // ch is channel
	var colSfx string
	rows := 1

	if ap.Input.Channels > 1 {
		colSfx = "_ch" + strconv.Itoa(ch)
	}

	var err error
	cn := "AudProc" + "_dft_pow" + colSfx // column name
	col := dt.ColByName(cn)
	if col == nil {
		err = dt.AddCol(etensor.NewFloat32([]int{rows, int(ap.Input.TotalSteps), int(ap.DftUse)}, nil, nil), cn)
		if err != nil {
			fmt.Printf("MelOutputToTable: column not found or failed to be created")
			return false
		}
	}

	if fmtOnly == false {
		colAsF32 := dt.ColByName(cn).(*etensor.Float32)
		dout, err := colAsF32.SubSpace(2, []int{dt.Rows - 1})
		if err != nil {
			fmt.Printf("MelOutputToTable: subspacing error")
			return false
		}
		for s := 0; s < int(ap.Input.TotalSteps); s++ {
			for i := 0; i < int(ap.DftUse); i++ {
				if ap.Dft.LogPow {
					val := ap.DftLogPowerTrialOut.FloatVal([]int{i, s, ch})
					dout.SetFloat([]int{s, i}, val)
				} else {
					val := ap.DftPowerTrialOut.FloatVal([]int{i, s, ch})
					dout.SetFloat([]int{s, i}, val)
				}
				//for _, v := range  colAsF32.Values {
				//	fmt.Printf("%v, ", v)
				//}
			}
		}
	}

	if ap.MelFBank.On {
		cn := "AudProc" + "_mel_fbank" + colSfx // column name
		col := dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, int(ap.Input.TotalSteps), int(ap.MelFBank.NFilters)}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}
		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpace(2, []int{dt.Rows - 1})
			if err != nil {
				fmt.Printf("MelOutputToTable: subspacing error")
				return false
			}
			for s := 0; s < int(ap.Input.TotalSteps); s++ {
				for i := 0; i < int(ap.MelFBank.NFilters); i++ {
					val := ap.MelFBankTrialOut.FloatVal([]int{i, s, ch})
					dout.SetFloat([]int{s, i}, val)
				}
			}
		}
	}

	if ap.Gabor1.On {
		cn := "AudProc" + "_mel_gabor1_raw" + colSfx // column name
		col := dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor1.NFilters, 2, ap.Gabor1Shape.X, ap.Gabor1Shape.Y}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}

		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpace(4, []int{dt.Rows - 1})
			if err != nil {
				fmt.Printf("MelOutputToTable: mel_gabor1_raw subspacing error")
				return false
			}
			nf := ap.Gabor1.NFilters
			for s := 0; s < ap.Gabor1Shape.X; s++ {
				for i := 0; i < ap.Gabor1Shape.Y; i++ {
					for ti := 0; ti < nf; ti++ {
						val0 := ap.Gabor1Raw.FloatVal([]int{ti, 0, i, s, ch})
						dout.SetFloat([]int{ti, 0, s, i}, val0)
						val1 := ap.Gabor1Raw.FloatVal([]int{ti, 1, i, s, ch})
						dout.SetFloat([]int{ti, 1, s, i}, val1)
					}
				}
			}
		}

		cn = "AudProc" + "_mel_gabor1" + colSfx // column name
		col = dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor1.NFilters, 2, ap.Gabor1Shape.X, ap.Gabor1Shape.Y}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}
		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpace(4, []int{dt.Rows - 1})
			if err != nil {
				fmt.Printf("MelOutputToTable: mel_gabor1 subspacing error")
				return false
			}
			nf := ap.Gabor1.NFilters
			for s := 0; s < ap.Gabor1Shape.X; s++ {
				for i := 0; i < ap.Gabor1Shape.Y; i++ {
					for ti := 0; ti < nf; ti++ {
						val0 := ap.Gabor1Out.FloatVal([]int{ti, 0, i, s, ch})
						dout.SetFloat([]int{ti, 0, s, i}, val0)
						val1 := ap.Gabor1Out.FloatVal([]int{ti, 1, i, s, ch})
						dout.SetFloat([]int{ti, 1, s, i}, val1)
					}
				}
			}
		}
	}

	if ap.Gabor2.On {
		cn := "AudProc" + "_mel_gabor2_raw" + colSfx // column name
		col := dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor2.NFilters, 2, ap.Gabor2Shape.X, ap.Gabor2Shape.Y}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}
		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpace(4, []int{dt.Rows - 1})
			if err != nil {
				fmt.Printf("MelOutputToTable: mel_gabor2_raw subspacing error")
				return false
			}
			nf := ap.Gabor2.NFilters
			for s := 0; s < ap.Gabor2Shape.X; s++ {
				for i := 0; i < ap.Gabor2Shape.Y; i++ {
					for ti := 0; ti < nf; ti++ {
						val0 := ap.Gabor2Raw.FloatVal([]int{ti, 0, i, s, ch})
						dout.SetFloat([]int{ti, 0, s, i}, val0)
						val1 := ap.Gabor2Raw.FloatVal([]int{ti, 1, i, s, ch})
						dout.SetFloat([]int{ti, 1, s, i}, val1)
					}
				}
			}
		}

		cn = "AudProc" + "_mel_gabor2" + colSfx // column name
		col = dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor2.NFilters, 2, ap.Gabor2Shape.X, ap.Gabor2Shape.Y}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}
		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpace(4, []int{dt.Rows - 1})
			if err != nil {
				fmt.Printf("MelOutputToTable: mel_gabor2 subspacing error")
				return false
			}
			nf := ap.Gabor2.NFilters
			for s := 0; s < ap.Gabor2Shape.X; s++ {
				for i := 0; i < ap.Gabor2Shape.Y; i++ {
					for ti := 0; ti < nf; ti++ {
						val0 := ap.Gabor2Out.FloatVal([]int{ti, 0, i, s, ch})
						dout.SetFloat([]int{ti, 0, s, i}, val0)
						val1 := ap.Gabor2Out.FloatVal([]int{ti, 1, i, s, ch})
						dout.SetFloat([]int{ti, 1, s, i}, val1)
					}
				}
			}
		}
	}

	if ap.Gabor3.On {
		cn := "AudProc" + "_mel_gabor3_raw" + colSfx // column name
		col := dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor3.NFilters, 2, ap.Gabor3Shape.X, ap.Gabor3Shape.Y}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}
		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpace(4, []int{dt.Rows - 1})
			if err != nil {
				fmt.Printf("MelOutputToTable: mel_gabor3_raw subspacing error")
				return false
			}
			nf := ap.Gabor3.NFilters
			for s := 0; s < ap.Gabor3Shape.X; s++ {
				for i := 0; i < ap.Gabor3Shape.Y; i++ {
					for ti := 0; ti < nf; ti++ {
						val0 := ap.Gabor3Raw.FloatVal([]int{ti, 0, i, s, ch})
						dout.SetFloat([]int{ti, 0, s, i}, val0)
						val1 := ap.Gabor3Raw.FloatVal([]int{ti, 1, i, s, ch})
						dout.SetFloat([]int{ti, 1, s, i}, val1)
					}
				}
			}
		}

		cn = "AudProc" + "_mel_gabor3" + colSfx // column name
		col = dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor3.NFilters, 2, ap.Gabor3Shape.X, ap.Gabor3Shape.Y}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}
		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpace(4, []int{dt.Rows - 1})
			if err != nil {
				fmt.Printf("MelOutputToTable: mel_gabor3 subspacing error")
				return false
			}
			nf := ap.Gabor3.NFilters
			for s := 0; s < ap.Gabor3Shape.X; s++ {
				for i := 0; i < ap.Gabor3Shape.Y; i++ {
					for ti := 0; ti < nf; ti++ {
						val0 := ap.Gabor3Out.FloatVal([]int{ti, 0, i, s, ch})
						dout.SetFloat([]int{ti, 0, s, i}, val0)
						val1 := ap.Gabor3Out.FloatVal([]int{ti, 1, i, s, ch})
						dout.SetFloat([]int{ti, 1, s, i}, val1)
					}
				}
			}
		}
	}

	if ap.Mfcc.On {
		cn = "AudProc" + "_mel_mfcc" + colSfx // column name
		col = dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, int(ap.Input.TotalSteps), ap.Mfcc.NCoeff}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}
		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpace(2, []int{dt.Rows - 1})
			if err != nil {
				fmt.Printf("MelOutputToTable: mel_mfcc subspacing error")
				return false
			}
			for s := 0; s < int(ap.Input.TotalSteps); s++ {
				for i := 0; i < ap.Mfcc.NCoeff; i++ {
					val := ap.MfccDctTrialOut.FloatVal([]int{i, s, ch})
					dout.SetFloat([]int{s, i}, val)
				}
			}
		}
	}
	return true
}
