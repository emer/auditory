// Copyright (c) 2021, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sound

import (
	"errors"
	"fmt"
	"log"
	"math"
	"runtime"

	"github.com/emer/auditory/agabor"
	"github.com/emer/auditory/dft"
	"github.com/emer/auditory/mel"
	"github.com/emer/etable/etable"
	"github.com/emer/etable/etensor"
	"github.com/emer/leabra/fffb"
	"github.com/emer/vision/kwta"
)

// Params defines the sound input parameters for auditory processing
type Params struct {

	// [def: 25] input window -- number of milliseconds worth of sound to filter at a time
	WinMs float64 `default:"25" desc:"input window -- number of milliseconds worth of sound to filter at a time"`

	// [def: 5,10,12.5] input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample
	StepMs float64 `default:"5,10,12.5" desc:"input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample"`

	// [def: 100] length of full segment's worth of input -- total number of milliseconds to accumulate into a complete segment -- must be a multiple of StepMs -- input will be SegmentMs / StepMs = SegmentSteps wide in the X axis, and number of filters in the Y axis
	SegmentMs float64 `default:"100" desc:"length of full segment's worth of input -- total number of milliseconds to accumulate into a complete segment -- must be a multiple of StepMs -- input will be SegmentMs / StepMs = SegmentSteps wide in the X axis, and number of filters in the Y axis"`

	// [def: 100] how far to move on each trial
	StrideMs float64 `default:"100" desc:"how far to move on each trial"`

	// [def: 6] [view: +] overlap with previous and next segment
	BorderSteps int `default:"6" view:"+" desc:"overlap with previous and next segment"`

	// [viewif: Channels=1] specific channel to process, if input has multiple channels, and we only process one of them (-1 = process all)
	Channel int `viewif:"Channels=1" desc:"specific channel to process, if input has multiple channels, and we only process one of them (-1 = process all)"`

	// number of samples to process each step
	WinSamples int `inactive:"+" desc:"number of samples to process each step"`

	// number of samples to step input by
	StepSamples int `inactive:"+" desc:"number of samples to step input by"`

	// number of samples in a segment
	SegmentSamples int `inactive:"+" desc:"number of samples in a segment"`

	// number of samples converted from StrideMS
	StrideSamples int `inactive:"+" desc:"number of samples converted from StrideMS"`

	// includes border steps on both sides
	SegmentSteps int `inactive:"+" desc:"includes border steps on both sides"`

	// pre-calculated start position for each step
	Steps []int `inactive:"+" desc:"pre-calculated start position for each step"`
}

// ParamDefaults initializes the Input
func (se *SndEnv) ParamDefaults() {
	se.Params.WinMs = 25.0
	se.Params.StepMs = 10.0
	se.Params.SegmentMs = 100.0
	se.Params.Channel = 0
	se.Params.StrideMs = 100.0
	se.Params.BorderSteps = 2
}

