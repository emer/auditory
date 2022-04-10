package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

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
	TheApp.GUI.ToolBar.UpdateActions()
	win.StartEventLoop()
}

////////////////////////////////////////////////////////////////////////////////////////////
// Environment - params and config for the train/test environment

// Table contains all the info for one table
type Table struct {
	Table *etable.Table     `desc:"jobs table"`
	View  *etview.TableView `view:"-" desc:"view of table"`
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

type ProcessParams struct {
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
}

type GaborParams struct {
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
}

type App struct {
	Nm        string            `view:"-" desc:"name of this environment"`
	Dsc       string            `view:"-" desc:"description of this environment"`
	GUI       egui.GUI          `view:"-" desc:"manages all the gui elements"`
	Corpus    string            `desc:"the set of sound files"`
	SndPath   string            `desc:"path to the open sound file"`
	SndFile   string            `view:"-" desc:"full path of open sound file"`
	Sequence  []speech.Sequence `view:"no-inline" desc:"a slice of Sequence structs, one per open sound file"`
	SndsTable Table             `desc:"table of sounds from the open sound files"`
	Row       int               `view:"-" desc:"SndsTable selected row, as seen by user"`

	Params    Params          `view:"inline" desc:"fundamental processing parameters"`
	PParams1  ProcessParams   `desc:"the power and mel parameters for processing the sound to generate a mel filter bank for sound 1"`
	GParams1  GaborParams     `desc:"gabor filter specifications and parameters for applying gabors to the mel output for sound 1"`
	PParams2  ProcessParams   `desc:"the power and mel parameters for processing the sound to generate a mel filter bank for sound 2"`
	GParams2  GaborParams     `desc:"gabor filter specifications and parameters for applying gabors to the mel output for sound 2"`
	Sound     sound.Wave      `view:"inline"`
	Signal    etensor.Float32 `view:"-" desc:" the full sound input obtained from the sound input - plus any added padding"`
	Window    etensor.Float32 `view:"-" desc:" [Input.WinSamples] the raw sound input, one channel at a time"`
	ByTime    bool            `desc:"display the gabor filtering result by time and then by filter, default is to order by filter and then time"`
	FirstStep bool            `view:"-" desc:" if first frame to process -- turns off prv smoothing of dft power"`
}

// this registers this Sim Type and gives it properties that e.g.,
// prompt for filename for save methods.
var KiT_App = kit.Types.AddType(&App{}, AppProps)

// TheApp is the overall state for this simulation
var TheApp App

func (ss *App) New() {
}

// Init
func (ap *App) Init() {
	ap.ParamDefaults()
	ap.PParams1.Mel.Defaults()
	ap.PParams2.Mel.Defaults()
	ap.InitGabors(&ap.GParams1)
	ap.InitGabors(&ap.GParams2)
	ap.UpdateGabors(&ap.GParams1)
	ap.UpdateGabors(&ap.GParams2)
	ap.ByTime = true
	ap.GUI.Active = false
}

// ParamDefaults initializes the sound processing parameters
func (ap *App) ParamDefaults() {
	ap.Params.WinMs = 25.0
	ap.Params.StepMs = 10.0
	ap.Params.Channel = 0
	ap.Params.PadValue = 0.0
	ap.Params.BorderSteps = 0
	ap.Params.Resize = true
}

// Config configures environment elements
func (ap *App) Config() {
	ap.Corpus = "TIMIT"
	if ap.Corpus == "TIMIT" {
		ap.SndPath = "/Users/rohrlich/ccn_images/sound/timit/TIMIT/TRAIN/"
	} else {
		ap.SndPath = "~"
	}

	ap.ConfigSoundsTable()
}

// InitGabors renders the gabor filters using the gabor specifications
func (ap *App) InitGabors(params *GaborParams) {
	params.GaborSet.Filters.SetMetaData("min", "-.25")
	params.GaborSet.Filters.SetMetaData("max", ".25")

	params.GaborSet.SizeX = 9
	params.GaborSet.SizeY = 9
	params.GaborSet.Gain = 1.5
	params.GaborSet.StrideX = 6
	params.GaborSet.StrideY = 3
	params.GaborSet.Distribute = false

	orient := []float32{0, 45, 90, 135}
	wavelen := []float32{2.0}
	phase := []float32{0}
	sigma := []float32{0.5}

	params.GaborSpecs = nil // in case there are some specs already

	for _, or := range orient {
		for _, wv := range wavelen {
			for _, ph := range phase {
				for _, wl := range sigma {
					spec := agabor.Filter{WaveLen: wv, Orientation: or, SigmaWidth: wl, SigmaLength: wl, PhaseOffset: ph, CircleEdge: true}
					params.GaborSpecs = append(params.GaborSpecs, spec)
				}
			}
		}
	}
}

// UpdateGabors rerenders based on current spec and filterset values
func (ap *App) UpdateGabors(params *GaborParams) {
	if params.GaborSet.SizeX < params.GaborSet.StrideX {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Stride > size", Prompt: "The stride in X is greater than the filter size in X"}, gi.AddOk, gi.AddCancel, nil, nil)
	}
	active := agabor.Active(params.GaborSpecs)
	params.GaborSet.Filters.SetShape([]int{len(active), params.GaborSet.SizeY, params.GaborSet.SizeX}, nil, nil)
	agabor.ToTensor(params.GaborSpecs, &params.GaborSet)
}

// Process generates the mel output and from that the result of the convolution with the gabor filters
// Must call ProcessSetup() first !
func (ap *App) Process(pparams *ProcessParams, gparams *GaborParams) (err error) {
	ap.LoadSound() // actually load the sound

	if ap.Sound.Buf == nil {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Sound buffer is empty", Prompt: "Open a sound file before processing"}, gi.AddOk, gi.NoCancel, nil, nil)
		return errors.New("Load a sound file and try again")
	}

	if ap.Params.SegmentEnd <= ap.Params.SegmentStart {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "End <= Start", Prompt: "SegmentEnd must be greater than SegmentStart."}, gi.AddOk, gi.NoCancel, nil, nil)
		return errors.New("SegmentEnd <= SegmentStart")
	}

	duration := ap.Params.SegmentEnd - ap.Params.SegmentStart
	stepMs := ap.Params.StepMs
	sizeXMs := float32(gparams.GaborSet.SizeX) * stepMs
	strideXMs := float32(gparams.GaborSet.StrideX) * stepMs
	add := float32(0)
	if duration < sizeXMs {
		add = sizeXMs - duration
	} else { // duration is longer than one filter so find the next stride end
		d := duration
		d -= sizeXMs
		rem := float32(int(d) % int(strideXMs))
		if rem > 0 {
			add = strideXMs - rem
		}
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
	pparams.Dft.Initialize(ap.Params.WinSamples)
	pparams.Mel.InitFilters(ap.Params.WinSamples, ap.Sound.SampleRate(), &pparams.MelFilters) // call after non-default values are set!
	ap.Window.SetShape([]int{ap.Params.WinSamples}, nil, nil)
	pparams.Power.SetShape([]int{winSamplesHalf}, nil, nil)
	pparams.LogPower.CopyShapeFrom(&pparams.Power)
	pparams.PowerSegment.SetShape([]int{ap.Params.StepsTotal, winSamplesHalf, ap.Sound.Channels()}, nil, nil)
	if pparams.Dft.CompLogPow {
		pparams.LogPowerSegment.CopyShapeFrom(&pparams.PowerSegment)
	}

	gparams.FftCoefs = make([]complex128, ap.Params.WinSamples)
	gparams.Fft = fourier.NewCmplxFFT(len(gparams.FftCoefs))

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

	pparams.MelFBank.SetShape([]int{pparams.Mel.FBank.NFilters}, nil, nil)
	pparams.MelFBankSegment.SetShape([]int{ap.Params.StepsTotal, pparams.Mel.FBank.NFilters, ap.Sound.Channels()}, nil, nil)
	if pparams.Mel.MFCC {
		pparams.MfccDctSegment.CopyShapeFrom(&pparams.MelFBankSegment)
		pparams.MfccDct.SetShape([]int{pparams.Mel.FBank.NFilters}, nil, nil)
	}
	samples := sound.MSecToSamples(ap.Params.SegmentEnd-ap.Params.SegmentStart, ap.Sound.SampleRate())
	siglen := len(ap.Signal.Values) - samples*ap.Sound.Channels()
	siglen = siglen / ap.Sound.Channels()

	pparams.Power.SetZeros()
	pparams.LogPower.SetZeros()
	pparams.PowerSegment.SetZeros()
	pparams.LogPowerSegment.SetZeros()
	pparams.MelFBankSegment.SetZeros()
	pparams.MfccDctSegment.SetZeros()

	for ch := int(0); ch < ap.Sound.Channels(); ch++ {
		for s := 0; s < int(ap.Params.StepsTotal); s++ {
			err := ap.ProcessStep(ch, s, pparams, gparams)
			if err != nil {
				break
			}
		}
	}
	return nil
}

