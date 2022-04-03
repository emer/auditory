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

// Table contains all the info for one table
type Table struct {
	Table *etable.Table     `desc:"jobs table"`
	View  *etview.TableView `desc:"view of table"`
	Sels  []string          `desc:"selected row ids in ascending order in view"`
}

// Params defines the sound input parameters for auditory processing
type Params struct {
	WinMs        float32 `def:"25" desc:"input window -- number of milliseconds worth of sound to filter at a time"`
	StepMs       float32 `def:"10" desc:"input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample"`
	SegmentStart float32 `desc:"start of sound segment in milliseconds"`
	SegmentEnd   float32 `desc:"end of sound segment in milliseconds"`
	BorderSteps  int     `view:"-" def:"0" desc:"overlap with previous segment"`
	Channel      int     `view:"-" desc:"specific channel to process, if input has multiple channels, and we only process one of them (-1 = process all)"`
	PadValue     float32 `view:"-"`
	Resize       bool    `desc:"if resize is true segment durations will be lengthened (a bit before and a bit after) to be a multiple of the filter width"`

	// these are calculated
	WinSamples  int   `view:"-" desc:"number of samples to process each step"`
	StepSamples int   `view:"-" desc:"number of samples to step input by"`
	StepsTotal  int   `view:"-" desc:"SegmentSteps plus steps overlapping next segment or for padding if no next segment"`
	Steps       []int `view:"-" desc:"pre-calculated start position for each step"`
}

type App struct {
	Nm      string `view:"-" desc:"name of this environment"`
	Dsc     string `view:"-" desc:"description of this environment"`
	Corpus  string `desc:"the set of sound files"`
	SndName string `desc:"name of the open sound file"`
	SndPath string `desc:"path to the open sound file"`
	SndFile string `view:"-" desc:"full path of open sound file"`

	GUI    egui.GUI        `view:"-" desc:"manages all the gui elements"`
	Params Params          `view:"inline" desc:"fundamental processing parameters"`
	Sound  sound.Wave      `view:"inline"`
	Signal etensor.Float32 `view:"-" desc:" the full sound input obtained from the sound input - plus any added padding"`
	Window etensor.Float32 `view:"-" desc:" [Input.WinSamples] the raw sound input, one channel at a time"`

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

	GaborSpecs []agabor.Filter   `view:"no-inline" desc:"array of params describing each gabor filter"`
	GaborSet   agabor.FilterSet  `view:"inline" desc:"a set of gabor filters with same x and y dimensions"`
	GborOutput etensor.Float32   `view:"no-inline" desc:" raw output of Gabor -- full segment's worth of gabor steps"`
	GborKwta   etensor.Float32   `view:"no-inline" desc:" post-kwta output of full segment's worth of gabor steps"`
	Inhibs     fffb.Inhibs       `view:"no-inline" desc:"inhibition values for A1 KWTA"`
	ExtGi      etensor.Float32   `view:"no-inline" desc:"A1 simple extra Gi from neighbor inhibition tensor"`
	NeighInhib kwta.NeighInhib   `desc:"neighborhood inhibition for V1s -- each unit gets inhibition from same feature in nearest orthogonal neighbors -- reduces redundancy of feature code"`
	Kwta       kwta.KWTA         `desc:"kwta parameters, using FFFB form"`
	FftCoefs   []complex128      `view:"-" desc:" discrete fourier transform (fft) output complex representation"`
	Fft        *fourier.CmplxFFT `view:"-" desc:" struct for fast fourier transform"`

	Sequence speech.Sequence
	SeqTable giv.TableView
	//Snds     etable.Table
	SndsTable Table

	// internal state - view:"-"
	ToolBar      *gi.ToolBar `view:"-" desc:" the master toolbar"`
	MoreSegments bool        `view:"-" desc:" are there more samples to process"`
	FirstStep    bool        `view:"-" desc:" if first frame to process -- turns off prv smoothing of dft power"`
}

// Init
func (ap *App) Init() {
	ap.ParamDefaults()
	ap.Mel.Defaults()
	ap.InitGabors()
	ap.UpdateGabors()
}

