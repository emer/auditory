// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package audio

import (
	"fmt"
	"github.com/chewxy/math32"
	"github.com/emer/etable/etable"
	"github.com/emer/etable/etensor"
	"github.com/emer/etable/minmax"
	"github.com/emer/leabra/leabra"
	"strconv"
)

// InitSound
func (ap *AuditoryProc) InitSound() bool {
	ap.InputPos = 0
	return true
}

// InitFromSound loads a sound and sets the Input channel vars and sample rate
func (in *Input) InitFromSound(snd *Sound, nChannels int, channel int) {
	if snd == nil {
		fmt.Printf("InitFromSound: sound nil")
		return
	}
	in.SampleRate = int(snd.SampleRate())
	in.ComputeSamples()
	if nChannels < 1 {
		in.Channels = int(snd.Channels())
	} else {
		in.Channels = int(math32.Min(float32(nChannels), float32(in.Channels)))
	}
	if in.Channels > 1 {
		in.Channel = channel
	} else {
		in.Channel = 0
	}
}

type AuditoryProc struct {
	Mel    Mel
	Data   *etable.Table `desc:"data table for saving filter results for viewing and applying to networks etc"`
	Input  Input         `desc:"specifications of the raw auditory input"`
	Gabor1 Gabor         `viewif:"MelFBank.On=true desc:"full set of frequency / time gabor filters -- first size"`

	FirstStep     bool            `inactive:"+" desc:" #NO_SAVE is this the first step of processing -- turns of prv smoothing of dft power"`
	InputPos      int             `inactive:"+" desc:" #NO_SAVE current position in the sound_full input -- in terms of sample number"`
	TrialStartPos int             `inactive:"+" desc:" #NO_SAVE starting position of the current trial -- in terms of sample number"`
	TrialEndPos   int             `inactive:"+" desc:" #NO_SAVE ending position of the current trial -- in terms of sample number"`
	SoundFull     etensor.Float32 `inactive:"+" desc:" #NO_SAVE the full sound input obtained from the sound input"`
	WindowIn      etensor.Float32 `inactive:"+" desc:" #NO_SAVE [input.win_samples] the raw sound input, one channel at a time"`
	UseInhib      bool            `viewif:"Gabor1.On=true" desc:"k-winner-take-all inhibitory dynamics for the time-gabor output"`

	Gabor1Raw etensor.Float32 `inactive:"+" desc:" #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] raw output of gabor1 -- full trial's worth of gabor steps"`
	Gabor1Out etensor.Float32 `inactive:"+" desc:" #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] post-kwta output of full trial's worth of gabor steps"`
}

// InitOutMatrices sets the shape of all output matrices
func (ap *AuditoryProc) InitOutputMatrices() bool {
	ap.WindowIn.SetShape([]int{ap.Input.WinSamples}, nil, nil)
	ap.Mel.DftOut.SetShape([]int{ap.Mel.DftSize}, nil, nil)
	ap.Mel.DftPowerOut.SetShape([]int{ap.Mel.DftUse}, nil, nil)
	ap.Mel.DftPowerTrialOut.SetShape([]int{ap.Mel.DftUse, ap.Input.TotalSteps, ap.Input.Channels}, nil, nil)

	if ap.Mel.Dft.LogPow {
		ap.Mel.DftLogPowerOut.SetShape([]int{ap.Mel.DftUse}, nil, nil)
		ap.Mel.DftLogPowerTrialOut.SetShape([]int{ap.Mel.DftUse, ap.Input.TotalSteps, ap.Input.Channels}, nil, nil)
	}

	if ap.Mel.MelFBank.On {
		ap.Mel.MelFBankOut.SetShape([]int{ap.Mel.MelFBank.NFilters}, nil, nil)
		ap.Mel.MelFBankTrialOut.SetShape([]int{ap.Mel.MelFBank.NFilters, ap.Input.TotalSteps, ap.Input.Channels}, nil, nil)
		if ap.Gabor1.On {
			ap.Gabor1Raw.SetShape([]int{ap.Input.Channels, ap.Gabor1.NFilters, 2, ap.Gabor1.Shape.Y, ap.Gabor1.Shape.X}, nil, nil)
			ap.Gabor1Out.SetShape([]int{ap.Input.Channels, ap.Gabor1.NFilters, 2, ap.Gabor1.Shape.Y, ap.Gabor1.Shape.X}, nil, nil)
		}
		if ap.Mel.Mfcc.On {
			ap.Mel.MfccDctOut.SetShape([]int{ap.Mel.MelFBank.NFilters}, nil, nil)
			ap.Mel.MfccDctTrialOut.SetShape([]int{ap.Mel.MelFBank.NFilters, ap.Input.TotalSteps, ap.Input.Channels}, nil, nil)
		}
	}
	return true
}