// ProcessStep processes a step worth of sound input from current input_pos, and increment input_pos by input.step_samples
// Process the data by doing a fourier transform and computing the power spectrum, then apply mel filters to get the frequency
// bands that mimic the non-linear human perception of sound
func (ap *App) ProcessStep(ch int, step int, pparams *ProcessParams, gparams *GaborParams) error {
	offset := ap.Params.Steps[step]
	start := sound.MSecToSamples(ap.Params.SegmentStart, ap.Sound.SampleRate()) + offset
	err := ap.SndToWindow(start, ch)
	if err == nil {
		gparams.Fft.Reset(ap.Params.WinSamples)
		pparams.Dft.Filter(int(ch), int(step), &ap.Window, ap.FirstStep, ap.Params.WinSamples, gparams.FftCoefs, gparams.Fft, &pparams.Power, &pparams.LogPower, &pparams.PowerSegment, &pparams.LogPowerSegment)
		pparams.Mel.Filter(int(ch), int(step), &ap.Window, &pparams.MelFilters, &pparams.Power, &pparams.MelFBankSegment, &pparams.MelFBank, &pparams.MfccDctSegment, &pparams.MfccDct)
		ap.FirstStep = false
	}
	return err
}

// LoadTranscription loads the transcription file. The sound file is loaded at start of processing by calling LoadSound()
func (ap *App) LoadTranscription(fpth string) {
	ap.SndFile = fpth
	err := ap.Sound.Load(fpth)
	if err != nil {
		log.Printf("LoadTranscription: error loading sound -- %v\n, err", fpth)
		return
	}

	seq := new(speech.Sequence)
	seq.File = fpth
	ap.Sequence = append(ap.Sequence, *seq)
	ap.ConfigTableView(ap.SndsTable.View)
	ap.GUI.Win.UpdateSig()

	fn := strings.TrimSuffix(seq.File, ".wav")
	if ap.Corpus == "TIMIT" {
		fn := strings.Replace(fn, "ExpWavs", "", 1) // different directory for timing data
		fn = strings.Replace(fn, ".WAV", "", 1)
		fnm := fn + ".PHN.MS" // PHN is "Phone" and MS is milliseconds
		names := []string{}
		seq.Units, err = timit.LoadTimes(fnm, names) // names can be empty for timit, LoadTimes loads names
		if err != nil {
			fmt.Printf("LoadTranscription: Problem loading %s transcription and timing data", fnm)
			return
		}
	} else {
		fmt.Println("NextSound: ap.Corpus no match")
	}

	if seq.Units == nil {
		fmt.Println("AdjSeqTimes: SpeechSeq.Units is nil. Some problem with loading file transcription and timing data")
		return
	}
	ap.AdjSeqTimes(seq)

	// todo: this works for timit - but need a more general solution
	seq.TimeCur = seq.Units[0].AStart
	n := len(seq.Units) // find last unit
	if seq.Units[n-1].Name == "h#" {
		seq.TimeStop = seq.Units[n-1].AStart
	} else {
		fmt.Println("last unit of speech sequence not silence")
	}

	curRows := ap.SndsTable.Table.Rows
	ap.SndsTable.Table.AddRows(len(seq.Units))
	fpth, nm := path.Split(fn)
	i := strings.LastIndex(nm, ".")
	if i > 0 {
		nm = nm[0:i]
	}

	fpth = strings.TrimSuffix(fpth, "/")
	splits := strings.Split(fpth, "/")
	n = len(splits)
	if n >= 2 {
		fpth = splits[n-2] + "/" + splits[n-1]
	} else if n >= 1 {
		fpth = splits[n-1]
	}

	for r, s := range seq.Units {
		r = r + curRows
		ap.SndsTable.Table.SetCellString("Sound", r, s.Name)
		ap.SndsTable.Table.SetCellFloat("Start", r, s.AStart)
		ap.SndsTable.Table.SetCellFloat("End", r, s.AEnd)
		ap.SndsTable.Table.SetCellString("File", r, nm)
		ap.SndsTable.Table.SetCellString("Dir", r, fpth)
	}
	ap.SndsTable.View.UpdateTable()
	ap.GUI.Active = true
	ap.GUI.UpdateWindow()

	return
}

