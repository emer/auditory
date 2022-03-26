package main

import (
	"errors"
	"fmt"
	"github.com/emer/auditory/agabor"
	"github.com/emer/auditory/dft"
	"github.com/emer/auditory/mel"
	"github.com/emer/auditory/sound"
	"github.com/emer/auditory/speech"
	"github.com/emer/auditory/speech/grafestes"
	"github.com/emer/auditory/speech/synthcvs"
	"github.com/emer/auditory/speech/timit"
	"github.com/emer/emergent/egui"
	"github.com/emer/emergent/evec"
	"github.com/emer/etable/etable"
	"github.com/emer/etable/etensor"
	"github.com/emer/etable/etview"
	"github.com/emer/leabra/fffb"
	"github.com/emer/vision/kwta"
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
	"gonum.org/v1/gonum/dsp/fourier"
	"log"
	"math"
	"os"
	"path"
	"strings"
)

func main() {
	TheApp.New()
	TheApp.Config()
	if len(os.Args) > 1 {
		TheApp.CmdArgs() // simple assumption is that any args = no gui -- could add explicit arg if you want
	} else {
		gimain.Main(func() { // this starts gui -- requires valid OpenGL display connection (e.g., X11)
			guirun()
		})
	}
}

func guirun() {
	TheApp.Init()
	win := TheApp.ConfigGui()
	win.StartEventLoop()
}

// this registers this Sim Type and gives it properties that e.g.,
// prompt for filename for save methods.
var KiT_Sim = kit.Types.AddType(&App{}, AppProps)

// TheApp is the overall state for this simulation
var TheApp App

func (ss *App) New() {
}

////////////////////////////////////////////////////////////////////////////////////////////
// Environment - params and config for the train/test environment

// User holds data (or points to other fields) the user will likely want to access quickly and edit frequently
type User struct {
	SegmentStart    int `view:"inline" desc:"the start of the audio segment, in milliseconds, that we will be processing"`
	SegmentEnd      int `view:"inline" desc:"the end of the audio segment, in milliseconds, that we will be processing"`
	SegmentDuration int `view:"inline" desc:"segment duration, in milliseconds, to step if stepping through sound sequence"`
}

// Params defines the sound input parameters for auditory processing
type Params struct {
	WinMs       float32 `def:"25" desc:"input window -- number of milliseconds worth of sound to filter at a time"`
	StepMs      float32 `def:"5,10,12.5" desc:"input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample"`
	SegmentMs   float32 `def:"100" desc:"length of full segment's worth of input -- total number of milliseconds to accumulate into a complete segment -- must be a multiple of StepMs -- input will be SegmentMs / StepMs = SegmentSteps wide in the X axis, and number of filters in the Y axis"`
	StrideMs    float32 `def:"100" desc:"how far to move on each trial"`
	BorderSteps int     `def:"0" desc:"overlap with previous segment"`
	Channel     int     `view:"-" desc:"specific channel to process, if input has multiple channels, and we only process one of them (-1 = process all)"`
	PadValue    float32 `view:"-"`

	// these are calculated
	WinSamples        int   `view:"-" desc:"number of samples to process each step"`
	StepSamples       int   `view:"-" desc:"number of samples to step input by"`
	SegmentSamples    int   `view:"-" desc:"number of samples in a segment"`
	StrideSamples     int   `view:"-" desc:"number of samples converted from StrideMS"`
	SegmentSteps      int   `view:"-" desc:"number of steps in a segment"`
	SegmentStepsTotal int   `view:"-" desc:"SegmentSteps plus steps overlapping next segment or for padding if no next segment"`
	Steps             []int `view:"-" desc:"pre-calculated start position for each step"`
}

