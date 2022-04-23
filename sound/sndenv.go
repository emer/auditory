// Copyright (c) 2021, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sound

import (
	"errors"
	"fmt"
	"log"
	"math"

	"github.com/emer/auditory/agabor"
	"github.com/emer/auditory/dft"
	"github.com/emer/auditory/mel"
	"github.com/emer/etable/etable"
	"github.com/emer/etable/etensor"
	"github.com/emer/leabra/fffb"
	"github.com/emer/vision/kwta"
	"gonum.org/v1/gonum/dsp/fourier"
)

// Params defines the sound input parameters for auditory processing
type Params struct {
	WinMs       float32 `def:"25" desc:"input window -- number of milliseconds worth of sound to filter at a time"`
	StepMs      float32 `def:"5,10,12.5" desc:"input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample"`
	SegmentMs   float32 `def:"100" desc:"length of full segment's worth of input -- total number of milliseconds to accumulate into a complete segment -- must be a multiple of StepMs -- input will be SegmentMs / StepMs = SegmentSteps wide in the X axis, and number of filters in the Y axis"`
	StrideMs    float32 `def:"100" desc:"how far to move on each trial"`
	BorderSteps int     `def:"6" view:"+" desc:"overlap with previous and next segment"`
	Channel     int     `viewif:"Channels=1" desc:"specific channel to process, if input has multiple channels, and we only process one of them (-1 = process all)"`
	PadEnd      bool    `desc:"should the end of the sequence be padded. Can be used to ensure pull segment at end"`
	PadValue    float32 `desc:"value to use of signal when padding"`

	// these are calculated
	WinSamples     int   `inactive:"+" desc:"number of samples to process each step"`
	StepSamples    int   `inactive:"+" desc:"number of samples to step input by"`
	SegmentSamples int   `inactive:"+" desc:"number of samples in a segment"`
	StrideSamples  int   `inactive:"+" desc:"number of samples converted from StrideMS"`
	SegmentSteps   int   `inactive:"+" desc:"includes border steps on both sides"`
	Steps          []int `inactive:"+" desc:"pre-calculated start position for each step"`
}

// ParamDefaults initializes the Input
func (se *SndEnv) ParamDefaults() {
	se.Params.WinMs = 25.0
	se.Params.StepMs = 10.0
	se.Params.SegmentMs = 100.0
	se.Params.Channel = 0
	se.Params.PadValue = 0.0
	se.Params.StrideMs = 100.0
	se.Params.BorderSteps = 2
}