type SndEnv struct {

	// name of this environment
	Nm string `desc:"name of this environment"`

	// description of this environment
	Dsc string `desc:"description of this environment"`

	// false turns off processing of this sound
	On bool `desc:"false turns off processing of this sound"`

	// specifications of the raw sensory input
	Sound  Wave `desc:"specifications of the raw sensory input"`
	Params Params

	// [view: no-inline]  the full sound input
	Signal etensor.Float64 `view:"no-inline" desc:" the full sound input"`

	// the number of segments in this sound file (based on current segment size)
	SegCnt int `desc:"the number of segments in this sound file (based on current segment size)"`

	//  [Input.WinSamples] the raw sound input, one channel at a time
	Window etensor.Float64 `inactive:"+" desc:" [Input.WinSamples] the raw sound input, one channel at a time"`

	DFT dft.Params

	// [view: -]  power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)
	Power etensor.Float64 `view:"-" desc:" power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)"`

	// [view: -]  log power of the dft, up to the nyquist liit frequency (1/2 input.WinSamples)
	LogPower etensor.Float64 `view:"-" desc:" log power of the dft, up to the nyquist liit frequency (1/2 input.WinSamples)"`

	// [view: no-inline]  full segment's worth of power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)
	PowerSegment etensor.Float64 `view:"no-inline" desc:" full segment's worth of power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`

	// [view: no-inline]  full segment's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)
	LogPowerSegment etensor.Float64 `view:"no-inline" desc:" full segment's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`

	// [view: no-inline]
	Mel mel.Params `view:"no-inline"`

	// [view: no-inline]  mel scale transformation of dft_power, resulting in the mel filterbank output -- the natural log of this is typically applied
	MelFBank etensor.Float64 `view:"no-inline" desc:" mel scale transformation of dft_power, resulting in the mel filterbank output -- the natural log of this is typically applied"`

	// [view: no-inline]  full segment's worth of mel feature-bank output
	MelFBankSegment etensor.Float64 `view:"no-inline" desc:" full segment's worth of mel feature-bank output"`

	// [view: no-inline]  the actual filters
	MelFilters etensor.Float64 `view:"no-inline" desc:" the actual filters"`

	// [view: no-inline]  sum of log power per segment step
	Energy etensor.Float64 `view:"no-inline" desc:" sum of log power per segment step"`

	// [view: no-inline]  discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients
	MFCCDCT etensor.Float64 `view:"no-inline" desc:" discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`

	// [view: no-inline]  full segment's worth of discrete cosine transform of log_mel_filter_out values, producing the final mel-frequency cepstral coefficients
	MFCCSegment etensor.Float64 `view:"no-inline" desc:" full segment's worth of discrete cosine transform of log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`

	// [view: no-inline]  MFCC deltas are the differences over time of the MFC coefficeints
	MFCCDeltas etensor.Float64 `view:"no-inline" desc:" MFCC deltas are the differences over time of the MFC coefficeints"`

	// [view: no-inline] MFCC delta deltas are the differences over time of the MFCC deltas
	MFCCDeltaDeltas etensor.Float64 `view:"no-inline" desc:"MFCC delta deltas are the differences over time of the MFCC deltas"`

	// [view: no-inline]  a set of gabor filter specifications, one spec per filter'
	GaborSpecs []agabor.Filter `view:"no-inline" desc:" a set of gabor filter specifications, one spec per filter'"`

	// the actual gabor filters, the first spec determines the size of all filters in the set
	GaborFilters agabor.FilterSet `desc:"the actual gabor filters, the first spec determines the size of all filters in the set"`

	// [view: no-inline] gabor filter table (view only)
	GaborTab etable.Table `view:"no-inline" desc:"gabor filter table (view only)"`

	// [view: +]  the number of neuron pools along the time dimension in the input layer
	GborOutPoolsX int `view:"+" desc:" the number of neuron pools along the time dimension in the input layer"`

	// [view: +]  the number of neuron pools along the frequency dimension in the input layer
	GborOutPoolsY int `view:"+" desc:" the number of neuron pools along the frequency dimension in the input layer"`

	// [view: +]  the number of neurons in a pool (typically the number of gabor filters) along the time dimension in the input layer
	GborOutUnitsX int `view:"+" desc:" the number of neurons in a pool (typically the number of gabor filters) along the time dimension in the input layer"`

	// [view: +]  the number of neurons in a pool along the frequency dimension in the input layer
	GborOutUnitsY int `view:"+" desc:" the number of neurons in a pool along the frequency dimension in the input layer"`

	// [view: no-inline]  raw output of Gabor -- full segment's worth of gabor steps
	GborOutput etensor.Float32 `view:"no-inline" desc:" raw output of Gabor -- full segment's worth of gabor steps"`

	// [view: no-inline]  post-kwta output of full segment's worth of gabor steps
	GborKwta etensor.Float32 `view:"no-inline" desc:" post-kwta output of full segment's worth of gabor steps"`

	// [view: no-inline] inhibition values for A1 KWTA
	Inhibs fffb.Inhibs `view:"no-inline" desc:"inhibition values for A1 KWTA"`

	// [view: no-inline] A1 simple extra Gi from neighbor inhibition tensor
	ExtGi etensor.Float32 `view:"no-inline" desc:"A1 simple extra Gi from neighbor inhibition tensor"`

	// neighborhood inhibition for V1s -- each unit gets inhibition from same feature in nearest orthogonal neighbors -- reduces redundancy of feature code
	NeighInhib kwta.NeighInhib `desc:"neighborhood inhibition for V1s -- each unit gets inhibition from same feature in nearest orthogonal neighbors -- reduces redundancy of feature code"`

	// kwta parameters, using FFFB form
	Kwta kwta.KWTA `desc:"kwta parameters, using FFFB form"`

	// if Kwta.On == true, call KwtaPool (true) or KwtaLayer (false)
	KwtaPool bool `desc:"if Kwta.On == true, call KwtaPool (true) or KwtaLayer (false)"`

	// display the gabor filtering result by time and then by filter, default is to order by filter and then time
	ByTime bool `desc:"display the gabor filtering result by time and then by filter, default is to order by filter and then time"`
}