// LoadSound initializes the AuditoryProc with the sound loaded from file by "Sound"
func (ap *AuditoryProc) LoadSound(snd *Sound) bool {
	var needsInit = false
	if ap.Mel.NeedsInit(ap.Input.WinSamples) {
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
		ap.Initialize()
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
func (ap *AuditoryProc) StartNewSound() {
	ap.FirstStep = true
	ap.InputPos = 0
	ap.TrialStartPos = 0
	ap.TrialEndPos = int(ap.TrialStartPos) + ap.Input.TrialSamples
}

// Init initializes AuditoryProc fields
func (ap *AuditoryProc) Initialize() {
	ap.Input.Initialize()
	ap.Mel.Initialize(ap.Input.WinSamples, ap.Input.SampleRate)
	ap.Gabor1.Initialize(ap.Input.TrialSteps, ap.Mel.MelFBank.NFilters)
	ap.Gabor1.On = true
	ap.Mel.Mfcc.Initialize()

	ap.InitOutputMatrices()
	ap.Data = &etable.Table{}
	ap.InitDataTable()
	ap.InitSound()
	ap.UseInhib = false
}

// InitDataTable readies ap.Data, an etable.etable
func (ap *AuditoryProc) InitDataTable() bool {
	if ap.Data == nil {
		fmt.Printf("InitDataTable: ap.Data is nil")
		return false
	}
	if ap.Input.Channels > 1 {
		for ch := 0; ch < int(ap.Input.Channels); ch++ {
			ap.MelOutputToTable(ap.Data, ch, true)
		}
	} else {
		ap.MelOutputToTable(ap.Data, ap.Input.Channel, true)

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
	if ap.Mel.NeedsInit(ap.Input.WinSamples) {
		ap.Initialize()
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
		sz := zeroN * 4 // 4 bytes - size of float32
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
		ap.Mel.CopyStepFromStep(s, int(srcStStep)+s, ch)
	}
	return true
}

// StepForward
func (ap *AuditoryProc) StepForward(ch int) bool {
	totalM1 := ap.Input.TotalSteps - 1
	for s := 0; s < int(totalM1); s++ {
		ap.Mel.CopyStepFromStep(s, s+1, ch)
	}
	return true
}

// ProcessStep process a step worth of sound input from current input_pos, and increment input_pos by input.step_samples
func (ap *AuditoryProc) ProcessStep(ch int, step int) bool {
	ap.SoundToWindow(ap.InputPos, ch)
	ap.Mel.FilterWindow(int(ch), int(step), ap.WindowIn, ap.FirstStep)
	ap.InputPos = ap.InputPos + ap.Input.StepSamples
	ap.FirstStep = false
	return true
}

// FilterTrial processes filters that operate over an entire trial at a time
func (ap *AuditoryProc) FilterTrial(ch int) {
	if ap.Gabor1.On {
		ap.GaborFilter(ch, ap.Gabor1, ap.Gabor1Raw, ap.Gabor1Out)
	}
}

// GaborFilter processes input using filters that operate over an entire trial at a time
func (ap *AuditoryProc) GaborFilter(ch int, spec Gabor, outRaw etensor.Float32, out etensor.Float32) {
	//func (ap *AuditoryProc) GaborFilter(ch int, spec Gabor, filters etensor.Float32, outRaw etensor.Float32, out etensor.Float32) {
	tHalfSz := spec.SizeTime / 2
	tOff := tHalfSz - ap.Input.BorderSteps
	tMin := tOff
	if tMin < 0 {
		tMin = 0
	}
	tMax := ap.Input.TrialSteps - tMin

	fMin := int(0)
	fMax := ap.Mel.MelFBank.NFilters - spec.SizeFreq

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
						fVal := spec.Filters.Value([]int{ft, ff, fi})
						iVal := ap.Mel.MelFBankTrialOut.Value([]int{flt + ff, inSt + ft, ch})
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

// OutputToTable
func (ap *AuditoryProc) OutputToTable(ch int) bool {
	if ap.Data == nil {
		return false
	}
	if ap.Mel.MelFBank.On {
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
		err = dt.AddCol(etensor.NewFloat32([]int{rows, int(ap.Input.TotalSteps), int(ap.Mel.DftUse)}, nil, nil), cn)
		if err != nil {
			fmt.Printf("MelOutputToTable: column %v not found or failed to be created", cn)
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
			for i := 0; i < int(ap.Mel.DftUse); i++ {
				if ap.Mel.Dft.LogPow {
					val := ap.Mel.DftLogPowerTrialOut.FloatVal([]int{i, s, ch})
					dout.SetFloat([]int{s, i}, val)
				} else {
					val := ap.Mel.DftPowerTrialOut.FloatVal([]int{i, s, ch})
					dout.SetFloat([]int{s, i}, val)
				}
			}
		}
	}

	if ap.Mel.MelFBank.On {
		cn := "AudProc" + "_mel_fbank" + colSfx // column name
		col := dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, int(ap.Input.TotalSteps), int(ap.Mel.MelFBank.NFilters)}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column %v not found or failed to be created", cn)
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
				for i := 0; i < int(ap.Mel.MelFBank.NFilters); i++ {
					val := ap.Mel.MelFBankTrialOut.FloatVal([]int{i, s, ch})
					dout.SetFloat([]int{s, i}, val)
				}
			}
		}
	}

	if ap.Gabor1.On {
		cn := "AudProc" + "_mel_gabor1_raw" + colSfx // column name
		col := dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor1.Shape.Y, ap.Gabor1.Shape.X, 2, ap.Gabor1.NFilters}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column %v not found or failed to be created", cn)
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
			for s := 0; s < ap.Gabor1.Shape.X; s++ {
				for i := 0; i < ap.Gabor1.Shape.Y; i++ {
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
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Gabor1.Shape.Y, ap.Gabor1.Shape.X, 2, ap.Gabor1.NFilters}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column %v not found or failed to be created", cn)
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
			for s := 0; s < ap.Gabor1.Shape.X; s++ {
				for i := 0; i < ap.Gabor1.Shape.Y; i++ {
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

	if ap.Mel.Mfcc.On {
		cn = "AudProc" + "_mel_mfcc" + colSfx // column name
		col = dt.ColByName(cn)
		if col == nil {
			err = dt.AddCol(etensor.NewFloat32([]int{rows, ap.Input.TotalSteps, ap.Mel.Mfcc.NCoeff}, nil, nil), cn)
			if err != nil {
				fmt.Printf("MelOutputToTable: column %v not found or failed to be created", cn)
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
				for i := 0; i < ap.Mel.Mfcc.NCoeff; i++ {
					val := ap.Mel.MfccDctTrialOut.FloatVal([]int{i, s, ch})
					dout.SetFloat([]int{s, i}, val)
				}
			}
		}
	}
	return true
}