type SndEnv struct {
	// the environment has the training/test data and the procedures for creating/choosing the input to the model
	// "Segment" in var name indicates that the data or value only applies to a segment of samples rather than the entire signal
	Nm      string `desc:"name of this environment"`
	Dsc     string `desc:"description of this environment"`
	Sound   Wave   `desc:"specifications of the raw sensory input"`
	Params  Params
	Signal  etensor.Float32 `view:"no-inline" desc:" the full sound input"`
	SegCnt  int             `desc:"the number of segments in this sound file (based on current segment size)"`
	Window  etensor.Float32 `inactive:"+" desc:" [Input.WinSamples] the raw sound input, one channel at a time"`
	Segment int             `inactive:"no-inline" desc:" the current chunk of samples (a full segment's' worth) - zero is first chunk"`

	Dft             dft.Params
	Power           etensor.Float32 `view:"-" desc:" power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)"`
	LogPower        etensor.Float32 `view:"-" desc:" log power of the dft, up to the nyquist liit frequency (1/2 input.WinSamples)"`
	PowerSegment    etensor.Float32 `view:"no-inline" desc:" full segment's worth of power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	LogPowerSegment etensor.Float32 `view:"no-inline" desc:" full segment's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	Mel             mel.Params      `view:"no-inline"`
	MelFBank        etensor.Float32 `view:"no-inline" desc:" mel scale transformation of dft_power, resulting in the mel filterbank output -- the natural log of this is typically applied"`
	MelFBankSegment etensor.Float32 `view:"no-inline" desc:" full segment's worth of mel feature-bank output"`
	MelFilters      etensor.Float32 `view:"no-inline" desc:" the actual filters"`
	MfccDctSegment  etensor.Float32 `view:"no-inline" desc:" full segment's worth of discrete cosine transform of log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
	MfccDct         etensor.Float32 `view:"no-inline" desc:" discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`

	GaborSpecs    []agabor.Filter   `view:"no-inline" desc:" a set of gabor filter specifications, one spec per filter'"`
	GaborFilters  agabor.FilterSet  `desc:"the actual gabor filters, the first spec determines the size of all filters in the set"`
	GaborTab      etable.Table      `view:"no-inline" desc:"gabor filter table (view only)"`
	GborOutPoolsX int               `view:"+" desc:" the number of neuron pools along the time dimension in the input layer"`
	GborOutPoolsY int               `view:"+" desc:" the number of neuron pools along the frequency dimension in the input layer"`
	GborOutUnitsX int               `view:"+" desc:" the number of neurons in a pool (typically the number of gabor filters) along the time dimension in the input layer"`
	GborOutUnitsY int               `view:"+" desc:" the number of neurons in a pool along the frequency dimension in the input layer"`
	GborOutput    etensor.Float32   `view:"no-inline" desc:" raw output of Gabor -- full segment's worth of gabor steps"`
	GborKwta      etensor.Float32   `view:"no-inline" desc:" post-kwta output of full segment's worth of gabor steps"`
	Inhibs        fffb.Inhibs       `view:"no-inline" desc:"inhibition values for A1 KWTA"`
	ExtGi         etensor.Float32   `view:"no-inline" desc:"A1 simple extra Gi from neighbor inhibition tensor"`
	NeighInhib    kwta.NeighInhib   `desc:"neighborhood inhibition for V1s -- each unit gets inhibition from same feature in nearest orthogonal neighbors -- reduces redundancy of feature code"`
	Kwta          kwta.KWTA         `desc:"kwta parameters, using FFFB form"`
	KwtaPool      bool              `desc:"if Kwta.On == true, call KwtaPool (true) or KwtaLayer (false)"`
	FftCoefs      []complex128      `view:"-" desc:" discrete fourier transform (fft) output complex representation"`
	Fft           *fourier.CmplxFFT `view:"-" desc:" struct for fast fourier transform"`
	ByTime        bool              `desc:"display the gabor filtering result by time and then by filter, default is to order by filter and then time"`
}

// Defaults
func (se *SndEnv) Defaults() {
	se.ParamDefaults()
	se.Mel.Defaults() // calls melfbank defaults
	se.Kwta.Defaults()
	se.KwtaPool = true
}

