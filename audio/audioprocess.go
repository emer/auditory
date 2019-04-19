// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package audio

import (
	"fmt"
	"github.com/emer/emergent/etensor"
	"math"
	"strconv"

	"github.com/chewxy/math32"
)

// AudInputSpec defines the sound input parameters for auditory processing
type AudInputSpec struct {
	WinMsec      float32 `desc:"#DEF_25 input window -- number of milliseconds worth of sound to filter at a time"`
	StepMsec     float32 `desc:"#DEF_5;10;12.5 input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample""`
	TrialMsec    float32 `desc:"#DEF_100 length of a full trial's worth of input -- total number of milliseconds to accumulate into a complete trial of activations to present to a network -- must be a multiple of step_msec -- input will be trial_msec / step_msec = trial_steps wide in the X axis, and number of filters in the Y axis"`
	BorderSteps  uint32  `desc:"number of steps before and after the trial window to preserve -- this is important when applying temporal filters that have greater temporal extent"`
	SampleRate   uint32  `desc:"rate of sampling in our sound input (e.g., 16000 = 16Khz) -- can initialize this from a taSound object using InitFromSound method"`
	Channels     uint16  `desc:"total number of channels to process"`
	Channel      uint16  `desc:"#CONDSHOW_ON_channels:1 specific channel to process, if input has multiple channels, and we only process one of them (-1 = process all)"`
	WinSamples   uint32  `desc:"#READ_ONLY #SHOW total number of samples to process (win_msec * .001 * sample_rate)"`
	StepSamples  uint32  `desc:"#READ_ONLY #SHOW total number of samples to step input by (step_msec * .001 * sample_rate)"`
	TrialSamples uint32  `desc:"#READ_ONLY #SHOW total number of samples in a trial  (trail_msec * .001 * sample_rate)"`
	TrialSteps   uint32  `desc:"#READ_ONLY #SHOW total number of steps in a trial  (trail_msec / step_msec)"`
	TotalSteps   uint32  `desc:"#READ_ONLY #SHOW 2*border_steps + trial_steps -- total in full window"`
}

//Init initializes the the AudInputSpec
func (ais *AudInputSpec) Initialize() {
	ais.WinMsec = 25.0
	ais.StepMsec = 5.0
	ais.TrialMsec = 100.0
	ais.BorderSteps = 12
	ais.SampleRate = 16000
	ais.Channels = 1
	ais.Channel = 0
	ais.ComputeSamples()
}

// ComputeSamples computes the sample counts based on time and sample rate
func (ais *AudInputSpec) ComputeSamples() {
	ais.WinSamples = MSecToSamples(ais.WinMsec, ais.SampleRate)
	ais.StepSamples = MSecToSamples(ais.StepMsec, ais.SampleRate)
	ais.TrialSamples = MSecToSamples(ais.TrialMsec, ais.SampleRate)
	ais.TrialSteps = uint32(math.Round(float64(ais.TrialMsec / ais.StepMsec)))
	ais.TotalSteps = 2*ais.BorderSteps + ais.TrialSteps
}

// MSecToSamples converts milliseconds to samples, in terms of sample_rate
func MSecToSamples(msec float32, rate uint32) uint32 {
	return uint32(math.Round(float64(msec) * 0.001 * float64(rate)))
}

// SamplesToMSec converts samples to milliseconds, in terms of sample_rate
func SamplesToMSec(samples uint32, rate uint32) float32 {
	return 1000.0 * float32(samples) / float32(rate)
}