type App struct {
	Nm          string `view:"-" desc:"name of this environment"`
	Dsc         string `view:"-" desc:"description of this environment"`
	Corpus      string `desc:"the set of sound files"`
	InitDir     string `desc:"initial directory for file open dialog"`
	SndFile     string `view:"-" desc:"full path of open sound file"`
	SndFileName string `desc:"name of the open sound file"`
	SndFilePath string `desc:"path to the open sound file"`

	UserVals User            `view:"inline" desc:"sound processing values and matrices for the short duration pathway"`
	Params   Params          `view:"inline" desc:"fundamental processing parameters"`
	Sound    sound.Wave      `view:"no-inline"`
	Signal   etensor.Float32 `view:"-" desc:" the full sound input obtained from the sound input - plus any added padding"`
	Samples  etensor.Float32 `view:"-" desc:" a window's worth of raw sound input, one channel at a time"`
	SegCnt   int             `desc:"the number of segments in this sound file (based on current segment size)"`
	Window   etensor.Float32 `inactive:"+" desc:" [Input.WinSamples] the raw sound input, one channel at a time"`
	Segment  int             `inactive:"+" desc:"current segment of full sound (zero based)"`

	Dft             dft.Params      `view:"-" desc:" "`
	Power           etensor.Float32 `view:"-" desc:" power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)"`
	LogPower        etensor.Float32 `view:"-" desc:" log power of the dft, up to the nyquist liit frequency (1/2 input.WinSamples)"`
	PowerSegment    etensor.Float32 `view:"no-inline" desc:" full segment's worth of power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)"`
	LogPowerSegment etensor.Float32 `view:"no-inline" desc:" full segment's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.WinSamples)"`
	Mel             mel.Params      `view:"inline"`
	MelFBank        etensor.Float32 `view:"no-inline" desc:" mel scale transformation of dft_power, using triangular filters, resulting in the mel filterbank output -- the natural log of this is typically applied"`
	MelFBankSegment etensor.Float32 `view:"no-inline" desc:" full segment's worth of mel feature-bank output"`
	MelFilters      etensor.Float32 `view:"no-inline" desc:" the actual filters"`
	MfccDct         etensor.Float32 `view:"-" desc:" discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
	MfccDctSegment  etensor.Float32 `view:"-" desc:" full segment's worth of discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`

	GaborSpecs []agabor.Filter   `view:"inline" desc:"array of params describing each gabor filter"`
	GaborSet   agabor.FilterSet  `view:"inline" desc:"a set of gabor filters with same x and y dimensions"`
	GaborTab   etable.Table      `view:"no-inline" desc:"gabor filter table (view only)"`
	GborOutput etensor.Float32   `view:"no-inline" desc:" raw output of Gabor -- full segment's worth of gabor steps"`
	GborKwta   etensor.Float32   `view:"no-inline" desc:" post-kwta output of full segment's worth of gabor steps"`
	Inhibs     fffb.Inhibs       `view:"no-inline" desc:"inhibition values for A1 KWTA"`
	ExtGi      etensor.Float32   `view:"no-inline" desc:"A1 simple extra Gi from neighbor inhibition tensor"`
	NeighInhib kwta.NeighInhib   `desc:"neighborhood inhibition for V1s -- each unit gets inhibition from same feature in nearest orthogonal neighbors -- reduces redundancy of feature code"`
	Kwta       kwta.KWTA         `desc:"kwta parameters, using FFFB form"`
	KwtaPool   bool              `desc:"if Kwta.On == true, call KwtaPool (true) or KwtaLayer (false)"`
	FftCoefs   []complex128      `view:"-" desc:" discrete fourier transform (fft) output complex representation"`
	Fft        *fourier.CmplxFFT `view:"-" desc:" struct for fast fourier transform"`

	// internal state - view:"-"
	FirstStep bool `view:"-" desc:" if first frame to process -- turns off prv smoothing of dft power"`

	//Output etensor.Float32 `view:"no-inline" desc:" raw output of gabor convolution"`

	SpeechSeq speech.SpeechSequence
	CurUnits  []string `desc:"names of units in the current audio segment"`
	// ToDo: maybe this should be calculated from based on filters?? Avoids mismatch!!
	OutputDims evec.Vec2i `desc:"X and Y dimensions of the output matrix (i.e. result after convolution"`

	// internal state - view:"-"
	ToolBar      *gi.ToolBar `view:"-" desc:" the master toolbar"`
	MoreSegments bool        `view:"-" desc:" are there more samples to process"`
	SndIdx       int         `view:"-" desc:"the index into the soundlist of the sound from last trial"`

	GUI          egui.GUI           `view:"-" desc:"manages all the gui elements"`
	PowerGrid    *etview.TensorGrid `view:"+" desc:"power grid view for the current segment"`
	MelFBankGrid *etview.TensorGrid `view:"-" desc:"melfbank grid view for the current segment"`
	GaborOutGrid *etview.TensorGrid `view:"+" desc:"result of convolving gabor filters with audio segment"`
}

