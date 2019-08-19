package audio

import (
	"fmt"
	"math"

	"github.com/chewxy/math32"
)

// Input defines the sound input parameters for auditory processing
type Input struct {
	WinMsec      float32 `def:"25" desc:"input window -- number of milliseconds worth of sound to filter at a time"`
	StepMsec     float32 `def:"5,10,12.5" desc:"input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample"`
	TrialMsec    float32 `def:"100" desc:"length of a full trial's worth of input -- total number of milliseconds to accumulate into a complete trial of activations to present to a network -- must be a multiple of step_msec -- input will be trial_msec / step_msec = trial_steps wide in the X axis, and number of filters in the Y axis"`
	SampleRate   int     `desc:"rate of sampling in our sound input (e.g., 16000 = 16Khz) -- can initialize this from a taSound object using InitFromSound method"`
	BorderSteps  int     `desc:"number of steps before and after the trial window to preserve -- this is important when applying temporal filters that have greater temporal extent"`
	Channels     int     `desc:"total number of channels to process"`
	Channel      int     `viewif:"Channels=1" desc:"specific channel to process, if input has multiple channels, and we only process one of them (-1 = process all)"`
	WinSamples   int     `inactive:"+" desc:"total number of samples to process (win_msec * .001 * sample_rate)"`
	StepSamples  int     `inactive:"+" desc:"total number of samples to step input by (step_msec * .001 * sample_rate)"`
	TrialSamples int     `inactive:"+" desc:"total number of samples in a trial  (trail_msec * .001 * sample_rate)"`
	TrialSteps   int     `inactive:"+" desc:"total number of steps in a trial  (trail_msec / step_msec)"`
	TotalSteps   int     `inactive:"+" desc:"2*border_steps + trial_steps -- total in full window"`
}

//Defaults initializes the Input
func (ais *Input) Defaults() {
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
func (ais *Input) ComputeSamples() {
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