// Init sets various sound processing params based on default params and user overrides
// add is the amount of random silence that should precede the start of the sequence
// start is the amount of silence preexisting at start
// All values milliseconds
func (se *SndEnv) Init(add, existing float64) (err error, offset int) {
	sr := se.Sound.SampleRate()
	if sr <= 0 {
		fmt.Println("sample rate <= 0")
		err = errors.New("sample rate <= 0")
		return err, 0
	}
	se.Params.WinSamples = MSecToSamples(se.Params.WinMs, sr)
	se.Params.StepSamples = MSecToSamples(se.Params.StepMs, sr)
	se.Params.SegmentSamples = MSecToSamples(se.Params.SegmentMs, sr)
	steps := int(math.Round(float64(se.Params.SegmentMs / se.Params.StepMs)))
	se.Params.SegmentSteps = steps + 2*se.Params.BorderSteps
	se.Params.StrideSamples = MSecToSamples(se.Params.StrideMs, sr)

	offset = 0.0
	if add >= 0 {
		if add < existing {
			offset = int(existing - add)
			se.Signal.Values = se.Signal.Values[int(offset):len(se.Signal.Values)]
		} else if add > existing {
			offset = int(add - existing)
			n := int((float64(offset) * float64(se.Params.StrideSamples)) / 100.0)
			silence := make([]float32, n)
			se.Signal.Values = append(silence, se.Signal.Values...)
		}
	}

	specs := agabor.Active(se.GaborSpecs)
	nfilters := len(specs)
	se.GaborFilters.Filters.SetShape([]int{nfilters, se.GaborFilters.SizeY, se.GaborFilters.SizeX}, nil, nil)
	se.NeighInhib.Defaults() // NeighInhib code not working yet - need to pass 4d tensor not 5d
	agabor.ToTensor(specs, &se.GaborFilters)
	se.GaborFilters.ToTable(se.GaborFilters, &se.GaborTab) // note: view only, testing
	if se.GborOutPoolsX == 0 && se.GborOutPoolsY == 0 {    // 2D
		se.GborOutput.SetShape([]int{se.Sound.Channels(), se.GborOutUnitsY, se.GborOutUnitsX}, nil, []string{"chan", "freq", "time"})
		se.ExtGi.SetShape([]int{se.GborOutUnitsY, nfilters}, nil, nil) // passed in for each channel
	} else if se.GborOutPoolsX > 0 && se.GborOutPoolsY > 0 { // 4D
		se.GborOutput.SetShape([]int{se.Sound.Channels(), se.GborOutPoolsY, se.GborOutPoolsX, se.GborOutUnitsY, se.GborOutUnitsX}, nil, []string{"chan", "freq", "time"})
		se.ExtGi.SetShape([]int{se.GborOutPoolsY, se.GborOutPoolsX, 2, nfilters}, nil, nil) // passed in for each channel
	} else {
		log.Println("GborOutPoolsX & GborOutPoolsY must both be == 0 or > 0 (i.e. 2D or 4D)")
		return err, 0
	}
	se.GborOutput.SetMetaData("odd-row", "true")
	se.GborOutput.SetMetaData("grid-fill", ".9")
	se.GborKwta.CopyShapeFrom(&se.GborOutput)
	se.GborKwta.CopyMetaData(&se.GborOutput)

	winSamplesHalf := se.Params.WinSamples/2 + 1
	se.Dft.Initialize(se.Params.WinSamples)
	se.Mel.InitFilters(se.Params.WinSamples, se.Sound.SampleRate(), &se.MelFilters) // call after non-default values are set!
	se.Window.SetShape([]int{se.Params.WinSamples}, nil, nil)
	se.Power.SetShape([]int{winSamplesHalf}, nil, nil)
	se.LogPower.CopyShapeFrom(&se.Power)
	se.PowerSegment.SetShape([]int{se.Params.SegmentSteps, winSamplesHalf, se.Sound.Channels()}, nil, nil)
	if se.Dft.CompLogPow {
		se.LogPowerSegment.CopyShapeFrom(&se.PowerSegment)
	}

	se.FftCoefs = make([]complex128, se.Params.WinSamples)
	se.Fft = fourier.NewCmplxFFT(len(se.FftCoefs))

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
	se.MelFBankSegment.SetShape([]int{se.Params.SegmentSteps, se.Mel.FBank.NFilters, se.Sound.Channels()}, nil, nil)
	if se.Mel.MFCC {
		se.MfccDctSegment.CopyShapeFrom(&se.MelFBankSegment)
		se.MfccDct.SetShape([]int{se.Mel.FBank.NFilters}, nil, nil)
	}

	siglen := len(se.Signal.Values) - se.Params.SegmentSamples*se.Sound.Channels()
	siglen = siglen / se.Sound.Channels()
	se.SegCnt = siglen/se.Params.StrideSamples + 1 // add back the first segment subtracted at from siglen calculation
	se.Segment = -1
	return nil, offset
}

// ToTensor
func (se *SndEnv) ToTensor() bool {
	if se.Sound.Channels() > 1 {
		se.Sound.SoundToTensor(&se.Signal, -1)
	} else {
		se.Sound.SoundToTensor(&se.Signal, se.Params.Channel)
	}
	return true
}

// ApplyKwta runs the kwta algorithm on the raw activations
func (se *SndEnv) ApplyKwta(ch int) {
	se.GborKwta.CopyFrom(&se.GborOutput)
	if se.Kwta.On {
		rawSS := se.GborOutput.SubSpace([]int{ch}).(*etensor.Float32)
		kwtaSS := se.GborKwta.SubSpace([]int{ch}).(*etensor.Float32)
		if se.KwtaPool == true {
			se.Kwta.KWTAPool(rawSS, kwtaSS, &se.Inhibs, &se.ExtGi)
		} else {
			se.Kwta.KWTALayer(rawSS, kwtaSS, &se.ExtGi)
		}
	}
}

