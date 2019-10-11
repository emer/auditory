package input

import (
	"fmt"
	"math"

	"github.com/chewxy/math32"
	"github.com/emer/auditory/sound"
)

// Input defines the sound input parameters for auditory processing
type Input struct {
	WinMs          float32 `def:"25" desc:"input window -- number of milliseconds worth of sound to filter at a time"`
	StepMs         float32 `def:"5,10,12.5" desc:"input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample"`
	TrialMs        float32 `def:"100" desc:"length of a full trial's worth of input -- total number of milliseconds to accumulate into a complete trial of activations to present to a network -- must be a multiple of step_ms -- input will be TrialMs / StepMs = TrialSteps wide in the X axis, and number of filters in the Y axis"`
	SampleRate     int     `desc:"rate of sampling in our sound input (e.g., 16000 = 16Khz) -- can initialize this from a taSound object using InitFromSound method"`
	Channels       int     `desc:"total number of channels to process"`
	Channel        int     `viewif:"Channels=1" desc:"specific channel to process, if input has multiple channels, and we only process one of them (-1 = process all)"`
	WinSamples     int     `inactive:"+" desc:"number of samples to process each step"`
	StepSamples    int     `inactive:"+" desc:"number of samples to step input by"`
	TrialSamples   int     `inactive:"+" desc:"number of samples in a trial"`
	TrialSteps     int     `inactive:"+" desc:"number of steps in a trial"`
	TrialStepsPlus int     `inactive:"+" desc:"TrialSteps plus steps overlapping next trial or for padding if no next trial"`
	Steps          []int   `inactive:"+" desc:"pre-calculated start position for each step"`
}

//Defaults initializes the Input
func (ais *Input) Defaults() {
	ais.WinMs = 25.0
	ais.StepMs = 5.0
	ais.TrialMs = 100.0
	ais.SampleRate = 44100
	ais.Channels = 1
	ais.Channel = 0
}

// ComputeSamples computes the sample counts based on time and sample rate
// signal padded with zeros to ensure complete trials
func (ais *Input) Config(signalRaw []float32, padValue float32) (signalPadded []float32) {
	ais.WinSamples = MSecToSamples(ais.WinMs, ais.SampleRate)
	ais.StepSamples = MSecToSamples(ais.StepMs, ais.SampleRate)
	ais.TrialSamples = MSecToSamples(ais.TrialMs, ais.SampleRate)
	ais.TrialSteps = int(math.Round(float64(ais.TrialMs / ais.StepMs)))
	ais.TrialStepsPlus = ais.TrialSteps + int(math.Round(float64(ais.WinSamples/ais.StepSamples)))
	tail := len(signalRaw) % ais.TrialSamples
	padLen := ais.TrialStepsPlus*ais.StepSamples - tail
	padLen = padLen + ais.WinSamples
	pad := make([]float32, padLen)
	for i := range pad {
		pad[i] = padValue
	}
	signalPadded = append(signalRaw, pad...)
	ais.Steps = make([]int, ais.TrialStepsPlus)
	for i := 0; i < ais.TrialStepsPlus; i++ {
		ais.Steps[i] = ais.StepSamples * i
	}
	return signalPadded
}

// MSecToSamples converts milliseconds to samples, in terms of sample_rate
func MSecToSamples(ms float32, rate int) int {
	return int(math.Round(float64(ms) * 0.001 * float64(rate)))
}

// SamplesToMSec converts samples to milliseconds, in terms of sample_rate
func SamplesToMSec(samples int, rate int) float32 {
	return 1000.0 * float32(samples) / float32(rate)
}

// InitFromSound loads a sound and sets the Input channel vars and sample rate
func (in *Input) InitFromSound(snd *sound.Sound, nChannels int, channel int) {
	if snd == nil {
		fmt.Printf("InitFromSound: sound nil")
		return
	}
	in.SampleRate = int(snd.SampleRate())
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