// Defaults
func (se *SndEnv) Defaults() {
	se.ParamDefaults()
	se.On = true
	se.Mel.Defaults() // calls melfbank defaults
	se.Kwta.Defaults()
	se.KwtaPool = true
	se.ByTime = false
}

// Init sets various sound processing params based on default params and user overrides
func (se *SndEnv) Init() (err error) {
	sr := se.Sound.SampleRate()
	if sr <= 0 {
		fmt.Println("sample rate <= 0")
		err = errors.New("sample rate <= 0")
		return err
	}
	se.Params.WinSamples = MSecToSamples(se.Params.WinMs, sr)
	se.Params.StepSamples = MSecToSamples(se.Params.StepMs, sr)
	se.Params.SegmentSamples = MSecToSamples(se.Params.SegmentMs, sr)
	steps := int(math.Round(se.Params.SegmentMs / se.Params.StepMs))
	se.Params.SegmentSteps = steps + 2*se.Params.BorderSteps
	se.Params.StrideSamples = MSecToSamples(se.Params.StrideMs, sr)

	specs := agabor.Active(se.GaborSpecs)
	nfilters := len(specs)
	se.GaborFilters.Filters.SetShape([]int{nfilters, se.GaborFilters.SizeY, se.GaborFilters.SizeX}, nil, nil)
	agabor.ToTensor(specs, &se.GaborFilters)
	se.GaborFilters.ToTable(se.GaborFilters, &se.GaborTab) // note: view only, testing
	if se.GborOutPoolsX == 0 && se.GborOutPoolsY == 0 {    // 2D
		se.GborOutput.SetShape([]int{se.GborOutUnitsY, se.GborOutUnitsX}, nil, nil)
		se.ExtGi.SetShape([]int{se.GborOutUnitsY, se.GborOutUnitsX}, nil, nil) // passed in for each channel
	} else if se.GborOutPoolsX > 0 && se.GborOutPoolsY > 0 { // 4D
		se.GborOutput.SetShape([]int{se.GborOutPoolsY, se.GborOutPoolsX, se.GborOutUnitsY, se.GborOutUnitsX}, nil, nil)
		se.ExtGi.SetShape([]int{se.GborOutPoolsY, se.GborOutPoolsX, 2, nfilters}, nil, nil) // passed in for each channel
	} else {
		log.Println("GborOutPoolsX & GborOutPoolsY must both be == 0 or > 0 (i.e. 2D or 4D)")
		return err
	}
	se.GborOutput.SetMetaData("odd-row", "true")
	se.GborOutput.SetMetaData("grid-fill", ".9")
	se.GborKwta.CopyShapeFrom(&se.GborOutput)
	se.GborKwta.CopyMetaData(&se.GborOutput)

	winSamplesHalf := se.Params.WinSamples/2 + 1
	se.DFT.Defaults()
	se.Mel.InitFilters(se.Params.WinSamples, se.Sound.SampleRate(), &se.MelFilters) // call after non-default values are set!
	se.Window.SetShape([]int{se.Params.WinSamples}, nil, nil)
	se.Power.SetShape([]int{winSamplesHalf}, nil, nil)
	se.LogPower.CopyShapeFrom(&se.Power)
	se.PowerSegment.SetShape([]int{winSamplesHalf, se.Params.SegmentSteps}, nil, nil)
	if se.DFT.CompLogPow {
		se.LogPowerSegment.CopyShapeFrom(&se.PowerSegment)
	}

	// 2 reasons for this code
	// 1 - the amount of signal handed to the fft has a "border" (some extra signal) to avoid edge effects.
	// On the first step there is no signal to act as the "border" so we pad the data handed on the front.
	// 2 - signals needs to be aligned when the number when multiple signals are input (e.g. 100 and 300 ms)
	// so that the leading edge (right edge) is the same time point.
	// This code does this by generating negative offsets for the start of the processing.
	// Also see SndToWindow for the use of the step values
	stepsBack := se.Params.BorderSteps
	se.Params.Steps = make([]int, se.Params.SegmentSteps)
	for i := 0; i < se.Params.SegmentSteps; i++ {
		se.Params.Steps[i] = se.Params.StepSamples * (i - stepsBack)
	}

	se.MelFBank.SetShape([]int{se.Mel.FBank.NFilters}, nil, nil)
	se.MelFBankSegment.SetShape([]int{se.Mel.FBank.NFilters, se.Params.SegmentSteps}, nil, nil)
	se.Energy.SetShape([]int{se.Params.SegmentSteps}, nil, nil)
	if se.Mel.MFCC {
		se.MFCCDCT.SetShape([]int{se.Mel.FBank.NFilters}, nil, nil)
		se.MFCCSegment.SetShape([]int{se.Mel.NCoefs, se.Params.SegmentSteps}, nil, nil)
		se.MFCCDeltas.SetShape([]int{se.Mel.NCoefs, se.Params.SegmentSteps}, nil, nil)
		se.MFCCDeltaDeltas.SetShape([]int{se.Mel.NCoefs, se.Params.SegmentSteps}, nil, nil)
	}

	siglen := len(se.Signal.Values) - se.Params.SegmentSamples*se.Sound.Channels()
	siglen = siglen / se.Sound.Channels()
	se.SegCnt = siglen/se.Params.StrideSamples + 1 // add back the first segment subtracted at from siglen calculation
	return nil
}