// ParamDefaults initializes the Input
func (ap *App) ParamDefaults() {
	ap.Params.WinMs = 25.0
	ap.Params.StepMs = 10.0
	ap.Params.Channel = 0
	ap.Params.PadValue = 0.0
	ap.Params.BorderSteps = 0
	ap.Sequence.Units = append(ap.Sequence.Units, *new(speech.Unit))
	ap.Params.Resize = true
}

// Config configures all the elements using the standard functions
func (ap *App) Config() {
	ap.Corpus = "TIMIT"
	if ap.Corpus == "TIMIT" {
		ap.SndPath = "/Users/rohrlich/ccn_images/sound/timit/TIMIT/TRAIN/"
	} else {
		ap.SndPath = "~"
	}

	ap.ConfigSoundsTable()
}

func (ap *App) InitGabors() {
	ap.GaborSet.Filters.SetMetaData("min", "-.25")
	ap.GaborSet.Filters.SetMetaData("max", ".25")

	ap.GaborSet.SizeX = 9
	ap.GaborSet.SizeY = 9
	ap.GaborSet.Gain = 1.5
	ap.GaborSet.StrideX = 6
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

func (ap *App) UpdateGabors() {
	if ap.GaborSet.SizeX < ap.GaborSet.StrideX {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Stride > size", Prompt: "The stride in X is greater than the filter size in X"}, gi.AddOk, gi.AddCancel, nil, nil)
	}
	active := agabor.Active(ap.GaborSpecs)
	ap.GaborSet.Filters.SetShape([]int{len(active), ap.GaborSet.SizeY, ap.GaborSet.SizeX}, nil, nil)
	agabor.ToTensor(ap.GaborSpecs, &ap.GaborSet)
}

func (ap *App) Process() (err error) {
	if ap.Sound.Buf == nil {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Sound buffer is empty", Prompt: "Open a sound file before processing"}, gi.AddOk, gi.NoCancel, nil, nil)
		errors.New("Load a sound and try again")
	}

	if ap.Params.SegmentEnd <= ap.Params.SegmentStart {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "End <= Start", Prompt: "SegmentEnd must be greater than SegmentStart."}, gi.AddOk, gi.NoCancel, nil, nil)
		return errors.New("SegmentEnd <= SegmentStart")
	}

	duration := ap.Params.SegmentEnd - ap.Params.SegmentStart
	stepMs := ap.Params.StepMs
	sizeXMs := float32(ap.GaborSet.SizeX) * stepMs
	strideXMs := float32(ap.GaborSet.StrideX) * stepMs
	add := float32(0)
	if duration < sizeXMs {
		add = sizeXMs - duration
	} else { // duration is longer than one filter so find the next stride end
		d := duration
		d -= sizeXMs
		rem := float32(int(d) % int(strideXMs))
		add = strideXMs - rem
	}
	if ap.Params.SegmentStart-add < 0 {
		ap.Params.SegmentEnd += add
	} else {
		ap.Params.SegmentStart -= add / 2
		ap.Params.SegmentEnd += add / 2
	}
	ap.GUI.UpdateWindow()

	sr := ap.Sound.SampleRate()
	if sr <= 0 {
		fmt.Println("sample rate <= 0")
		return errors.New("sample rate <= 0")

	}
	ap.Params.WinSamples = sound.MSecToSamples(ap.Params.WinMs, sr)
	ap.Params.StepSamples = sound.MSecToSamples(ap.Params.StepMs, sr)

	// round up to nearest step interval
	segmentMs := ap.Params.SegmentEnd - ap.Params.SegmentStart
	segmentMs = segmentMs + ap.Params.StepMs*float32(int(segmentMs)%int(ap.Params.StepMs))

	steps := int(segmentMs / ap.Params.StepMs)
	ap.Params.StepsTotal = steps + 2*ap.Params.BorderSteps

	winSamplesHalf := ap.Params.WinSamples/2 + 1
	ap.Dft.Initialize(ap.Params.WinSamples)
	ap.Mel.InitFilters(ap.Params.WinSamples, ap.Sound.SampleRate(), &ap.MelFilters) // call after non-default values are set!
	ap.Window.SetShape([]int{ap.Params.WinSamples}, nil, nil)
	ap.Power.SetShape([]int{winSamplesHalf}, nil, nil)
	ap.LogPower.CopyShapeFrom(&ap.Power)
	ap.PowerSegment.SetShape([]int{ap.Params.StepsTotal, winSamplesHalf, ap.Sound.Channels()}, nil, nil)
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
	ap.Params.Steps = make([]int, ap.Params.StepsTotal)
	for i := 0; i < ap.Params.StepsTotal; i++ {
		ap.Params.Steps[i] = ap.Params.StepSamples * (i - stepsBack)
	}

	ap.MelFBank.SetShape([]int{ap.Mel.FBank.NFilters}, nil, nil)
	ap.MelFBankSegment.SetShape([]int{ap.Params.StepsTotal, ap.Mel.FBank.NFilters, ap.Sound.Channels()}, nil, nil)
	if ap.Mel.MFCC {
		ap.MfccDctSegment.CopyShapeFrom(&ap.MelFBankSegment)
		ap.MfccDct.SetShape([]int{ap.Mel.FBank.NFilters}, nil, nil)
	}
	samples := sound.MSecToSamples(ap.Params.SegmentEnd-ap.Params.SegmentStart, ap.Sound.SampleRate())
	siglen := len(ap.Signal.Values) - samples*ap.Sound.Channels()
	siglen = siglen / ap.Sound.Channels()

	ap.Power.SetZeros()
	ap.LogPower.SetZeros()
	ap.PowerSegment.SetZeros()
	ap.LogPowerSegment.SetZeros()
	ap.MelFBankSegment.SetZeros()
	ap.MfccDctSegment.SetZeros()

	for ch := int(0); ch < ap.Sound.Channels(); ch++ {
		for s := 0; s < int(ap.Params.StepsTotal); s++ {
			err := ap.ProcessStep(ch, s)
			if err != nil {
				break
			}
		}
	}
	return nil
}