// InitFromSound loads a sound and sets the AudInputSpec channel vars and sample rate
func (ais *AudInputSpec) InitFromSound(snd *Sound, nChannels uint16, channel uint16) {
	if snd == nil {
		fmt.Printf("InitFromSound: sound nil")
		return
	}

	ais.SampleRate = snd.SampleRate()
	ais.ComputeSamples()
	if nChannels < 1 {
		ais.Channels = snd.Channels()
	} else {
		ais.Channels = uint16(math32.Min(float32(nChannels), float32(ais.Channels)))
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

func (ad *AudDftSpec) Initialize() {
	ad.PreviousSmooth = 0
	ad.CurrentSmooth = 1.0 - ad.PreviousSmooth
	ad.LogPow = true
	ad.LogOff = 0
	ad.LogMin = -100
}

// MelFBankSpec contains mel frequency feature bank sampling parameters
type MelFBankSpec struct {
	On       bool    `desc:"perform mel-frequency filtering of the fft input"`
	LoHz     float32 `desc:"#DEF_120;300 #CONDSHOW_ON_on low frequency end of mel frequency spectrum"`
	HiHz     float32 `desc:"#DEF_10000;8000 #CONDSHOW_ON_on high frequency end of mel frequency spectrum -- must be <= sample_rate / 2 (i.e., less than the Nyquist frequency)"`
	NFilters uint32  `desc:"#DEF_32;26 #CONDSHOW_ON_on number of Mel frequency filters to compute"`
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
func FreqToBin(freq, nFft, sampleRate float32) uint32 {
	return uint32(math32.Floor(((nFft + 1) * freq) / sampleRate))
}

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

type AuditoryProc struct {
	//	enum SaveMode {               // how to add new data to the data table
	//	NONE_SAVE,                  // don't save anything at all -- overrides any more specific save guys and prevents any addition or modification to the data table
	//	FIRST_ROW,                  // always overwrite the first row -- does EnforceRows(1) if rows = 0
	//	ADD_ROW,                    // always add a new row and write to that, preserving a history of inputs over time -- should be reset at some interval!
	//};
	//
	//
	//	DataTableRef  data_table;     // data table for saving filter results for viewing and applying to networks etc
	//	SaveMode      save_mode;      // how to add new data to the data table
	Input    AudInputSpec `desc:"specifications of the raw auditory input"`
	Dft      AudDftSpec   `desc:"specifications for how to compute the discrete fourier transform (DFT, using FFT)"`
	MelFBank MelFBankSpec `desc:"specifications of the mel feature bank frequency sampling of the DFT (FFT) of the input sound"`
	//	AudRenormSpec fbank_renorm;   // #CONDSHOW_ON_mel_fbank.on renormalization parmeters for the mel_fbank values -- performed prior to further processing
	//	AudGaborSpec  gabor1;    // #CONDSHOW_ON_mel_fbank.on full set of frequency / time gabor filters -- first size
	//	AudGaborSpec  gabor2;    // #CONDSHOW_ON_mel_fbank.on full set of frequency / time gabor filters -- second size
	//	AudGaborSpec  gabor3;    // #CONDSHOW_ON_mel_fbank.on full set of frequency / time gabor filters -- third size
	//	MelCepstrumSpec mfcc;         // #CONDSHOW_ON_mel_fbank.on specifications of the mel cepstrum discrete cosine transform of the mel fbank filter features
	//	V1KwtaSpec    gabor_kwta;     // #CONDSHOW_ON_gabor1.on k-winner-take-all inhibitory dynamics for the time-gabor output

	// Filters
	DftSize        uint32 `desc:"#READ_ONLY #NO_SAVE full size of fft output -- should be input.win_samples"`
	DftUse         uint32 `desc:"#READ_ONLY #NO_SAVE number of dft outputs to actually use -- should be dft_size / 2 + 1"`
	MelNFiltersEff uint32 `desc:"#READ_ONLY #NO_SAVE effective number of mel filters: mel.n_filters + 2"`

	MelPtsMel        etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel_n_filters_eff] scale points in mel units (mels)"`
	MetPtsHz         etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel_n_filters_eff] mel scale points in hz units"`
	MelPtsBin        etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel_n_filters_eff] mel scale points in fft bins"`
	MelFilterMaxBins uint32          `desc:"#READ_ONLY #NO_SAVE maximum number of bins for mel filter -- number of bins in highest filter"`
	MelFilters       etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel_filt_max_bins][mel.n_filters] the actual filters for actual number of mel filters"`

	Gabor1Filters etensor.Float32 `desc:"#READ_ONLY #NO_SAVE full gabor filters"`
	Gabor2Filters etensor.Float32 `desc:"#READ_ONLY #NO_SAVE full gabor filters"`
	Gabor3Filters etensor.Float32 `desc:"#READ_ONLY #NO_SAVE full gabor filters"`

	// Outputs
	FirstStep     bool   `desc:"#READ_ONLY #NO_SAVE #SHOW is this the first step of processing -- turns of prv smoothing of dft power"`
	InputPos      uint32 `desc:"#READ_ONLY #NO_SAVE #SHOW current position in the sound_full input -- in terms of sample number"`
	TrialStartPos uint32 `desc:" #READ_ONLY #NO_SAVE #SHOW starting position of the current trial -- in terms of sample number"`
	TrialEndPos   uint32 `desc:"#READ_ONLY #NO_SAVE #SHOW ending position of the current trial -- in terms of sample number"`
	XYNGeom               gabor1_geom; // #CONDSHOW_ON_gabor1.on #READ_ONLY #SHOW overall geometry of gabor1 output (group-level geometry -- feature / unit level geometry is n_features, 2)
	XYNGeom               gabor2_geom; // #CONDSHOW_ON_gabor2.on #READ_ONLY #SHOW overall geometry of gabor1 output (group-level geometry -- feature / unit level geometry is n_features, 2)
	XYNGeom               gabor3_geom; // #CONDSHOW_ON_gabor3.on #READ_ONLY #SHOW overall geometry of gabor1 output (group-level geometry -- feature / unit level geometry is n_features, 2)

	SoundFull etensor.Float32 `desc:"#READ_ONLY #NO_SAVE the full sound input obtained from the sound input"`
	WindowIn  etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [input.win_samples] the raw sound input, one channel at a time"`

	DftOut etensor.ComplexFloat32 `desc:"#READ_ONLY #NO_SAVE [2, dft_size] discrete fourier transform (fft) output complex representation"`

	DftPowerOut etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [dft_use] power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPowerOut etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [dft_use] log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftPowerTrialOut        etensor.Float32 `desc"#READ_ONLY #NO_SAVE [dft_use][input.total_steps][input.channels] full trial's worth of power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPowerTrialOut etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [dft_use][input.total_steps][input.channels] full trial's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`

	mel_fbank_out etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel.n_filters] mel scale transformation of dft_power, using triangular filters, resulting in the mel filterbank output -- the natural log of this is typically applied"`

	mel_fbank_trial_out etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel.n_filters][input.total_steps][input.channels] full trial's worth of mel feature-bank output -- only if using gabors"`
	GaborGci etensor.Float32       `desc:"#READ_ONLY #NO_SAVE inhibitory conductances, for computing kwta"`
	Gabor1TrialRaw etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] raw output of gabor1 -- full trial's worth of gabor steps"`
	Gabor1TrialOut etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] post-kwta output of full trial's worth of gabor steps"`

	Gabor2TrialRaw etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] raw output of gabor1 -- full trial's worth of gabor steps"`
	Gabor2TrialOut etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] post-kwta output of full trial's worth of gabor steps"`

	Gabor3TrialRaw etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] raw output of gabor1 -- full trial's worth of gabor steps"`
	Gabor3TrialOut etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] post-kwta output of full trial's worth of gabor steps"`

	mfcc_dct_out etensor.Float32 `desc:"#READ_ONLY #NO_SAVE discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
	mfcc_dct_trial_out etensor.Float32 `desc:"#READ_ONLY #NO_SAVE full trial's worth of discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
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

	if snd.SampleRate() != ap.Input.SampleRate {
		fmt.Printf("LoadSound: sample rate does not match sound -- re-initializing with new rate of: %v", strconv.Itoa(int(snd.SampleRate())))
		ap.Input.SampleRate = snd.SampleRate()
		needsInit = true
	}

	if needsInit {
		ap.Init()
	}

	if ap.Input.Channels > 1 {
		snd.SoundToMatrix(soundFull, -1)
	} else {
		snd.SoundToMatrix(soundFull, Input.Channel)
	}
	snd.StartNewSound();
	return true;
}