// AdjustForSilence trims or adds silence
// add is the amount of random silence that should precede the start of the sequence.
// existing is the amount of silence preexisting at start of sound.
// offset is the amount of silence trimmed from or added to the existing silence.
// add and existing values are in milliseconds
func (se *SndEnv) AdjustForSilence(add, existing float64) (offset int) {
	sr := se.Sound.SampleRate()
	if sr <= 0 {
		fmt.Println("sample rate <= 0")
		return -1
	}

	offset = 0.0 // in milliseconds
	if add >= 0 {
		if add < existing {
			offset = int(existing - add)
			se.Signal.Values = se.Signal.Values[int(MSecToSamples(float64(offset), sr)):len(se.Signal.Values)]
		} else if add > existing {
			offset = int(add - existing)
			n := int(MSecToSamples(float64(offset), sr))
			silence := make([]float64, n)
			se.Signal.Values = append(silence, se.Signal.Values...)
		}
	}
	return offset
}

// ToTensor
func (se *SndEnv) ToTensor() bool {
	se.Sound.SoundToTensor(&se.Signal)
	return true
}

// ApplyNeighInhib - each unit gets inhibition from same feature in nearest orthogonal neighbors
func (se *SndEnv) ApplyNeighInhib() {
	if se.NeighInhib.On {
		//rawSS := se.GborOutput.SubSpace([]int{ch}).(*etensor.Float32)
		//extSS := se.ExtGi.SubSpace([]int{ch}).(*etensor.Float32)
		se.NeighInhib.Inhib4(&se.GborOutput, &se.ExtGi)
	} else {
		se.ExtGi.SetZeros()
	}
}

// ApplyKwta runs the kwta algorithm on the raw activations
func (se *SndEnv) ApplyKwta() {
	se.GborKwta.CopyFrom(&se.GborOutput)
	if se.Kwta.On {
		if se.KwtaPool == true {
			se.Kwta.KWTAPool(&se.GborOutput, &se.GborKwta, &se.Inhibs, &se.ExtGi)
		} else {
			se.Kwta.KWTALayer(&se.GborOutput, &se.GborKwta, &se.ExtGi)
		}
	}
}