// LoadSnd
func (ap *App) LoadSnd(path string) {
	ap.SndFile = path
	err := ap.Sound.Load(path)
	if err != nil {
		log.Printf("LoadSnd: error loading sound -- %v\n, err", ap.Sequence.File)
		return
	}
	ap.LoadSound() // actually load the sound

	fn := strings.TrimSuffix(ap.Sequence.File, ".wav")
	ap.SndName = fn

	if ap.Corpus == "TIMIT" {
		fn := strings.Replace(fn, "ExpWavs", "", 1) // different directory for timing data
		fn = strings.Replace(fn, ".WAV", "", 1)
		fnm := fn + ".PHN.MS" // PHN is "Phone" and MS is milliseconds
		names := []string{}
		ap.Sequence.Units, err = timit.LoadTimes(fnm, names) // names can be empty for timit, LoadTimes loads names
		if err != nil {
			fmt.Printf("LoadSnd: Problem loading %s transcription and timing data", fnm)
			return
		}
	} else {
		fmt.Println("NextSound: ap.Corpus no match")
	}

	if ap.Sequence.Units == nil {
		fmt.Println("AdjSeqTimes: SpeechSeq.Units is nil. Some problem with loading file transcription and timing data")
		return
	}
	ap.AdjSeqTimes()

	// todo: this works for timit - but need a more general solution
	ap.Sequence.TimeCur = ap.Sequence.Units[0].AStart
	n := len(ap.Sequence.Units) // find last unit
	if ap.Sequence.Units[n-1].Name == "h#" {
		ap.Sequence.TimeStop = ap.Sequence.Units[n-1].AStart
	} else {
		fmt.Println("last unit of speech sequence not silence")
	}

	ap.SndsTable.Table.AddRows(len(ap.Sequence.Units))
	for r, s := range ap.Sequence.Units {
		ap.SndsTable.Table.SetCellString("Sound", r, s.Name)
		ap.SndsTable.Table.SetCellFloat("Start", r, s.AStart)
		ap.SndsTable.Table.SetCellFloat("End", r, s.AEnd)
	}
	ap.SndsTable.View.UpdateTable()
	ap.GUI.UpdateWindow()

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

// ClearSoundsAndData empties the sound list, sets current sound to nothing, etc
func ClearSoundsAndData(ap *App) {
	ap.Sequence.File = ""
	ap.MoreSegments = false // this will force a new sound to be loaded
}

// LoadSndData loads
func LoadSndData(ap *App) {
	ClearSoundsAndData(ap)
}

// AdjSeqTimes adjust for any offset if the sequence doesn't start at 0 ms. Also adjust for random silence that
// might have been added to front of signal
func (ap *App) AdjSeqTimes() {
	silence := ap.Sequence.Silence // random silence added to start of sequence for variability
	offset := 0.0
	if ap.Sequence.Units[0].Start > 0 {
		offset = ap.Sequence.Units[0].Start // some sequences are sections of longer ones so times don't start at zero (not true for timit)
	}
	for i := range ap.Sequence.Units {
		ap.Sequence.Units[i].AStart = ap.Sequence.Units[i].Start + silence - offset
		ap.Sequence.Units[i].AEnd = ap.Sequence.Units[i].End + silence - offset
	}
}

// IdxFmSnd simplies the lookup by keeping the corpus conditional in one function
func (ap *App) IdxFmSnd(s string) (idx int, ok bool) {
	idx = -1
	ok = false
	if ap.Corpus == "TIMIT" {
		idx, ok = timit.IdxFmSnd(s, ap.Sequence.ID)
	} else if ap.Corpus == "SYNTHCVS" {
		idx, ok = synthcvs.IdxFmSnd(s, ap.Sequence.ID)
	} else if ap.Corpus == "GRAFESTES" {
		idx, ok = grafestes.IdxFmSnd(s, ap.Sequence.ID)
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
		snd, ok = timit.SndFmIdx(idx, ap.Sequence.ID)
	} else {
		fmt.Println("SndFmIdx: fell through corpus ifelse ")
	}
	return
}

// ProcessStep processes a step worth of sound input from current input_pos, and increment input_pos by input.step_samples
// Process the data by doing a fourier transform and computing the power spectrum, then apply mel filters to get the frequency
// bands that mimic the non-linear human perception of sound
func (ap *App) ProcessStep(ch int, step int) error {
	offset := ap.Params.Steps[step]
	start := sound.MSecToSamples(ap.Params.SegmentStart, ap.Sound.SampleRate()) + offset
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

// ApplyGabor convolves the gabor filters with the mel output
func (ap *App) ApplyGabor() (tsr *etensor.Float32) {
	if ap.Params.SegmentEnd <= ap.Params.SegmentStart {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "End <= Start", Prompt: "SegmentEnd must be greater than SegmentStart."}, gi.AddOk, gi.NoCancel, nil, nil)
		return
	}

	y1 := ap.MelFBank.Dim(0)
	y2 := ap.GaborSet.SizeY
	y := float32(y1 - y2)
	sy := (int(mat32.Floor(y/float32(ap.GaborSet.StrideY))) + 1) * 2 // double - two rows, off-center and on-center

	x1 := float32(ap.Params.SegmentEnd-ap.Params.SegmentStart) / ap.Params.StepMs
	x2 := float32(ap.GaborSet.SizeX)
	x := x1 - x2
	active := agabor.Active(ap.GaborSpecs)
	sx := (int(mat32.Floor(x/float32(ap.GaborSet.StrideX))) + 1) * len(active)

	ap.UpdateGabors()
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
		// This app is 2D only - no pools
		//if ap.KwtaPool == true {
		//	ap.Kwta.KWTAPool(rawSS, kwtaSS, &ap.Inhibs, &ap.ExtGi)
		//} else {
		ap.Kwta.KWTALayer(rawSS, kwtaSS, &ap.ExtGi)
		//}
	}
}