// Init
func (ap *App) Init() {
	ap.ParamDefaults()
	ap.Mel.Defaults()
	ap.InitGabors()
	active := agabor.Active(ap.GaborSpecs)
	ap.GaborSet.Filters.SetShape([]int{len(active), ap.GaborSet.SizeY, ap.GaborSet.SizeX}, nil, nil)
	ap.GenGabors()
}

// ParamDefaults initializes the Input
func (ap *App) ParamDefaults() {
	ap.Params.WinMs = 25.0
	ap.Params.StepMs = 10.0
	ap.Params.SegmentMs = 100.0
	ap.Params.Channel = 0
	ap.Params.PadValue = 0.0
	ap.Params.StrideMs = 100.0
	ap.Params.BorderSteps = 0
}

// Config configures all the elements using the standard functions
func (ap *App) Config() {
	ap.Corpus = "TIMIT"
	if ap.Corpus == "TIMIT" {
		ap.InitDir = "/Users/rohrlich/ccn_images/sound/timit/TIMIT/TRAIN/"
	} else {
		ap.InitDir = "~"
	}
}

func (ap *App) InitGabors() {
	ap.GaborSet.SizeX = 9
	ap.GaborSet.SizeY = 9
	ap.GaborSet.Gain = 1.5
	ap.GaborSet.StrideX = 4
	ap.GaborSet.StrideY = 3
	ap.GaborSet.Distribute = false

	orient := []float32{0, 45, 90, 135}
	wavelen := []float32{1.0, 2.0}
	phase := []float32{0}
	sigma := []float32{0.5}

	ap.GaborSpecs = nil // in case there are some specs already

	for _, or := range orient {
		for _, wv := range wavelen {
			for _, ph := range phase {
				for _, wl := range sigma {
					spec := agabor.Filter{WaveLen: wv, Orientation: or, SigmaWidth: wl, SigmaLength: wl, PhaseOffset: ph, CircleEdge: true}
					ap.GaborSpecs = append(ap.GaborSpecs, spec)
				}
			}
		}
	}
}

