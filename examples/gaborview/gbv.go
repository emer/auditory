package main

import (
	"fmt"
	"github.com/emer/auditory/agabor"
	"github.com/emer/auditory/mel"
	"github.com/emer/auditory/sound"
	"github.com/emer/auditory/speech"
	"github.com/emer/auditory/speech/grafestes"
	"github.com/emer/auditory/speech/synthcvs"
	"github.com/emer/auditory/speech/timit"
	"github.com/emer/emergent/egui"
	"github.com/emer/emergent/evec"
	"github.com/emer/etable/etensor"
	"github.com/emer/etable/etview"
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
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

type App struct {
	// the environment has the training/test data and the procedures for creating/choosing the input to the model
	// "Segment" in var name indicates that the data or value only applies to a segment of samples rather than the entire signal
	Nm          string `view:"-" desc:"name of this environment"`
	Dsc         string `view:"-" desc:"description of this environment"`
	Corpus      string `desc:"the set of sound files"`
	InitDir     string `desc:"initial directory for file open dialog"`
	SndFile     string `view:"-" desc:"full path of open sound file"`
	SndFileName string `desc:"name of the open sound file"`
	SndFilePath string `desc:"path to the open sound file"`

	Snd sound.SndEnv `view:"+" desc:" sound processing values and matrices for the short duration pathway"`
	// shortcuts to structs so you don't need to open a hierarchy of windows
	SndParams  sound.Params     `desc:"audio processing params"`
	MelParams  mel.Params       `desc:"audio processing params"`
	GaborSpecs []agabor.Filter  `desc:"individual gabor specifications, orientation, phase, sigma, wavelength"`
	GaborSet   agabor.FilterSet `desc:"gabor params that apply to all gabors, e.g. size and stride"`
	Gabors     etensor.Float64  `desc:"visualization of the gabors"`

	OutSize   evec.Vec2i `desc:"the output tensor geometry -- must be >= number of cats"`
	SpeechSeq speech.SpeechSequence
	CurUnits  []string `desc:"names of units in the current audio segment"`

	// internal state - view:"-"
	ToolBar      *gi.ToolBar `view:"-" desc:" the master toolbar"`
	MoreSegments bool        `view:"-" desc:" are there more samples to process"`
	SndIdx       int         `view:"-" desc:"the index into the soundlist of the sound from last trial"`

	GUI          egui.GUI           `view:"-" desc:"manages all the gui elements"`
	PowerGrid    *etview.TensorGrid `view:"-" desc:"power grid view for the current segment"`
	MelFBankGrid *etview.TensorGrid `view:"-" desc:"melfbank grid view for the current segment"`
}

// Init
func (ap *App) Init() {

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

func (ap *App) InitSnd() {
	ap.MoreSegments = true
	ap.Snd.Defaults()
	ap.Snd.Nm = ""
	ap.Snd.Dsc = "Input/processing params"
	ap.Snd.Kwta.On = true
	ap.Snd.KwtaPool = false // false means use KwtaLayer

	ap.Snd.GborOutPoolsX = 0
	ap.Snd.GborOutPoolsY = 0
	ap.Snd.GborOutUnitsY = 26
	ap.Snd.GborOutUnitsX = 16

	// override defaults
	ap.Snd.Params.SegmentMs = 90
	ap.Snd.Params.WinMs = 25
	ap.Snd.Params.StepMs = 90
	ap.Snd.Params.StrideMs = 10
	ap.Snd.Params.BorderSteps = 0

	// for example, with Stride/StepMs equal to 10 and 3 border steps on either side there will be 16 values for the gabor stepping to cover
	// so the gbor (size of 6) will go from 0-5, 2-7, 4-9 ... 10-15

	// these overrides must follow Mel.Defaults
	if ap.Corpus == "TIMIT" {
		ap.Snd.Mel.FBank.RenormMin = -6
		ap.Snd.Mel.FBank.RenormMax = 4
	} else {
		ap.Snd.Mel.FBank.RenormMin = 2
		ap.Snd.Mel.FBank.RenormMax = 9
	}
	ap.Snd.Mel.FBank.LoHz = 20
	ap.Snd.Mel.FBank.HiHz = 8000
	ap.Snd.Mel.FBank.NFilters = 45

	ap.Snd.GaborFilters.SizeX = 9
	ap.Snd.GaborFilters.SizeY = 9
	ap.Snd.GaborFilters.Gain = 1.5
	ap.Snd.GaborFilters.StrideX = 1
	ap.Snd.GaborFilters.StrideY = 3
	ap.Snd.GaborFilters.Distribute = false

	orient := []float32{0, 45, 90, 135}
	wavelen := []float32{1.0, 2.0}
	phase := []float32{0}
	sigma := []float32{.25, .5}

	ap.Snd.GaborSpecs = nil // in case there are some specs already

	for _, or := range orient {
		for _, wv := range wavelen {
			for _, ph := range phase {
				for _, wl := range sigma {
					spec := agabor.Filter{WaveLen: wv, Orientation: or, SigmaWidth: wl, SigmaLength: wl, PhaseOffset: ph, CircleEdge: true}
					ap.Snd.GaborSpecs = append(ap.Snd.GaborSpecs, spec)
				}
			}
		}
	}

	s := ap.Snd
	ap.SndParams = s.Params
	ap.MelParams = s.Mel
	ap.GaborSpecs = s.GaborSpecs
	ap.GaborSet = s.GaborFilters
	ap.Gabors = s.GaborFilters.Filters
}

// LoadSnd
func (ap *App) LoadSnd(path string) {
	ap.SndFile = path
	err := ap.Snd.Sound.Load(path)
	if err != nil {
		log.Printf("LoadSnd: error loading sound -- %v\n, err", ap.SpeechSeq.File)
		return
	}
	ap.Snd.LoadSound() // actually load the sound

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
	ap.InitSnd()

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
	} else if ap.Corpus == "SYNTHCVS" {
		snd, ok = synthcvs.SndFmIdx(idx, ap.SpeechSeq.ID)
	} else if ap.Corpus == "GRAFESTES" {
		snd, ok = grafestes.SndFmIdx(idx, ap.SpeechSeq.ID)
	} else {
		fmt.Println("SndFmIdx: fell through corpus ifelse ")
	}
	return
}

// ConfigGui configures the GoGi gui interface for this simulation,
func (ap *App) ConfigGui() *gi.Window {
	title := "Gabor View"
	ap.GUI.MakeWindow(ap, "gb", title, "Application/Utility to allow viewing of gabor convolution with sound")

	// sound rostral
	tg := ap.GUI.TabView.AddNewTab(etview.KiT_TensorGrid, "MelFBank").(*etview.TensorGrid)
	tg.SetStretchMax()
	ap.MelFBankGrid = tg
	tg.SetTensor(&ap.Snd.MelFBankSegment)

	// sound caudal
	tg = ap.GUI.TabView.AddNewTab(etview.KiT_TensorGrid, "Power").(*etview.TensorGrid)
	tg.SetStretchMax()
	ap.PowerGrid = tg
	ap.Snd.LogPowerSegment.SetMetaData("grid-min", "10")
	tg.SetTensor(&ap.Snd.LogPowerSegment)

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
			//giv.FileViewDialog(ap.GUI.ViewPort, string(ap.SndFile), exts, giv.DlgOpts{Title: "Open .wav Sound File", Prompt: "Open a .wav file to load for sound processing."}, nil,
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
						ap.Snd.Sound.Load(ap.SndFile)
						ap.Snd.Init(0, 0, 0) // for this app we don't remove or add any silence
						ap.Snd.ProcessSegment()
					}
				})
		},
	})

	ap.GUI.FinalizeGUI(false)
	return ap.GUI.Win
}

// These props register Save methods so they can be used
var AppProps = ki.Props{}

func (ap *App) CmdArgs() {

}