// LoadSound loads the sound file, e.g. .wav file, the transcription is loaded in LoadTranscription()
func (ap *App) LoadSound() bool {
	if ap.Sound.Channels() > 1 {
		ap.Sound.SoundToTensor(&ap.Signal, -1)
	} else {
		ap.Sound.SoundToTensor(&ap.Signal, ap.Params.Channel)
	}
	return true
}

// FilterSounds filters the table available sounds
func (ap *App) FilterSounds(sound string) {
	ap.SndsTable.View.Table.FilterColName("Sound", sound, false, true, true)
}

// UnfilterSounds clears the table of sounds
func (ap *App) UnfilterSounds() {
	ap.SndsTable.View.Table.Sequential()
}

// AdjSeqTimes adjust for any offset if the sequence doesn't start at 0 ms. Also adjust for random silence that
// might have been added to front of signal
func (ap *App) AdjSeqTimes(seq *speech.Sequence) {
	silence := seq.Silence // random silence added to start of sequence for variability
	offset := 0.0
	if seq.Units[0].Start > 0 {
		offset = seq.Units[0].Start // some sequences are sections of longer ones so times don't start at zero (not true for timit)
	}
	for i := range seq.Units {
		seq.Units[i].AStart = seq.Units[i].Start + silence - offset
		seq.Units[i].AEnd = seq.Units[i].End + silence - offset
	}
}