func (ap *App) InitProcess() (err error, segments int) {
	if ap.Sound.Buf == nil {
		return errors.New("Load a sound and try again"), 0
	}
	sr := ap.Sound.SampleRate()
	if sr <= 0 {
		fmt.Println("sample rate <= 0")
		err = errors.New("sample rate <= 0")
		return err, 0
	}
	ap.Params.WinSamples = sound.MSecToSamples(ap.Params.WinMs, sr)
	ap.Params.StepSamples = sound.MSecToSamples(ap.Params.StepMs, sr)
	ap.Params.SegmentSamples = sound.MSecToSamples(ap.Params.SegmentMs, sr)
	ap.Params.SegmentSteps = int(math.Round(float64(ap.Params.SegmentMs / ap.Params.StepMs)))
	ap.Params.SegmentStepsTotal = ap.Params.SegmentSteps + 2*ap.Params.BorderSteps
	ap.Params.StrideSamples = sound.MSecToSamples(ap.Params.StrideMs, sr)

	winSamplesHalf := ap.Params.WinSamples/2 + 1
	ap.Dft.Initialize(ap.Params.WinSamples)
	ap.Mel.InitFilters(ap.Params.WinSamples, ap.Sound.SampleRate(), &ap.MelFilters) // call after non-default values are set!
	ap.Window.SetShape([]int{ap.Params.WinSamples}, nil, nil)
	ap.Power.SetShape([]int{winSamplesHalf}, nil, nil)
	ap.LogPower.CopyShapeFrom(&ap.Power)
	ap.PowerSegment.SetShape([]int{ap.Params.SegmentStepsTotal, winSamplesHalf, ap.Sound.Channels()}, nil, nil)
	if ap.Dft.CompLogPow {
		ap.LogPowerSegment.CopyShapeFrom(&ap.PowerSegment)
	}

	ap.FftCoefs = make([]complex128, ap.Params.WinSamples)
	ap.Fft = fourier.NewCmplxFFT(len(ap.FftCoefs))

	// 2 reasons for this code
	// 1 - the amount of signal handed to the fft has a "border" (some extra signal) to avoid edge effects.
	// On the first step there is no signal to act as the "border" so we pad the data handed on the front.
	// 2 - signals needs to be aligned when the number when multiple signals are input (e.g. 100 and 300 ms)
	// so that the leading edge (right edge) is the same time point.
	// This code does this by generating negative offsets for the start of the processing.
	// Also see SndToWindow for the use of the step values
	stepsBack := ap.Params.BorderSteps
	ap.Params.Steps = make([]int, ap.Params.SegmentStepsTotal)
	for i := 0; i < ap.Params.SegmentStepsTotal; i++ {
		ap.Params.Steps[i] = ap.Params.StepSamples * (i - stepsBack)
	}

	ap.MelFBank.SetShape([]int{ap.Mel.FBank.NFilters}, nil, nil)
	ap.MelFBankSegment.SetShape([]int{ap.Params.SegmentStepsTotal, ap.Mel.FBank.NFilters, ap.Sound.Channels()}, nil, nil)
	if ap.Mel.MFCC {
		ap.MfccDctSegment.CopyShapeFrom(&ap.MelFBankSegment)
		ap.MfccDct.SetShape([]int{ap.Mel.FBank.NFilters}, nil, nil)
	}

	siglen := len(ap.Signal.Values) - ap.Params.SegmentSamples*ap.Sound.Channels()
	siglen = siglen / ap.Sound.Channels()
	ap.SegCnt = siglen/ap.Params.StrideSamples + 1 // add back the first segment subtracted at from siglen calculation
	ap.Segment = -1
	return nil, ap.SegCnt
}

// LoadSnd
func (ap *App) LoadSnd(path string) {
	ap.SndFile = path
	err := ap.Sound.Load(path)
	if err != nil {
		log.Printf("LoadSnd: error loading sound -- %v\n, err", ap.SpeechSeq.File)
		return
	}
	ap.LoadSound() // actually load the sound

	fn := strings.TrimSuffix(ap.SpeechSeq.File, ".wav")
	ap.SndFileName = fn

	if ap.Corpus == "TIMIT" {
		fn := strings.Replace(fn, "ExpWavs", "", 1) // different directory for timing data
		fn = strings.Replace(fn, ".WAV", "", 1)
		fnm := fn + ".PHN.MS" // PHN is "Phone" and MS is milliseconds
		names := []string{}
		ap.SpeechSeq.Units, err = timit.LoadTimes(fnm, names) // names can be empty for timit, LoadTimes loads names
		if err != nil {
			fmt.Printf("LoadSnd: Problem loading %s transcription and timing data", fnm)
			return
		}
	} else {
		fmt.Println("NextSound: ap.Corpus no match")
	}
	//ap.InitProcess()

	if ap.SpeechSeq.Units == nil {
		fmt.Println("AdjSeqTimes: SpeechSeq.Units is nil. Some problem with loading file transcription and timing data")
		return
	}

	ap.AdjSeqTimes()

	// todo: this works for timit - but need a more general solution
	ap.SpeechSeq.TimeCur = ap.SpeechSeq.Units[0].AStart
	n := len(ap.SpeechSeq.Units) // find last unit
	if ap.SpeechSeq.Units[n-1].Name == "h#" {
		ap.SpeechSeq.TimeStop = ap.SpeechSeq.Units[n-1].AStart
	} else {
		fmt.Println("last unit of speech sequence not silence")
	}

	return
}