//func (se *SndEnv) ApplyKwta(ch int) {
//	se.GborKwta.CopyFrom(&se.GborOutput)
//	if se.Kwta.On {
//		rawSS := se.GborOutput.SubSpace([]int{ch}).(*etensor.Float32)
//		kwtaSS := se.GborKwta.SubSpace([]int{ch}).(*etensor.Float32)
//		if se.KwtaPool == true {
//			se.Kwta.KWTAPool(rawSS, kwtaSS, &se.Inhibs, &se.ExtGi)
//		} else {
//			se.Kwta.KWTALayer(rawSS, kwtaSS, &se.ExtGi)
//		}
//	}
//}

// ProcessSegment processes the entire segment's input by processing a small overlapping set of samples on each pass
// The add argument allows for compensation if there are multiple sounds of different duration to different input layers
// of the network. For example, durations of 80 and 120 ms. Add half the difference (e.g. 20 ms) so the sounds are
// centered on the same moment of sound
func (se *SndEnv) ProcessSegment(segment, add int) {
	se.Power.SetZeros()
	se.LogPower.SetZeros()
	se.PowerSegment.SetZeros()
	se.LogPowerSegment.SetZeros()
	se.Energy.SetZeros()
	se.MelFBankSegment.SetZeros()
	if se.Mel.MFCC == true {
		se.MFCCSegment.SetZeros()
	}

	for s := 0; s < int(se.Params.SegmentSteps); s++ {
		err := se.ProcessStep(segment, s, add)
		if err != nil {
			fmt.Println(err)
			break
		}
	}
	for s := 0; s < se.Params.SegmentSteps; s++ {
		e := 0.0
		for f := 0; f < se.LogPowerSegment.Shape.Dim(1); f++ {
			e += se.LogPowerSegment.FloatValRowCell(s, f)
		}
		se.Energy.SetFloat1D(s, e)
	}

	if se.Mel.MFCC == true {
		for s := 0; s < se.Params.SegmentSteps; s++ {
			se.MFCCSegment.SetFloatRowCell(0, s, se.Energy.FloatVal1D(s))
		}
	}
	//calculate the MFCC deltas (change in MFCC coeficient over time - basically first derivative)
	//One source of the equation - https://privacycanada.net/mel-frequency-cepstral-coefficient/#Mel-filterbank-Computation

	//denominator = 2 * sum([i**2 for i in range(1, N+1)])
	// npn: For each frame, calculate delta features based on preceding and following npn frames
	npn := 2
	if se.Mel.MFCC && se.Mel.Deltas {
		for s := 0; s < int(se.Params.SegmentSteps); s++ {
			prv := 0.0
			nxt := 0.0
			for i := 0; i < se.Mel.NCoefs; i++ {
				nume := 0.0
				for n := 1; n <= npn; n++ {
					sprv := s - n
					snxt := s + n
					if sprv < 0 {
						sprv = 0
					}
					if snxt > se.Params.SegmentSteps-1 {
						snxt = se.Params.SegmentSteps - 1
					}
					prv += se.MFCCSegment.FloatValRowCell(i, sprv)
					nxt += se.MFCCSegment.FloatValRowCell(i, snxt)
					nume += float64(n) * (nxt - prv)

					denom := float64(2 * n * n)
					d := nume / denom
					// ToDo: better to have separate matrix and combine if using one layer as input
					se.MFCCDeltas.SetFloatRowCell(i, s, d)
				}
			}
		}

		// now the delta deltas
		for s := 0; s < int(se.Params.SegmentSteps); s++ {
			prv := 0.0
			nxt := 0.0
			for i := 0; i < se.Mel.NCoefs; i++ {
				nume := 0.0
				for n := 1; n <= npn; n++ {
					sprv := s - n
					snxt := s + n
					if sprv < 0 {
						sprv = 0
					}
					if snxt > se.Params.SegmentSteps-1 {
						snxt = se.Params.SegmentSteps - 1
					}
					prv += se.MFCCDeltas.FloatValRowCell(i, sprv)
					nxt += se.MFCCDeltas.FloatValRowCell(i, snxt)
					nume += float64(n) * (nxt - prv)

					denom := float64(2 * n * n)
					d := nume / denom
					// ToDo: better to have separate matrix and combine if using one layer as input
					se.MFCCDeltaDeltas.SetFloatRowCell(i, s, d)
				}
			}
		}
	}
}