func (ap *AuditoryProc) StartNewSound() bool {
	ap.FirstStep = true
	ap.InputPos = 0
	ap.TrialStartPos = 0
	ap.TrialEndPos = ap.TrialStartPos + ap.Input.TrialSamples
	ap.dftPowerTrialOut.InitVals(0.0)
	if ap.Dft.LogPow {
		dft_log_power_trial_out.InitVals(0.0)
	}

	if (ap.MelFBank.On) {
		mel_fbank_trial_out.InitVals(0.0
		f);
		if (gabor1.on) {
			gabor1_trial_raw.InitVals(0.0
			f);
			gabor1_trial_out.InitVals(0.0
			f);
		}
		if (gabor2.on) {
			gabor2_trial_raw.InitVals(0.0
			f);
			gabor2_trial_out.InitVals(0.0
			f);
		}
		if (gabor3.on) {
			gabor3_trial_raw.InitVals(0.0
			f);
			gabor3_trial_out.InitVals(0.0
			f);
		}
		if (mfcc.on) {
			mfcc_dct_trial_out.InitVals(0.0
			f);
		}
	}
	return true;
}

func (ap *AuditoryProc) NeedsInit() bool {
	if ap.DftSize != ap.Input.WinSamples || ap.MelNFiltersEff != ap.MelFBank.NFilters+2 {
		return true
	}
	return false

}

func (ap *AuditoryProc) Init() bool {
	ap.UpdateConfig();
	ap.InitFilters();
	ap.InitOutMatrix();
	ap.InitDataTable();
	ap.InitSound();
	return true
}