// LoadSound
func (ap *App) LoadSound() bool {
	if ap.Sound.Channels() > 1 {
		ap.Sound.SoundToTensor(&ap.Signal, -1)
	} else {
		ap.Sound.SoundToTensor(&ap.Signal, ap.Params.Channel)
	}
	return true
}

//// NextSegment calls to process the next segment of sound, loading a new sound if the last sound was fully processed
//func (ap *App) NextSegment() error {
//	if ap.MoreSegments == false {
//		done, err := ap.NextSound()
//		if done && err == nil {
//			return err
//		}
//		if err != nil {
//			return err
//		}
//	}
//	ap.SpeechSeq.TimeCur += float64(ap.Snd.Params.StrideMs)
//	ap.MoreSegments = ap.Snd.ProcessSegment()
//	if ap.SpeechSeq.TimeCur >= ap.SpeechSeq.TimeStop {
//		ap.MoreSegments = false
//	}
//	return nil
//}

// ClearSoundsAndData empties the sound list, sets current sound to nothing, etc
func ClearSoundsAndData(ap *App) {
	ap.SpeechSeq.File = ""
	ap.SndIdx = -1
	ap.MoreSegments = false // this will force a new sound to be loaded
}

// LoadSndData loads
func LoadSndData(ap *App) {
	ClearSoundsAndData(ap)
}

// AdjSeqTimes adjust for any offset if the sequence doesn't start at 0 ms. Also adjust for random silence that
// might have been added to front of signal
func (ap *App) AdjSeqTimes() {
	silence := ap.SpeechSeq.Silence // random silence added to start of sequence for variability
	offset := 0.0
	if ap.SpeechSeq.Units[0].Start > 0 {
		offset = ap.SpeechSeq.Units[0].Start // some sequences are sections of longer ones so times don't start at zero (not true for timit)
	}
	for i := range ap.SpeechSeq.Units {
		ap.SpeechSeq.Units[i].AStart = ap.SpeechSeq.Units[i].Start + silence - offset
		ap.SpeechSeq.Units[i].AEnd = ap.SpeechSeq.Units[i].End + silence - offset
	}
}

// IdxFmSnd simplies the lookup by keeping the corpus conditional in one function
func (ap *App) IdxFmSnd(s string) (idx int, ok bool) {
	idx = -1
	ok = false
	if ap.Corpus == "TIMIT" {
		idx, ok = timit.IdxFmSnd(s, ap.SpeechSeq.ID)
	} else if ap.Corpus == "SYNTHCVS" {
		idx, ok = synthcvs.IdxFmSnd(s, ap.SpeechSeq.ID)
	} else if ap.Corpus == "GRAFESTES" {
		idx, ok = grafestes.IdxFmSnd(s, ap.SpeechSeq.ID)
	} else {
		fmt.Println("IdxFmSnd: fell through corpus ifelse ")
	}
	return
}

// IdxFmSnd simplies the lookup by keeping the corpus conditional in one function
func (ap *App) SndFmIdx(idx int) (snd string, ok bool) {
	snd = ""
	ok = false
	if ap.Corpus == "TIMIT" {
		snd, ok = timit.SndFmIdx(idx, ap.SpeechSeq.ID)
	} else {
		fmt.Println("SndFmIdx: fell through corpus ifelse ")
	}
	return
}