// ProcessSegment processes the entire segment's input by processing a small overlapping set of samples on each pass
func (se *SndEnv) ProcessSegment() (moreSegments bool) {
	moreSegments = true
	se.Power.SetZeros()
	se.LogPower.SetZeros()
	se.PowerSegment.SetZeros()
	se.LogPowerSegment.SetZeros()
	se.MelFBankSegment.SetZeros()
	se.MfccDctSegment.SetZeros()
	//moreSamples := true
	se.Segment++
	//fmt.Printf("Segment: %d\n", se.Segment)
	for ch := int(0); ch < se.Sound.Channels(); ch++ {
		for s := 0; s < int(se.Params.SegmentSteps); s++ {
			err := se.ProcessStep(ch, s)
			if err != nil {
				break
			}
		}
	}
	remaining := len(se.Signal.Values) - (se.Segment+1)*se.Params.StrideSamples
	//fmt.Printf("total length = %v, remaining = %v\n", len(se.Signal.Values), remaining)
	if remaining < se.Params.SegmentSamples {
		moreSegments = false
		//fmt.Printf("Last Segment for %v: %d\n", se.SndFileCur, se.Segment)
	}
	se.ApplyGabor()
	return moreSegments
}

// ProcessStep processes a step worth of sound input from current input_pos, and increment input_pos by input.step_samples
// Process the data by doing a fourier transform and computing the power spectrum, then apply mel filters to get the frequency
// bands that mimic the non-linear human perception of sound
func (se *SndEnv) ProcessStep(ch int, step int) error {
	offset := se.Params.Steps[step]
	err := se.SndToWindow(offset, ch)
	if err == nil {
		se.Fft.Reset(se.Params.WinSamples)
		se.Dft.Filter(int(ch), int(step), &se.Window, se.Params.WinSamples, se.FftCoefs, se.Fft, &se.Power, &se.LogPower, &se.PowerSegment, &se.LogPowerSegment)
		se.Mel.FilterDft(int(ch), int(step), &se.Power, &se.MelFBankSegment, &se.MelFBank, &se.MelFilters)
		if se.Mel.MFCC {
			se.Mel.CepstrumDct(ch, step, &se.MelFBank, &se.MfccDctSegment, &se.MfccDct)
		}
	}
	return err
}

// SndToWindow gets sound from the signal (i.e. the slice of input values) at given position and channel, into Window
func (se *SndEnv) SndToWindow(stepOffset int, ch int) error {
	if se.Signal.NumDims() == 1 {
		start := se.Segment*int(se.Params.StrideSamples) + stepOffset // segments start at zero
		end := start + se.Params.WinSamples
		if end > len(se.Signal.Values) {
			return errors.New("SndToWindow: end beyond signal length!!")
		}
		var pad []float32
		if start < 0 && end <= 0 {
			pad = make([]float32, end-start)
			se.Window.Values = pad[0:]
		} else if start < 0 && end > 0 {
			pad = make([]float32, 0-start)
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
	for ch := int(0); ch < se.Sound.Channels(); ch++ {
		agabor.Convolve(ch, &se.MelFBankSegment, se.GaborFilters, &se.GborOutput, se.ByTime)
		//if se.NeighInhib.On {
		//	se.NeighInhib.Inhib4(&se.GborOutput, &se.ExtGi)
		//} else {
		//	se.ExtGi.SetZeros()
		//}

		if se.Kwta.On {
			se.ApplyKwta(ch)
			tsr = &se.GborKwta
		} else {
			tsr = &se.GborOutput
		}
	}
	return tsr
}

func (se *SndEnv) Name() string { return se.Nm }
func (se *SndEnv) Desc() string { return se.Dsc }

// Tail returns the number of samples that remain beyond the last full stride
func (se *SndEnv) Tail(signal []float32) int {
	temp := len(signal) - se.Params.SegmentSamples
	tail := temp % se.Params.StrideSamples
	return tail
}

// Pad pads the signal so that the length of signal divided by stride has no remainder
func (se *SndEnv) Pad(signal []float32) (padded []float32) {
	tail := se.Tail(signal)
	padLen := se.Params.SegmentSamples - se.Params.StepSamples - tail%se.Params.StepSamples
	pad := make([]float32, padLen)
	for i := range pad {
		pad[i] = se.Params.PadValue
	}
	padded = append(signal, pad...)
	return padded
}

// MSecToSamples converts milliseconds to samples, in terms of sample_rate
func MSecToSamples(ms float32, rate int) int {
	return int(math.Round(float64(ms) * 0.001 * float64(rate)))
}

// SamplesToMSec converts samples to milliseconds, in terms of sample_rate
func SamplesToMSec(samples int, rate int) float32 {
	return 1000.0 * float32(samples) / float32(rate)
}