// IdxFmSnd simplies the lookup by keeping the corpus conditional in one function
func (ap *App) IdxFmSnd(seq speech.Sequence, s string) (idx int, ok bool) {
	idx = -1
	ok = false
	if ap.Corpus == "TIMIT" {
		idx, ok = timit.IdxFmSnd(s, seq.ID)
	} else if ap.Corpus == "SYNTHCVS" {
		idx, ok = synthcvs.IdxFmSnd(s, seq.ID)
	} else if ap.Corpus == "GRAFESTES" {
		idx, ok = grafestes.IdxFmSnd(s, seq.ID)
	} else {
		fmt.Println("IdxFmSnd: fell through corpus ifelse ")
	}
	return
}

// IdxFmSnd simplies the lookup by keeping the corpus conditional in one function
func (ap *App) SndFmIdx(seq speech.Sequence, idx int) (snd string, ok bool) {
	snd = ""
	ok = false
	if ap.Corpus == "TIMIT" {
		snd, ok = timit.SndFmIdx(idx, seq.ID)
	} else {
		fmt.Println("SndFmIdx: fell through corpus ifelse ")
	}
	return
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
func (ap *App) ApplyGabor(pparams *ProcessParams, gparams *GaborParams) {
	// determine gabor output size
	y1 := pparams.MelFBankSegment.Dim(1)
	y2 := gparams.GaborSet.SizeY
	y := float32(y1 - y2)
	sy := (int(mat32.Floor(y/float32(gparams.GaborSet.StrideY))) + 1) * 2 // double - two rows, off-center and on-center

	x1 := pparams.MelFBankSegment.Dim(0)
	x2 := gparams.GaborSet.SizeX
	x := x1 - x2
	active := agabor.Active(gparams.GaborSpecs)
	sx := (int(mat32.Floor(float32(x)/float32(gparams.GaborSet.StrideX))) + 1) * len(active)

	ap.UpdateGabors(gparams)
	gparams.GborOutput.SetShape([]int{ap.Sound.Channels(), sy, sx}, nil, []string{"chan", "freq", "time"})
	gparams.ExtGi.SetShape([]int{sy, gparams.GaborSet.Filters.Dim(0)}, nil, nil) // passed in for each channel
	gparams.GborOutput.SetMetaData("odd-row", "true")
	gparams.GborOutput.SetMetaData("grid-fill", ".9")
	gparams.GborKwta.CopyShapeFrom(&gparams.GborOutput)
	gparams.GborKwta.CopyMetaData(&gparams.GborOutput)

	for ch := int(0); ch < ap.Sound.Channels(); ch++ {
		agabor.Convolve(ch, &pparams.MelFBankSegment, gparams.GaborSet, &gparams.GborOutput, ap.ByTime)
		//if ap.NeighInhib.On {
		//	ap.NeighInhib.Inhib4(&gparams.GborOutput, &ap.ExtGi)
		//} else {
		//	ap.ExtGi.SetZeros()
		//}

		if gparams.Kwta.On {
			ap.ApplyKwta(ch, gparams)
			//tsr = &gparams.GborKwta
			//} else {
			//	tsr = &gparams.GborOutput
		}
	}
	//return tsr
}

// ApplyKwta runs the kwta algorithm on the raw activations
func (ap *App) ApplyKwta(ch int, gparams *GaborParams) {
	gparams.GborKwta.CopyFrom(&gparams.GborOutput)
	if gparams.Kwta.On {
		rawSS := gparams.GborOutput.SubSpace([]int{ch}).(*etensor.Float32)
		kwtaSS := gparams.GborKwta.SubSpace([]int{ch}).(*etensor.Float32)
		// This app is 2D only - no pools
		//if ap.KwtaPool == true {
		//	ap.Kwta.KWTAPool(rawSS, kwtaSS, &ap.Inhibs, &ap.ExtGi)
		//} else {
		gparams.Kwta.KWTALayer(rawSS, kwtaSS, &gparams.ExtGi)
		//}
	}
}

// ProcessSetup grabs params from the selected sounds table row and sets params for the actual processing step
func (ap *App) ProcessSetup() error {
	if ap.SndsTable.Table.Rows == 0 {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Sounds table empty", Prompt: "Open a sound file before processing"}, gi.AddOk, gi.NoCancel, nil, nil)
		return errors.New("Load a sound file and try again")
	}
	ap.Row = ap.SndsTable.View.SelectedIdx
	idx := ap.SndsTable.View.Table.Idxs[ap.Row]
	ap.Params.SegmentStart = float32(ap.SndsTable.Table.CellFloat("Start", idx))
	ap.Params.SegmentEnd = float32(ap.SndsTable.Table.CellFloat("End", idx))
	d := ap.SndsTable.Table.CellString("Dir", idx)
	f := ap.SndsTable.Table.CellString("File", idx)
	id := d + "/" + f
	for _, s := range ap.Sequence {
		if strings.Contains(s.File, id) {
			ap.SndFile = s.File
		}
	}
	return nil
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
		{"File", etensor.STRING, nil, nil},
		{"Dir", etensor.STRING, nil, nil},
	}
	ap.SndsTable.Table.SetFromSchema(sch, 0)
}