// ConfigSoundsTable
func (ap *App) ConfigSoundsTable() {
	ap.SndsTable.Table = &etable.Table{}
	ap.SndsTable.Table.SetMetaData("name", "The Loaded Sounds")
	ap.SndsTable.Table.SetMetaData("desc", "Sounds loaded from audio files")
	ap.SndsTable.Table.SetMetaData("read-only", "true")

	sch := etable.Schema{
		{"Sound", etensor.STRING, nil, nil},
		{"Start", etensor.FLOAT32, nil, nil},
		{"End", etensor.FLOAT32, nil, nil},
	}
	ap.SndsTable.Table.SetFromSchema(sch, 0)
}

// ConfigTableView configures given tableview
func (ap *App) ConfigTableView(tv *etview.TableView) {
	tv.SetProp("inactive", true)
	tv.SetInactive()
	tv.SliceViewSig.Connect(ap.GUI.ViewPort, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(giv.SliceViewDoubleClicked) {
			row := ap.SndsTable.View.SelectedIdx
			u := ap.Sequence.Units[row]
			ap.Params.SegmentStart = float32(u.AStart)
			ap.Params.SegmentEnd = float32(u.AEnd)
			ap.GUI.UpdateWindow()
		}
	})
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
			giv.FileViewDialog(ap.GUI.ViewPort, ap.SndPath, exts, giv.DlgOpts{Title: "Open .wav Sound File", Prompt: "Open a .wav file to load for sound processing."}, nil,
				ap.GUI.Win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
					if sig == int64(gi.DialogAccepted) {
						dlg, _ := send.Embed(gi.KiT_Dialog).(*gi.Dialog)
						fn := giv.FileViewDialogValue(dlg)
						ap.SndFile = fn
						ap.Sequence.File = fn
						ap.SndPath, ap.SndName = path.Split(fn)
						ap.GUI.Win.UpdateSig()
						ap.LoadSnd(ap.SndFile)
						ap.Sound.Load(ap.SndFile)
					}
				})
		},
	})

	ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Process", Icon: "play",
		Tooltip: "Process the segment of audio from SegmentStart to SegmentEnd applying the gabor filters to the Mel tensor",
		Active:  egui.ActiveStopped,
		Func: func() {
			err := ap.Process()
			if err == nil {
				ap.ApplyGabor()
				ap.GUI.UpdateWindow()
			}
		},
	})

	ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Update Gabors", Icon: "update",
		Tooltip: "Call this to see the result of changing the Gabor specifications",
		Active:  egui.ActiveStopped,
		Func: func() {
			ap.UpdateGabors()
			ap.GUI.UpdateWindow()
		},
	})

	split1 := gi.AddNewSplitView(mfr, "split1")
	split1.Dim = mat32.X
	split1.SetStretchMax()

	split := gi.AddNewSplitView(split1, "split")
	split.Dim = mat32.Y
	split.SetStretchMax()

	tv1 := gi.AddNewTabView(split1, "tv1")
	ap.SndsTable.View = tv1.AddNewTab(etview.KiT_TableView, "Sounds").(*etview.TableView)
	ap.ConfigTableView(ap.SndsTable.View)
	ap.SndsTable.View.SetTable(ap.SndsTable.Table, nil)

	split1.SetSplits(.80, .20)

	ap.GUI.StructView = giv.AddNewStructView(split, "app")
	ap.GUI.StructView.SetStruct(ap)

	specs := giv.AddNewTableView(split, "specs")
	specs.Viewport = ap.GUI.ViewPort
	specs.SetSlice(&ap.GaborSpecs)

	tv := gi.AddNewTabView(split, "tv")
	split.SetSplits(.3, .3, .4)

	tg := tv.AddNewTab(etview.KiT_TensorGrid, "Gabors").(*etview.TensorGrid)
	tg.SetStretchMax()
	//ap.LogPowerSegment.SetMetaData("grid-min", "10")
	tg.SetTensor(&ap.GaborSet.Filters)

	tg = tv.AddNewTab(etview.KiT_TensorGrid, "Power").(*etview.TensorGrid)
	tg.SetStretchMax()
	ap.LogPowerSegment.SetMetaData("grid-min", "10")
	tg.SetTensor(&ap.LogPowerSegment)

	tg = tv.AddNewTab(etview.KiT_TensorGrid, "Mel").(*etview.TensorGrid)
	tg.SetStretchMax()
	tg.SetTensor(&ap.MelFBankSegment)

	tg = tv.AddNewTab(etview.KiT_TensorGrid, "Gabor Result").(*etview.TensorGrid)
	tg.SetStretchMax()
	tg.SetTensor(&ap.GborOutput)

	ap.GUI.FinalizeGUI(false)
	return ap.GUI.Win
}

// These props register Save methods so they can be used
var AppProps = ki.Props{}

func (ap *App) CmdArgs() {

}