// ProcessStep processes a step worth of sound input from current input_pos, and increment input_pos by input.step_samples
// Process the data by doing a fourier transform and computing the power spectrum, then apply mel filters to get the frequency
// bands that mimic the non-linear human perception of sound
func (se *SndEnv) ProcessStep(segment, step, add int) error {
	//fmt.Println("step: ", step)
	offset := se.Params.Steps[step] + MSecToSamples(float64(add), se.Sound.SampleRate())
	start := segment*int(se.Params.StrideSamples) + offset // segments start at zero
	err := se.SndToWindow(start)
	if err == nil {
		//gparams.Fft.Reset(wparams.WinSamples)
		se.DFT.Filter(step, &se.Window, se.Params.WinSamples, &se.Power, &se.LogPower, &se.PowerSegment, &se.LogPowerSegment)
		se.Mel.FilterDft(step, &se.Power, &se.MelFBankSegment, &se.MelFBank, &se.MelFilters)
		if se.Mel.MFCC {
			se.Mel.CepstrumDct(step, &se.MelFBank, &se.MFCCSegment, &se.MFCCDCT)
		}
	}
	return err
}

// SndToWindow gets sound from the signal (i.e. the slice of input values) at given position
func (se *SndEnv) SndToWindow(start int) error {
	if se.Signal.NumDims() == 1 {
		end := start + se.Params.WinSamples
		if end > len(se.Signal.Values) {
			return errors.New("SndToWindow: end beyond signal length!!")
		}
		var pad []float64
		if start < 0 && end <= 0 {
			pad = make([]float64, end-start)
			se.Window.Values = pad[0:]
		} else if start < 0 && end > 0 {
			pad = make([]float64, 0-start)
			se.Window.Values = pad[0:]
			se.Window.Values = append(se.Window.Values, se.Signal.Values[0:end]...)
		} else {
			se.Window.Values = se.Signal.Values[start:end]
		}
		//fmt.Println("start / end in samples:", start, end)
	} else {
		// ToDo: implement
		fmt.Printf("SndToWindow: else case not implemented - please report this issue")
	}
	return nil
}

// ApplyGabor convolves the gabor filters with the mel output
func (se *SndEnv) ApplyGabor() (tsr *etensor.Float32) {
	agabor.Convolve(&se.MelFBankSegment, se.GaborFilters, &se.GborOutput, se.ByTime)

	if se.NeighInhib.On {
		se.ApplyNeighInhib()
	} else {
		se.ExtGi.SetZeros()
	}

	if se.Kwta.On {
		se.ApplyKwta()
		tsr = &se.GborKwta
	} else {
		tsr = &se.GborOutput
	}
	return tsr
}

func (se *SndEnv) Name() string { return se.Nm }
func (se *SndEnv) Desc() string { return se.Dsc }

// Tail returns the number of samples that remain beyond the last full stride
func (se *SndEnv) Tail(signal []float64) int {
	temp := len(signal) - se.Params.SegmentSamples
	tail := temp % se.Params.StrideSamples
	return tail
}

// Pad pads the signal so that the length of signal divided by stride has no remainder
func (se *SndEnv) Pad(signal []float64, value float64) (padded []float64) {
	tail := se.Tail(signal)
	padLen := se.Params.SegmentSamples - se.Params.StepSamples - tail%se.Params.StepSamples
	pad := make([]float64, padLen)
	for i := range pad {
		pad[i] = value
	}
	padded = append(signal, pad...)
	return padded
}

// MSecToSamples converts milliseconds to samples, in terms of sample_rate
func MSecToSamples(ms float64, rate int) int {
	return int(math.Round(ms * 0.001 * float64(rate)))
}

// SamplesToMSec converts samples to milliseconds, in terms of sample_rate
func SamplesToMSec(samples int, rate int) float64 {
	return 1000.0 * float64(samples) / float64(rate)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	//fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	//fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("Alloc = %v B", m.Alloc)
	fmt.Printf("\tTotalAlloc = %v B", m.TotalAlloc)
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}