// ProcessSegment processes the entire segment's input by processing a small overlapping set of samples on each pass
func (ap *App) ProcessSegment() {
	ap.Power.SetZeros()
	ap.LogPower.SetZeros()
	ap.PowerSegment.SetZeros()
	ap.LogPowerSegment.SetZeros()
	ap.MelFBankSegment.SetZeros()
	ap.MfccDctSegment.SetZeros()
	for ch := int(0); ch < ap.Sound.Channels(); ch++ {
		for s := 0; s < int(ap.Params.SegmentStepsTotal); s++ {
			err := ap.ProcessStep(ch, s)
			if err != nil {
				break
			}
		}
	}
	//fmt.Printf("total length = %v, remaining = %v\n", len(ap.Signal.Values), remaining)
}

// ProcessStep processes a step worth of sound input from current input_pos, and increment input_pos by input.step_samples
// Process the data by doing a fourier transform and computing the power spectrum, then apply mel filters to get the frequency
// bands that mimic the non-linear human perception of sound
func (ap *App) ProcessStep(ch int, step int) error {
	offset := ap.Params.Steps[step]
	start := sound.MSecToSamples(float32(ap.UserVals.SegmentStart), ap.Sound.SampleRate()) + offset
	err := ap.SndToWindow(start, ch)
	if err == nil {
		ap.Fft.Reset(ap.Params.WinSamples)
		ap.Dft.Filter(int(ch), int(step), &ap.Window, ap.FirstStep, ap.Params.WinSamples, ap.FftCoefs, ap.Fft, &ap.Power, &ap.LogPower, &ap.PowerSegment, &ap.LogPowerSegment)
		ap.Mel.Filter(int(ch), int(step), &ap.Window, &ap.MelFilters, &ap.Power, &ap.MelFBankSegment, &ap.MelFBank, &ap.MfccDctSegment, &ap.MfccDct)
		ap.FirstStep = false
	}
	return err
}

// SndToWindow gets sound from the signal (i.e. the slice of input values) at given position and channel, into Window
func (ap *App) SndToWindow(start, ch int) error {
	if ap.Signal.NumDims() == 1 {
		//start := ap.Segment*int(ap.Params.StrideSamples) + stepOffset // segments start at zero
		end := start + ap.Params.WinSamples
		if end > len(ap.Signal.Values) {
			return errors.New("SndToWindow: end beyond signal length!!")
		}
		var pad []float32
		if start < 0 && end <= 0 {
			pad = make([]float32, end-start)
			ap.Window.Values = pad[0:]
		} else if start < 0 && end > 0 {
			pad = make([]float32, 0-start)
			ap.Window.Values = pad[0:]
			ap.Window.Values = append(ap.Window.Values, ap.Signal.Values[0:end]...)
		} else {
			ap.Window.Values = ap.Signal.Values[start:end]
		}
		//fmt.Println("start / end in samples:", start, end)
	} else {
		// ToDo: implement
		fmt.Printf("SndToWindow: else case not implemented - please report this issue")
	}
	return nil
}

func (ap *App) GenGabors() {
	agabor.ToTensor(ap.GaborSpecs, &ap.GaborSet)
}

