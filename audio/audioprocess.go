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
	"github.com/emer/etable/minmax"
	"github.com/emer/leabra/leabra"
	"gonum.org/v1/gonum/fourier"
)

// InitSound
func (ap *AuditoryProc) InitSound() bool {
	ap.InputPos = 0
	//ap.SoundFull = nil
	return true
}

// InitFromSound loads a sound and sets the Input channel vars and sample rate
func (ais *Input) InitFromSound(snd *Sound, nChannels int, channel int) {
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

// AudRenormSpec holds the auditory renormalization parameters
type AudRenormSpec struct {
	On          bool    `desc:"perform renormalization of this level of the auditory signal"`
	RenormMin   float32 `viewif:"On" desc:"minimum value to use for renormalization -- you must experiment with range of inputs to determine appropriate values"`
	RenormMax   float32 `viewif:"On" desc:"maximum value to use for renormalization -- you must experiment with range of inputs to determine appropriate values"`
	RenormScale float32 `inactive:"+" desc:"1.0 / (ren_max - ren_min)"`
}

//Initialize initializes the AudRenormSpec
func (ar *AudRenormSpec) Initialize() {
	ar.On = true
	ar.RenormMin = -5.0
	ar.RenormMax = 9.0
	ar.RenormScale = 1.0 / (ar.RenormMax - ar.RenormMin)
}

// MelFBankSpec contains mel frequency feature bank sampling parameters
type MelFBankSpec struct {
	On       bool    `desc:"perform mel-frequency filtering of the fft input"`
	NFilters int     `viewif:"On" def:"32,26" desc:"number of Mel frequency filters to compute"`
	LoHz     float32 `viewif:"On" def:"120,300" desc:"low frequency end of mel frequency spectrum"`
	HiHz     float32 `viewif:"On" def:"10000,8000" desc:"high frequency end of mel frequency spectrum -- must be <= sample_rate / 2 (i.e., less than the Nyquist frequencY"`
	LogOff   float32 `viewif:"On" def:"0" desc:"on add this amount when taking the log of the Mel filter sums to produce the filter-bank output -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs"`
	LogMin   float32 `viewif:"On" def:"-10" desc:"minimum value a log can produce -- puts a lower limit on log output"`
	LoMel    float32 `viewif:"On" inactive:"+" desc:" low end of mel scale in mel units"`
	HiMel    float32 `viewif:"On" inactive:"+" desc:" high end of mel scale in mel units"`
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
	NCoeff int  `def:"13" desc:"number of mfcc coefficients to output -- typically 1/2 of the number of filterbank features"`
}

//Initialize initializes the MelCepstrumSpec
func (mc *MelCepstrumSpec) Initialize() {
	mc.On = false
	mc.NCoeff = 13
}

type AuditoryProc struct {
	Data          *etable.Table   `desc:"data table for saving filter results for viewing and applying to networks etc"`
	Input         Input           `desc:"specifications of the raw auditory input"`
	Dft           AudDftSpec      `desc:"specifications for how to compute the discrete fourier transform (DFT, using FFT)"`
	MelFBank      MelFBankSpec    `desc:"specifications of the mel feature bank frequency sampling of the DFT (FFT) of the input sound"`
	FBankRenorm   AudRenormSpec   `viewif:"MelFBank.On=true desc:"renormalization parmeters for the mel_fbank values -- performed prior to further processing"`
	Gabor1        Gabor           `viewif:"MelFBank.On=true desc:"full set of frequency / time gabor filters -- first size"`
	Gabor2        Gabor           `viewif:"MelFBank.On=true desc:"full set of frequency / time gabor filters -- second size"`
	Gabor3        Gabor           `viewif:"MelFBank.On=true desc:"full set of frequency / time gabor filters -- third size"`
	Gabor1Filters etensor.Float32 `inactive:"+" desc:" #NO_SAVE full gabor filters"`
	Gabor2Filters etensor.Float32 `inactive:"+" desc:" #NO_SAVE full gabor filters"`
	Gabor3Filters etensor.Float32 `inactive:"+" desc:" #NO_SAVE full gabor filters"`

	Mfcc     MelCepstrumSpec `viewif:"MelFBank.On=true desc:"specifications of the mel cepstrum discrete cosine transform of the mel fbank filter features"`
	UseInhib bool            `viewif:"Gabor1.On=true" desc:"k-winner-take-all inhibitory dynamics for the time-gabor output"`

	// Filters
	DftSize        int `inactive:"+" desc:" #NO_SAVE full size of fft output -- should be input.win_samples"`
	DftUse         int `inactive:"+" desc:" #NO_SAVE number of dft outputs to actually use -- should be dft_size / 2 + 1"`
	MelNFiltersEff int `inactive:"+" desc:" #NO_SAVE effective number of mel filters: mel.n_filters + 2"`

	MelPtsMel        etensor.Float32 `inactive:"+" desc:" #NO_SAVE [mel_n_filters_eff] scale points in mel units (mels)"`
	MelPtsHz         etensor.Float32 `inactive:"+" desc:" #NO_SAVE [mel_n_filters_eff] mel scale points in hz units"`
	MelPtsBin        etensor.Int32   `inactive:"+" desc:" #NO_SAVE [mel_n_filters_eff] mel scale points in fft bins"`
	MelFilters       etensor.Float32 `inactive:"+" desc:" #NO_SAVE [mel_filt_max_bins][mel.n_filters] the actual filters for actual number of mel filters"`
	MelFilterMaxBins int             `inactive:"+" desc:" #NO_SAVE maximum number of bins for mel filter -- number of bins in highest filter"`

	// Outputs
	FirstStep     bool        `inactive:"+" desc:" #NO_SAVE is this the first step of processing -- turns of prv smoothing of dft power"`
	InputPos      int         `inactive:"+" desc:" #NO_SAVE current position in the sound_full input -- in terms of sample number"`
	TrialStartPos int         `inactive:"+" desc:" #NO_SAVE starting position of the current trial -- in terms of sample number"`
	TrialEndPos   int         `inactive:"+" desc:" #NO_SAVE ending position of the current trial -- in terms of sample number"`
	Gabor1Shape   image.Point `viewif:Gabor1.On=true" inactive:"+" desc:"overall geometry of gabor1 output (group-level geometry -- feature / unit level geometry is n_features, 2)"`
	Gabor2Shape   image.Point `viewif:Gabor2.On=true" inactive:"+" desc:"overall geometry of gabor2 output (group-level geometry -- feature / unit level geometry is n_features, 2)"`
	Gabor3Shape   image.Point `viewif:Gabor3.On=true" inactive:"+" desc:"overall geometry of gabor3 output (group-level geometry -- feature / unit level geometry is n_features, 2)"`

	SoundFull           etensor.Float32    `inactive:"+" desc:" #NO_SAVE the full sound input obtained from the sound input"`
	WindowIn            etensor.Float32    `inactive:"+" desc:" #NO_SAVE [input.win_samples] the raw sound input, one channel at a time"`
	DftOut              etensor.Complex128 `inactive:"+" desc:" #NO_SAVE [dft_size] discrete fourier transform (fft) output complex representation"`
	DftPowerOut         etensor.Float32    `inactive:"+" desc:" #NO_SAVE [dft_use] power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPowerOut      etensor.Float32    `inactive:"+" desc:" #NO_SAVE [dft_use] log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftPowerTrialOut    etensor.Float32    `inactive:"+" desc:" #NO_SAVE [dft_use][input.total_steps][input.channels] full trial's worth of power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPowerTrialOut etensor.Float32    `inactive:"+" desc:" #NO_SAVE [dft_use][input.total_steps][input.channels] full trial's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`

	MelFBankOut      etensor.Float32 `inactive:"+" desc:" #NO_SAVE [mel.n_filters] mel scale transformation of dft_power, using triangular filters, resulting in the mel filterbank output -- the natural log of this is typically applied"`
	MelFBankTrialOut etensor.Float32 `inactive:"+" desc:" #NO_SAVE [mel.n_filters][input.total_steps][input.channels] full trial's worth of mel feature-bank output -- only if using gabors"`

	//GaborGci  etensor.Float32 `inactive:"+" desc:" #NO_SAVE inhibitory conductances, for computing kwta"`
	Gabor1Raw etensor.Float32 `inactive:"+" desc:" #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] raw output of gabor1 -- full trial's worth of gabor steps"`
	Gabor1Out etensor.Float32 `inactive:"+" desc:" #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] post-kwta output of full trial's worth of gabor steps"`
	Gabor2Raw etensor.Float32 `inactive:"+" desc:" #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] raw output of gabor1 -- full trial's worth of gabor steps"`
	Gabor2Out etensor.Float32 `inactive:"+" desc:" #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] post-kwta output of full trial's worth of gabor steps"`
	Gabor3Raw etensor.Float32 `inactive:"+" desc:" #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] raw output of gabor1 -- full trial's worth of gabor steps"`
	Gabor3Out etensor.Float32 `inactive:"+" desc:" #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] post-kwta output of full trial's worth of gabor steps"`

	MfccDctOut      etensor.Float32 `inactive:"+" desc:" #NO_SAVE discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
	MfccDctTrialOut etensor.Float32 `inactive:"+" desc:" #NO_SAVE full trial's worth of discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`

	NCycles int
	Layer   *leabra.Layer
}

// InitFilters
func (ap *AuditoryProc) InitGaborFilters() {
	if ap.Gabor1.On {
		ap.Gabor1.InitFilters(&ap.Gabor1Filters)
	}
	if ap.Gabor2.On {
		ap.Gabor2.InitFilters(&ap.Gabor2Filters)
	}
	if ap.Gabor3.On {
		ap.Gabor3.InitFilters(&ap.Gabor3Filters)
	}
}

// InitFiltersMel
func (ap *AuditoryProc) InitFiltersMel() bool {
	ap.MelNFiltersEff = ap.MelFBank.NFilters + 2
	ap.MelPtsMel.SetShape([]int{ap.MelNFiltersEff}, nil, nil)
	ap.MelPtsHz.SetShape([]int{ap.MelNFiltersEff}, nil, nil)
	ap.MelPtsBin.SetShape([]int{ap.MelNFiltersEff}, nil, nil)

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
	ap.MelFilters.SetShape([]int{ap.MelFBank.NFilters, ap.MelFilterMaxBins}, nil, nil)

	for f := 0; f < ap.MelFBank.NFilters; f++ {
		mnbin := int(ap.MelPtsBin.Value1D(f))
		pkbin := int(ap.MelPtsBin.Value1D(f + 1))
		mxbin := int(ap.MelPtsBin.Value1D(f + 2))
		pkmin := pkbin - mnbin
		pkmax := mxbin - pkbin

		fi := 0
		bin := 0
		for bin = mnbin; bin <= pkbin; bin, fi = bin+1, fi+1 {
			fval := float32((bin - mnbin) / pkmin)
			ap.MelFilters.SetFloat([]int{f, fi}, float64(fval))
		}
		for ; bin <= mxbin; bin, fi = bin+1, fi+1 {
			fval := float32((mxbin - bin) / pkmax)
			ap.MelFilters.SetFloat([]int{f, fi}, float64(fval))
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
			ap.Gabor1Raw.SetShape([]int{ap.Input.Channels, ap.Gabor1.NFilters, 2, ap.Gabor1Shape.Y, ap.Gabor1Shape.X}, nil, nil)
			ap.Gabor1Out.SetShape([]int{ap.Input.Channels, ap.Gabor1.NFilters, 2, ap.Gabor1Shape.Y, ap.Gabor1Shape.X}, nil, nil)
		}
		if ap.Gabor2.On {
			ap.Gabor2Raw.SetShape([]int{ap.Input.Channels, ap.Gabor2.NFilters, 2, ap.Gabor2Shape.Y, ap.Gabor2Shape.X}, nil, nil)
			ap.Gabor2Out.SetShape([]int{ap.Input.Channels, ap.Gabor2.NFilters, 2, ap.Gabor2Shape.Y, ap.Gabor2Shape.X}, nil, nil)
		}
		if ap.Gabor3.On {
			ap.Gabor3Raw.SetShape([]int{ap.Input.Channels, ap.Gabor3.NFilters, 2, ap.Gabor3Shape.Y, ap.Gabor3Shape.X}, nil, nil)
			ap.Gabor3Out.SetShape([]int{ap.Input.Channels, ap.Gabor3.NFilters, 2, ap.Gabor3Shape.Y, ap.Gabor3Shape.X}, nil, nil)
		}
		if ap.Mfcc.On {
			ap.MfccDctOut.SetShape([]int{ap.MelFBank.NFilters}, nil, nil)
			ap.MfccDctTrialOut.SetShape([]int{ap.MelFBank.NFilters, ap.Input.TotalSteps, ap.Input.Channels}, nil, nil)
		}
	}
	return true
}

// LoadSound initializes the AuditoryProc with the sound loaded from file by "Sound"
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

// StartNewSound sets a few vars to 0 before processing a new sound
func (ap *AuditoryProc) StartNewSound() bool {
	ap.FirstStep = true
	ap.InputPos = 0
	ap.TrialStartPos = 0
	ap.TrialEndPos = int(ap.TrialStartPos) + ap.Input.TrialSamples
	return true
}

// NeedsInit checks to see if we need to reinitialize AuditoryProc
func (ap *AuditoryProc) NeedsInit() bool {
	if int(ap.DftSize) != ap.Input.WinSamples || ap.MelNFiltersEff != ap.MelFBank.NFilters+2 {
		return true
	}
	return false

}

// UpdateConfig sets the shape of each gabor
func (ap *AuditoryProc) UpdateConfig() {
	ap.Gabor1Shape.X = ((ap.Input.TrialSteps - 1) / ap.Gabor1.SpaceTime) + 1
	ap.Gabor1Shape.Y = ((ap.MelFBank.NFilters - ap.Gabor1.SizeFreq - 1) / ap.Gabor1.SpaceFreq) + 1

	ap.Gabor2Shape.X = ((ap.Input.TrialSteps - 1) / ap.Gabor2.SpaceTime) + 1
	ap.Gabor2Shape.Y = ((ap.MelFBank.NFilters - ap.Gabor2.SizeFreq - 1) / ap.Gabor2.SpaceFreq) + 1

	ap.Gabor3Shape.X = ((ap.Input.TrialSteps - 1) / ap.Gabor3.SpaceTime) + 1
	ap.Gabor3Shape.Y = ((ap.MelFBank.NFilters - ap.Gabor3.SizeFreq - 1) / ap.Gabor3.SpaceFreq) + 1
}

// Init initializes AuditoryProc fields
func (ap *AuditoryProc) Init() bool {
	ap.Input.Initialize()
	ap.Dft.Initialize()
	ap.MelFBank.Initialize()
	ap.FBankRenorm.Initialize()
	ap.Gabor1.Initialize()
	ap.Gabor2.Initialize()
	ap.Gabor2.On = false
	ap.Gabor3.Initialize()
	ap.Gabor3.On = false
	ap.UpdateConfig()
	ap.Mfcc.Initialize()

	ap.DftSize = ap.Input.WinSamples
	ap.DftUse = ap.DftSize/2 + 1
	ap.InitFiltersMel()
	ap.InitGaborFilters()
	ap.InitOutMatrix()
	ap.Data = &etable.Table{}
	ap.InitDataTable()
	ap.InitSound()
	ap.NCycles = 20
	ap.UseInhib = false
	return true
}

// InitDataTable readies ap.Data, an etable.etable
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

// InitDataTableChan initializes ap.Data by channel
func (ap *AuditoryProc) InitDataTableChan(ch int) bool {
	if ap.MelFBank.On {
		ap.MelOutputToTable(ap.Data, ch, true)
	}
	return true
}

// InputStepsLeft returns the number of steps left to process in the current input sound
func (ap *AuditoryProc) InputStepsLeft() int {
	samplesLeft := len(ap.SoundFull.Values) - ap.InputPos
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
		ap.TrialStartPos = ap.InputPos - ap.Input.StepSamples*ap.Input.BorderSteps
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

// FftReal
func (ap *AuditoryProc) FftReal(out etensor.Complex128, in etensor.Float32) bool {
	if !out.IsEqual(&in.Shape) {
		fmt.Printf("FftReal: out shape different from in shape - modifying out to match")
		out.CopyShape(&in.Shape)
	}
	if out.NumDims() == 1 {
		for i := 0; i < out.Len(); i++ {
			out.SetFloat1D(i, in.FloatVal1D(i))
			out.SetFloat1DImag(i, 0)
		}
	}
	return true
}

// DftInput applies dft (fft) to input
func (ap *AuditoryProc) DftInput(windowInVals []float64) {
	ap.FftReal(ap.DftOut, ap.WindowIn)
	fft := fourier.NewCmplxFFT(ap.DftOut.Len())
	ap.DftOut.Values = fft.Coefficients(nil, ap.DftOut.Values)
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
	for f := 0; f < int(ap.MelFBank.NFilters); f, mi = f+1, mi+1 { // f is filter
		minBin := ap.MelPtsBin.Value1D(f)
		maxBin := ap.MelPtsBin.Value1D(f + 2)

		sum := float32(0)
		fi := 0
		for bin := minBin; bin < maxBin; bin, fi = bin+1, fi+1 {
			fVal := ap.MelFilters.Value([]int{mi, fi})
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
		ap.MelFBankOut.SetFloat1D(mi, float64(val))
		ap.MelFBankTrialOut.Set([]int{mi, step, ch}, val)
	}
}

// FilterTrial process filters that operate over an entire trial at a time
func (ap *AuditoryProc) FilterTrial(ch int) bool {
	if ap.Gabor1.On {
		ap.GaborFilter(ch, ap.Gabor1, ap.Gabor1Filters, ap.Gabor1Raw, ap.Gabor1Out)
	}
	if ap.Gabor2.On {
		ap.GaborFilter(ch, ap.Gabor2, ap.Gabor2Filters, ap.Gabor2Raw, ap.Gabor2Out)
	}
	if ap.Gabor3.On {
		ap.GaborFilter(ch, ap.Gabor3, ap.Gabor3Filters, ap.Gabor3Raw, ap.Gabor3Out)
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
func (ap *AuditoryProc) GaborFilter(ch int, spec Gabor, filters etensor.Float32, outRaw etensor.Float32, out etensor.Float32) {
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
		inSt := s - tOff
		if tIdx > outRaw.Dim(4) {
			fmt.Printf("GaborFilter: time index %v out of range: %v", tIdx, outRaw.Dim(3))
			break
		}

		fIdx := 0
		for flt := fMin; flt < fMax; flt, fIdx = flt+spec.SpaceFreq, fIdx+1 {
			if fIdx > outRaw.Dim(3) {
				fmt.Printf("GaborFilter: freq index %v out of range: %v", tIdx, outRaw.Dim(2))
				break
			}
			nf := spec.NFilters
			for fi := int(0); fi < nf; fi++ {
				fSum := float32(0.0)
				for ff := int(0); ff < spec.SizeFreq; ff++ {
					for ft := int(0); ft < spec.SizeTime; ft++ {
						fVal := filters.Value([]int{ft, ff, fi})
						iVal := ap.MelFBankTrialOut.Value([]int{flt + ff, inSt + ft, ch})
						fSum += fVal * iVal
					}
				}
				pos := fSum >= 0.0
				act := spec.Gain * math32.Abs(fSum)
				if pos {
					outRaw.SetFloat([]int{ch, fi, 0, fIdx, tIdx}, float64(act))
					outRaw.SetFloat([]int{ch, fi, 1, fIdx, tIdx}, 0)
					out.SetFloat([]int{ch, fi, 0, fIdx, tIdx}, float64(act))
					out.SetFloat([]int{ch, fi, 1, fIdx, tIdx}, 0)
				} else {
					outRaw.SetFloat([]int{ch, fi, 0, fIdx, tIdx}, 0)
					outRaw.SetFloat([]int{ch, fi, 1, fIdx, tIdx}, float64(act))
					out.SetFloat([]int{ch, fi, 0, fIdx, tIdx}, 0)
					out.SetFloat([]int{ch, fi, 1, fIdx, tIdx}, float64(act))
				}
			}
		}
	}

	if ap.UseInhib {
		ap.Layer.Inhib.Pool.On = true
		rawSS := outRaw.SubSpace(outRaw.NumDims()-1, []int{ch}).(*etensor.Float32)
		outSS := out.SubSpace(outRaw.NumDims()-1, []int{ch}).(*etensor.Float32)

		// Chans are ion channels used in computing point-neuron activation function
		type Chans struct {
			E float32 `desc:"excitatory sodium (Na) AMPA channels activated by synaptic glutamate"`
			L float32 `desc:"constant leak (potassium, K+) channels -- determines resting potential (typically higher than resting potential of K)"`
			I float32 `desc:"inhibitory chloride (Cl-) channels activated by synaptic GABA"`
			K float32 `desc:"gated / active potassium channels -- typically hyperpolarizing relative to leak / rest"`
		}

		type ActParams struct {
			Gbar       Chans `view:"inline" desc:"[Defaults: 1, .2, 1, 1] maximal conductances levels for channels"`
			Erev       Chans `view:"inline" desc:"[Defaults: 1, .3, .25, .1] reversal potentials for each channel"`
			ErevSubThr Chans `inactive:"+" view:"-" desc:"Erev - Act.Thr for each channel -- used in computing GeThrFmG among others"`
		}

		ac := ActParams{}
		ac.Gbar.E = 1.0
		ac.Gbar.L = 0.2
		ac.Gbar.I = 1.0
		ac.Gbar.K = 1.0
		ac.Erev.E = 1.0
		ac.Erev.L = 0.3
		ac.Erev.I = 0.25
		ac.Erev.K = 0.1
		// these really should be calculated - see update method in Act
		ac.ErevSubThr.E = 0.5
		ac.ErevSubThr.L = -0.19999999
		ac.ErevSubThr.I = -0.25
		ac.ErevSubThr.K = -0.4

		xx1 := leabra.XX1Params{}
		xx1.Defaults()

		inhibPars := leabra.FFFBParams{}
		inhibPars.Defaults()
		inhib := leabra.FFFBInhib{}

		//max_delta_crit := float32(.005)

		values := rawSS.Values // these are ge
		acts := make([]float32, 0)
		acts = append(acts, values...)
		avgMaxGe := minmax.AvgMax32{}
		avgMaxAct := minmax.AvgMax32{}
		for i, ge := range acts {
			avgMaxGe.UpdateVal(ge, i)
		}

		fmt.Println()
		cycles := 20
		for cy := 0; cy < cycles; cy++ {
			inhibPars.Inhib(avgMaxGe.Avg, avgMaxGe.Max, avgMaxAct.Avg, &inhib)
			fmt.Printf("geAvg: %v, geMax: %v, actMax: %v\n", avgMaxGe.Avg, avgMaxGe.Max, avgMaxAct.Max)
			geThr := float32((ac.Gbar.I*inhib.Gi*ac.ErevSubThr.I + ac.Gbar.L*ac.ErevSubThr.L) / ac.ErevSubThr.E)
			//max := avgMaxAct.Max
			for i, act := range acts {
				nwAct := xx1.NoisyXX1(act*float32(ac.Gbar.E) - geThr) // act is ge
				acts[i] = nwAct
				avgMaxAct.UpdateVal(nwAct, i)
			}
			//if avgMaxAct.Max - max < max_delta_crit {
			//	break
			//}
		}
		for i, act := range acts {
			//fmt.Printf("%v\n", act)
			outSS.SetFloat1D(i, float64(act))
		}
	}
}

// FilterWindow filters the current window_in input data according to current settings -- called by ProcessStep, but can be called separately
func (ap *AuditoryProc) FilterWindow(ch int, step int) bool {
	ap.FftReal(ap.DftOut, ap.WindowIn)
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

// MelOutputToTable mel filter bank to output table - this function puts all of the data into ap.Data.
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
		dout, err := colAsF32.SubSpaceTry(2, []int{dt.Rows - 1})
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
			dout, err := colAsF32.SubSpaceTry(2, []int{dt.Rows - 1})
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
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor1Shape.Y, ap.Gabor1Shape.X, 2, ap.Gabor1.NFilters}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}

		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpaceTry(4, []int{dt.Rows - 1})
			if err != nil {
				fmt.Printf("MelOutputToTable: mel_gabor1_raw subspacing error")
				return false
			}
			nf := ap.Gabor1.NFilters
			for s := 0; s < ap.Gabor1Shape.X; s++ {
				for i := 0; i < ap.Gabor1Shape.Y; i++ {
					for ti := 0; ti < nf; ti++ {
						val0 := ap.Gabor1Raw.FloatVal([]int{ch, ti, 0, i, s})
						dout.SetFloat([]int{i, s, 0, ti}, val0)
						val1 := ap.Gabor1Raw.FloatVal([]int{ch, ti, 1, i, s})
						dout.SetFloat([]int{i, s, 1, ti}, val1)
					}
				}
			}
		}

		cn = "AudProc" + "_mel_gabor1" + colSfx // column name
		col = dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor1Shape.Y, ap.Gabor1Shape.X, 2, ap.Gabor1.NFilters}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}
		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpaceTry(4, []int{dt.Rows - 1})
			if err != nil {
				fmt.Printf("MelOutputToTable: mel_gabor1 subspacing error")
				return false
			}
			nf := ap.Gabor1.NFilters
			for s := 0; s < ap.Gabor1Shape.X; s++ {
				for i := 0; i < ap.Gabor1Shape.Y; i++ {
					for ti := 0; ti < nf; ti++ {
						val0 := ap.Gabor1Out.FloatVal([]int{ch, ti, 0, i, s})
						dout.SetFloat([]int{i, s, 0, ti}, val0)
						val1 := ap.Gabor1Out.FloatVal([]int{ch, ti, 1, i, s})
						dout.SetFloat([]int{i, s, 1, ti}, val1)
					}
				}
			}
		}
	}

	if ap.Gabor2.On {
		cn := "AudProc" + "_mel_gabor2_raw" + colSfx // column name
		col := dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor2Shape.Y, ap.Gabor2Shape.X, 2, ap.Gabor2.NFilters}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}
		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpaceTry(4, []int{dt.Rows - 1})
			if err != nil {
				fmt.Printf("MelOutputToTable: mel_gabor2_raw subspacing error")
				return false
			}
			nf := ap.Gabor2.NFilters
			for s := 0; s < ap.Gabor2Shape.X; s++ {
				for i := 0; i < ap.Gabor2Shape.Y; i++ {
					for ti := 0; ti < nf; ti++ {
						val0 := ap.Gabor2Raw.FloatVal([]int{ch, ti, 0, i, s})
						dout.SetFloat([]int{i, s, 0, ti}, val0)
						val1 := ap.Gabor2Raw.FloatVal([]int{ch, ti, 1, i, s})
						dout.SetFloat([]int{i, s, 1, ti}, val1)
					}
				}
			}
		}

		cn = "AudProc" + "_mel_gabor2" + colSfx // column name
		col = dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor2Shape.Y, ap.Gabor2Shape.X, 2, ap.Gabor2.NFilters}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}
		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpaceTry(4, []int{dt.Rows - 1})
			if err != nil {
				fmt.Printf("MelOutputToTable: mel_gabor2 subspacing error")
				return false
			}
			nf := ap.Gabor2.NFilters
			for s := 0; s < ap.Gabor2Shape.X; s++ {
				for i := 0; i < ap.Gabor2Shape.Y; i++ {
					for ti := 0; ti < nf; ti++ {
						val0 := ap.Gabor2Out.FloatVal([]int{ch, ti, 0, i, s})
						dout.SetFloat([]int{i, s, 0, ti}, val0)
						val1 := ap.Gabor2Out.FloatVal([]int{ch, ti, 1, i, s})
						dout.SetFloat([]int{i, s, 1, ti}, val1)
					}
				}
			}
		}
	}

	if ap.Gabor3.On {
		cn := "AudProc" + "_mel_gabor3_raw" + colSfx // column name
		col := dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor3Shape.Y, ap.Gabor3Shape.X, 2, ap.Gabor3.NFilters}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}
		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpaceTry(4, []int{dt.Rows - 1})
			if err != nil {
				fmt.Printf("MelOutputToTable: mel_gabor3_raw subspacing error")
				return false
			}
			nf := ap.Gabor3.NFilters
			for s := 0; s < ap.Gabor3Shape.X; s++ {
				for i := 0; i < ap.Gabor3Shape.Y; i++ {
					for ti := 0; ti < nf; ti++ {
						val0 := ap.Gabor3Raw.FloatVal([]int{ch, ti, 0, i, s})
						dout.SetFloat([]int{i, s, 0, ti}, val0)
						val1 := ap.Gabor3Raw.FloatVal([]int{ch, ti, 1, i, s})
						dout.SetFloat([]int{i, s, 1, ti}, val1)
					}
				}
			}
		}

		cn = "AudProc" + "_mel_gabor3" + colSfx // column name
		col = dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor3Shape.Y, ap.Gabor3Shape.X, 2, ap.Gabor3.NFilters}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}
		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpaceTry(4, []int{dt.Rows - 1})
			if err != nil {
				fmt.Printf("MelOutputToTable: mel_gabor3 subspacing error")
				return false
			}
			nf := ap.Gabor3.NFilters
			for s := 0; s < ap.Gabor3Shape.X; s++ {
				for i := 0; i < ap.Gabor3Shape.Y; i++ {
					for ti := 0; ti < nf; ti++ {
						val0 := ap.Gabor3Out.FloatVal([]int{ch, ti, 0, i, s})
						dout.SetFloat([]int{i, s, 0, ti}, val0)
						val1 := ap.Gabor3Out.FloatVal([]int{ch, ti, 1, i, s})
						dout.SetFloat([]int{i, s, 1, ti}, val1)
					}
				}
			}
		}
	}

	// todo: this one needs to be checked
	if ap.Mfcc.On {
		cn = "AudProc" + "_mel_mfcc" + colSfx // column name
		col = dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor3Shape.Y, ap.Gabor3Shape.X, 2, ap.Gabor3.NFilters}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column not found or failed to be created")
				return false
			}
		}
		if fmtOnly == false {
			colAsF32 := dt.ColByName(cn).(*etensor.Float32)
			dout, err := colAsF32.SubSpaceTry(2, []int{dt.Rows - 1})
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