// ConfigTableView configures given tableview
func (ap *App) ConfigTableView(tv *etview.TableView) {
	tv.SetProp("inactive", true)
	tv.SetInactive()
	//tv.SliceViewSig.Connect(ap.GUI.ViewPort, func(recv, send ki.Ki, sig int64, data interface{}) {
	//if sig == int64(giv.SliceViewDoubleClicked) {
	//	ap.ProcessSetup()
	//	ap.Sound.Load(ap.SndFile)
	//	ap.GUI.UpdateWindow()
	//}
	//})
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
		Active:  egui.ActiveAlways,
		Func: func() {
			exts := ".wav"
			giv.FileViewDialog(ap.GUI.ViewPort, ap.SndPath, exts, giv.DlgOpts{Title: "Open .wav Sound File", Prompt: "Open a .wav file to load for sound processing."}, nil,
				ap.GUI.Win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
					if sig == int64(gi.DialogAccepted) {
						dlg, _ := send.Embed(gi.KiT_Dialog).(*gi.Dialog)
						fn := giv.FileViewDialogValue(dlg)
						info, err := os.Stat(fn)
						if err != nil {
							fmt.Println("error stating %s", fn)
							return
						}
						if info.IsDir() {
							filepath.Walk(fn, func(path string, info os.FileInfo, err error) error {
								if err != nil {
									log.Fatalf(err.Error())
								}
								fmt.Printf("File Name: %s\n", info.Name())
								fp := filepath.Join(fn, info.Name())
								ap.LoadTranscription(fp)
								return nil
							})
						} else {
							ap.LoadTranscription(fn)
						}
						ap.ConfigTableView(ap.SndsTable.View)
						ap.GUI.IsRunning = true
						ap.GUI.ToolBar.UpdateActions()
						ap.GUI.Win.UpdateSig()
					}
				})
		},
	})

	ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Unload Sounds",
		Icon:    "file-close",
		Tooltip: "Clears the table of sounds and closes the open sound files",
		Active:  egui.ActiveRunning,
		Func: func() {
			ap.SndsTable.Table.SetNumRows(0)
			ap.SndsTable.View.UpdateTable()
			ap.GUI.IsRunning = false
			ap.GUI.ToolBar.UpdateActions()
		},
	})

	ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Process 1", Icon: "play",
		Tooltip: "Process the segment of audio from SegmentStart to SegmentEnd applying the gabor filters to the Mel tensor",
		Active:  egui.ActiveRunning,
		Func: func() {
			ap.GUI.ToolBar.UpdateActions()
			err := ap.ProcessSetup()
			if err == nil {
				err = ap.Process(&ap.PParams1, &ap.GParams1)
				if err == nil {
					ap.ApplyGabor(&ap.PParams1, &ap.GParams1)
					ap.GUI.UpdateWindow()
				}
			}
		},
	})

	ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Process 2", Icon: "play",
		Tooltip: "Process the segment of audio from SegmentStart to SegmentEnd applying the gabor filters to the Mel tensor",
		Active:  egui.ActiveRunning,
		Func: func() {
			err := ap.ProcessSetup()
			if err == nil {
				err = ap.Process(&ap.PParams2, &ap.GParams2)
				if err == nil {
					ap.ApplyGabor(&ap.PParams2, &ap.GParams2)
					ap.GUI.UpdateWindow()
				}
			}
		},
	})

	ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Next 1", Icon: "fast-fwd",
		Tooltip: "Process the next segment of audio",
		Active:  egui.ActiveRunning,
		Func: func() {
			// setup the next segment of sound
			ap.SndsTable.View.ResetSelectedIdxs()
			if ap.Row == ap.SndsTable.View.DispRows-1 {
				ap.Row = 0
			} else {
				ap.Row += 1
			}
			ap.SndsTable.View.SelectedIdx = ap.Row
			ap.SndsTable.View.SelectIdx(ap.Row)
			err := ap.ProcessSetup()
			if err == nil {
				err = ap.Process(&ap.PParams1, &ap.GParams1)
				if err == nil {
					ap.ApplyGabor(&ap.PParams1, &ap.GParams1)
					ap.GUI.UpdateWindow()
				}
			}
		},
	})

	ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Next 2", Icon: "fast-fwd",
		Tooltip: "Process the next segment of audio",
		Active:  egui.ActiveRunning,
		Func: func() {
			// setup the next segment of sound
			ap.SndsTable.View.ResetSelectedIdxs()
			if ap.Row == ap.SndsTable.View.DispRows-1 {
				ap.Row = 0
			} else {
				ap.Row += 1
			}
			ap.SndsTable.View.SelectedIdx = ap.Row
			ap.SndsTable.View.SelectIdx(ap.Row)
			err := ap.ProcessSetup()
			if err == nil {
				err = ap.Process(&ap.PParams2, &ap.GParams2)
				if err == nil {
					ap.ApplyGabor(&ap.PParams2, &ap.GParams2)
					ap.GUI.UpdateWindow()
				}
			}
		},
	})

	ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Update Gabors", Icon: "update",
		Tooltip: "Call this to see the result of changing the Gabor specifications",
		Active:  egui.ActiveAlways,
		Func: func() {
			ap.UpdateGabors(&ap.GParams1)
			ap.UpdateGabors(&ap.GParams2)
			ap.GUI.UpdateWindow()
		},
	})

	ap.GUI.ToolBar.AddSeparator("filt")

	ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Filter sounds...", Icon: "search",
		Tooltip: "filter the table of sounds for sounds containing string...",
		Active:  egui.ActiveRunning,
		Func: func() {
			giv.CallMethod(ap, "FilterSounds", ap.GUI.ViewPort)
		},
	})

	ap.GUI.AddToolbarItem(egui.ToolbarItem{Label: "Unilter sounds...", Icon: "reset",
		Tooltip: "clear sounds table filter",
		Active:  egui.ActiveRunning,
		Func: func() {
			giv.CallMethod(ap, "UnfilterSounds", ap.GUI.ViewPort)
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

	split1.SetSplits(.75, .25)

	ap.GUI.StructView = giv.AddNewStructView(split, "app")
	ap.GUI.StructView.SetStruct(ap)

	specs := giv.AddNewTableView(split, "specs1")
	specs.Viewport = ap.GUI.ViewPort
	specs.SetSlice(&ap.GParams1.GaborSpecs)

	specs = giv.AddNewTableView(split, "specs2")
	specs.Viewport = ap.GUI.ViewPort
	specs.SetSlice(&ap.GParams2.GaborSpecs)

	tv := gi.AddNewTabView(split, "tv")

	tg := tv.AddNewTab(etview.KiT_TensorGrid, "Gabors").(*etview.TensorGrid)
	tg.SetStretchMax()
	//ap.PParams1.LogPowerSegment.SetMetaData("grid-min", "10")
	tg.SetTensor(&ap.GParams1.GaborSet.Filters)

	tg = tv.AddNewTab(etview.KiT_TensorGrid, "Power").(*etview.TensorGrid)
	tg.SetStretchMax()
	ap.PParams1.LogPowerSegment.SetMetaData("grid-min", "10")
	tg.SetTensor(&ap.PParams1.LogPowerSegment)

	tg = tv.AddNewTab(etview.KiT_TensorGrid, "Mel").(*etview.TensorGrid)
	tg.SetStretchMax()
	tg.SetTensor(&ap.PParams1.MelFBankSegment)

	tg = tv.AddNewTab(etview.KiT_TensorGrid, "Result").(*etview.TensorGrid)
	tg.SetStretchMax()
	tg.SetTensor(&ap.GParams1.GborOutput)

	tv2 := gi.AddNewTabView(split, "tv2")
	split.SetSplits(.2, .2, .2, .2, .2)

	tg = tv2.AddNewTab(etview.KiT_TensorGrid, "Gabors").(*etview.TensorGrid)
	tg.SetStretchMax()
	tg.SetTensor(&ap.GParams2.GaborSet.Filters)

	tg = tv2.AddNewTab(etview.KiT_TensorGrid, "Power").(*etview.TensorGrid)
	tg.SetStretchMax()
	ap.PParams2.LogPowerSegment.SetMetaData("grid-min", "10")
	tg.SetTensor(&ap.PParams2.LogPowerSegment)

	tg = tv2.AddNewTab(etview.KiT_TensorGrid, "Mel").(*etview.TensorGrid)
	tg.SetStretchMax()
	tg.SetTensor(&ap.PParams2.MelFBankSegment)

	tg = tv2.AddNewTab(etview.KiT_TensorGrid, "Result").(*etview.TensorGrid)
	tg.SetStretchMax()
	tg.SetTensor(&ap.GParams2.GborOutput)

	ap.GUI.FinalizeGUI(false)
	return ap.GUI.Win
}

// CmdArgs
func (ap *App) CmdArgs() {

}

var AppProps = ki.Props{
	"CallMethods": ki.PropSlice{
		{"FilterSounds", ki.Props{
			"desc": "Filter sounds table...",
			"Args": ki.PropSlice{
				{"sound", ki.Props{
					"width": 60,
				}},
			},
		}},
		{"UnfilterSounds", ki.Props{
			"desc": "Unfilter sounds table...",
		}},
	},
}