// ApplyGabor convolves the gabor filters with the mel output
func (ap *App) ApplyGabor() (tsr *etensor.Float32) {
	y1 := ap.MelFBank.Dim(0)
	y2 := ap.GaborSet.SizeY
	y := float32(y1 - y2)
	sy := (int(mat32.Floor(y/float32(ap.GaborSet.StrideY))) + 1) * 2 // double - two rows, off-center and on-center

	if ap.UserVals.SegmentEnd > ap.UserVals.SegmentStart {
		ap.UserVals.SegmentDuration = ap.UserVals.SegmentEnd - ap.UserVals.SegmentStart
	}
	if ap.UserVals.SegmentDuration <= 0 {
		ap.UserVals.SegmentDuration = int(ap.Params.SegmentMs)
	}
	x1 := float32(ap.UserVals.SegmentDuration)
	x2 := float32(ap.GaborSet.SizeX)
	x := float32(x1 - x2)
	active := agabor.Active(ap.GaborSpecs)
	sx := (int(mat32.Floor(x/float32(ap.GaborSet.StrideX))) + 1) * len(active)

	ap.GborOutput.SetShape([]int{ap.Sound.Channels(), sy, sx}, nil, []string{"chan", "freq", "time"})
	ap.ExtGi.SetShape([]int{sy, ap.GaborSet.Filters.Dim(0)}, nil, nil) // passed in for each channel
	ap.GborOutput.SetMetaData("odd-row", "true")
	ap.GborOutput.SetMetaData("grid-fill", ".9")
	ap.GborKwta.CopyShapeFrom(&ap.GborOutput)
	ap.GborKwta.CopyMetaData(&ap.GborOutput)

	for ch := int(0); ch < ap.Sound.Channels(); ch++ {
		agabor.Convolve(ch, &ap.MelFBankSegment, ap.GaborSet, &ap.GborOutput)
		//if ap.NeighInhib.On {
		//	ap.NeighInhib.Inhib4(&ap.GborOutput, &ap.ExtGi)
		//} else {
		//	ap.ExtGi.SetZeros()
		//}

		if ap.Kwta.On {
			ap.ApplyKwta(ch)
			tsr = &ap.GborKwta
		} else {
			tsr = &ap.GborOutput
		}
	}
	return tsr
}

// ApplyKwta runs the kwta algorithm on the raw activations
func (ap *App) ApplyKwta(ch int) {
	ap.GborKwta.CopyFrom(&ap.GborOutput)
	if ap.Kwta.On {
		rawSS := ap.GborOutput.SubSpace([]int{ch}).(*etensor.Float32)
		kwtaSS := ap.GborKwta.SubSpace([]int{ch}).(*etensor.Float32)
		if ap.KwtaPool == true {
			ap.Kwta.KWTAPool(rawSS, kwtaSS, &ap.Inhibs, &ap.ExtGi)
		} else {
			ap.Kwta.KWTALayer(rawSS, kwtaSS, &ap.ExtGi)
		}
	}
}

// ConfigGui configures the GoGi gui interface for this simulation,
func (ap *App) ConfigGui() *gi.Window {
	gi.SetAppName("Gabor View")
	gi.SetAppAbout("Application/Utility to allow viewing of gabor convolution with sound")

	ap.GUI.Win = gi.NewMainWindow("gb", "Gabor View", 1600, 1200)
	ap.GUI.ViewPort = ap.GUI.Win.Viewport
	ap.GUI.ViewPort.UpdateStart()

	mfr := ap.GUI.Win.SetMainFrame()

	ap.GUI.ToolBar = gi.AddNewToolBar(mfr, "tbar")
	ap.GUI.ToolBar.SetStretchMaxWidth()

	ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Init", Icon: "update",
		Tooltip: "Initialize everything including network weights, and start over.  Also applies current params.",
		Active:  egui.ActiveStopped,
		Func: func() {
			ap.Init()
			ap.GUI.UpdateWindow()
		},
	})

	ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Open Sound File",
		Icon:    "file-open",
		Tooltip: "Opens a file dialog for selecting a sound file",
		Active:  egui.ActiveStopped,
		Func: func() {
			exts := ".wav"
			//giv.FileViewDialog(ap.GUI.ViewPort, string(apFile), exts, giv.DlgOpts{Title: "Open .wav Sound File", Prompt: "Open a .wav file to load for sound processing."}, nil,
			giv.FileViewDialog(ap.GUI.ViewPort, ap.InitDir, exts, giv.DlgOpts{Title: "Open .wav Sound File", Prompt: "Open a .wav file to load for sound processing."}, nil,
				ap.GUI.Win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
					if sig == int64(gi.DialogAccepted) {
						dlg, _ := send.Embed(gi.KiT_Dialog).(*gi.Dialog)
						fn := giv.FileViewDialogValue(dlg)
						ap.SndFile = fn
						ap.SpeechSeq.File = fn
						ap.SndFilePath, ap.SndFileName = path.Split(fn)
						ap.GUI.Win.UpdateSig()
						ap.LoadSnd(ap.SndFile)
						ap.Sound.Load(ap.SndFile)
					}
				})
		},
	})

	ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Process and Apply", Icon: "update",
		Tooltip: "Process the next segment of audio using the SegStart and SegEnd params. Then apply the gabors to the Mel output",
		Active:  egui.ActiveStopped,
		Func: func() {
			err, _ := ap.InitProcess()
			if err == nil {
				ap.ProcessSegment()
				ap.ApplyGabor()
				ap.GUI.UpdateWindow()
			}
		},
	})

	ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Generate Gabors", Icon: "update",
		Tooltip: "Generate gabors from the gabor specifications",
		Active:  egui.ActiveStopped,
		Func: func() {
			ap.GenGabors()
			ap.GUI.UpdateWindow()
		},
	})

	//
	//ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Next Segment", Icon: "update",
	//	Tooltip: "Process the next segment of audio using the current parameters.",
	//	Active:  egui.ActiveStopped,
	//	Func: func() {
	//		ap.ProcessSegment()
	//		ap.GUI.UpdateWindow()
	//	},
	//})
	//

	split := gi.AddNewSplitView(mfr, "split")
	split.Dim = mat32.Y
	split.SetStretchMax()

	ap.GUI.StructView = giv.AddNewStructView(split, "app")
	ap.GUI.StructView.SetStruct(ap)

	snds := giv.AddNewSliceView(split, "snds")
	snds.Viewport = ap.GUI.ViewPort
	snds.SetSlice(&ap.SpeechSeq.Units)
	snds.NoAdd = true
	snds.NoDelete = true
	snds.SetInactiveState(true)

	specs := giv.AddNewTableView(split, "specs")
	specs.Viewport = ap.GUI.ViewPort
	specs.SetSlice(&ap.GaborSpecs)

	tv := gi.AddNewTabView(split, "tv")
	split.SetSplits(.3, .2, .2, .3)

	tg := tv.AddNewTab(etview.KiT_TensorGrid, "Gabors").(*etview.TensorGrid)
	tg.SetStretchMax()
	//ap.LogPowerSegment.SetMetaData("grid-min", "10")
	tg.SetTensor(&ap.GaborSet.Filters)

	tg = tv.AddNewTab(etview.KiT_TensorGrid, "Power").(*etview.TensorGrid)
	tg.SetStretchMax()
	ap.PowerGrid = tg
	ap.LogPowerSegment.SetMetaData("grid-min", "10")
	tg.SetTensor(&ap.LogPowerSegment)

	tg = tv.AddNewTab(etview.KiT_TensorGrid, "Mel").(*etview.TensorGrid)
	tg.SetStretchMax()
	ap.MelFBankGrid = tg
	tg.SetTensor(&ap.MelFBankSegment)

	tg = tv.AddNewTab(etview.KiT_TensorGrid, "Gabor Result").(*etview.TensorGrid)
	tg.SetStretchMax()
	ap.GaborOutGrid = tg
	tg.SetTensor(&ap.GborOutput)

	ap.GUI.FinalizeGUI(false)
	return ap.GUI.Win
}

// These props register Save methods so they can be used
var AppProps = ki.Props{}

func (ap *App) CmdArgs() {

}
