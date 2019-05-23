package audio

import (
	"github.com/chewxy/math32"
	"github.com/emer/etable/etensor"
)

// Gabor params for auditory gabor filters: 2d Gaussian envelope times a sinusoidal plane wave --
// by default produces 2 phase asymmetric edge detector filters -- horizontal tuning is different from V1 version --
// has elongated frequency-band specific tuning, not a parallel horizontal tuning -- and has multiple of these
type Gabor struct {
	On              bool    `desc:"use this gabor filtering of the time-frequency space filtered input (time in terms of steps of the DFT transform, and discrete frequency factors based on the FFT window and input sample rate)"`
	SizeTime        int     `viewif:"On" def:"6,8,12,16,24" desc:" #DEF_6;8;12;16;24 size of the filter in the time (horizontal) domain, in terms of steps of the underlying DFT filtering steps"`
	SizeFreq        int     `viewif:"On" def:"6,8,12,16,24" desc:" #DEF_6;8;12;16;24 size of the filter in the frequency domain, in terms of discrete frequency factors based on the FFT window and input sample rate"`
	SpaceTime       int     `viewif:"On" desc:" spacing in the time (horizontal) domain, in terms of steps"`
	SpaceFreq       int     `viewif:"On" desc:" spacing in the frequency (vertical) domain"`
	WaveLen         float32 `viewif:"On" def:"1.5,2" desc:"wavelength of the sine waves in normalized units"`
	SigmaLen        float32 `viewif:"On" def:"0.6" desc:"gaussian sigma for the length dimension (elongated axis perpendicular to the sine waves) -- normalized as a function of filter size in relevant dimension"`
	SigmaWidth      float32 `viewif:"On" def:"0.3" desc:"gaussian sigma for the width dimension (in the direction of the sine waves) -- normalized as a function of filter size in relevant dimension"`
	HorizSigmaLen   float32 `viewif:"On" def:"0.3" desc:"gaussian sigma for the length of special horizontal narrow-band filters -- normalized as a function of filter size in relevant dimension"`
	HorizSigmaWidth float32 `viewif:"On" def:"0.1" desc:"gaussian sigma for the horizontal dimension for special horizontal narrow-band filters -- normalized as a function of filter size in relevant dimension"`
	Gain            float32 `viewif:"On" def:"2" desc:"overall gain multiplier applied after gabor filtering -- only relevant if not using renormalization (otherwize it just gets renormed awaY"`
	NHoriz          int     `viewif:"On" def:"4" desc:"number of horizontally-elongated,  pure time-domain, frequency-band specific filters to include, evenly spaced over the available frequency space for this filter set -- in addition to these, there are two diagonals (45, 135) and a vertically-elongated (wide frequency band) filter"`
	PhaseOffset     float32 `viewif:"On" def:"0,1.5708" desc:"offset for the sine phase -- default is an asymmetric sine wave -- can make it into a symmetric cosine gabor by using PI/2 = 1.5708"`
	CircleEdge      bool    `viewif:"On" def:"true" desc:"cut off the filter (to zero) outside a circle of diameter filter_size -- makes the filter more radially symmetric"`
	NFilters        int     `viewif:"On" desc:" #READ_ONLY total number of filters = 3 + n_horiz"`
}

//Initialize initializes the Gabor
func (ag *Gabor) Initialize() {
	ag.On = true
	ag.Gain = 2.0
	ag.NHoriz = 4
	ag.SizeTime = 6.0
	ag.SizeFreq = 6.0
	ag.SpaceTime = 2.0
	ag.SpaceFreq = 2.0
	ag.WaveLen = 2.0
	ag.SigmaLen = 0.6
	ag.SigmaWidth = 0.3
	ag.HorizSigmaLen = 0.3
	ag.HorizSigmaWidth = 0.1
	ag.PhaseOffset = 0.0
	ag.CircleEdge = true
	ag.NFilters = 3 + ag.NHoriz // 3 is number of angle filters
}

// RenderFilters generates filters into the given matrix, which is formatted as: [ag.SizeTime_steps][ag.SizeFreq][n_filters]
func (ag *Gabor) RenderFilters(filters etensor.Float32) {
	ctrTime := (float32(ag.SizeTime) - 1) / 2.0
	ctrFreq := (float32(ag.SizeFreq) - 1) / 2.0
	angInc := math32.Pi / 4.0
	radiusTime := float32(ag.SizeTime / 2.0)
	radiusFreq := float32(ag.SizeFreq / 2.0)

	lenNorm := 1.0 / (2.0 * ag.SigmaLen * ag.SigmaLen)
	widthNorm := 1.0 / (2.0 * ag.SigmaWidth * ag.SigmaWidth)
	lenHorizNorm := 1.0 / (2.0 * ag.HorizSigmaLen * ag.HorizSigmaLen)
	widthHorizNorm := 1.0 / (2.0 * ag.HorizSigmaWidth * ag.HorizSigmaWidth)

	twoPiNorm := (2.0 * math32.Pi) / ag.WaveLen
	hCtrInc := (ag.SizeFreq - 1) / (ag.NHoriz + 1)

	fli := 0
	for hi := 0; hi < ag.NHoriz; hi, fli = hi+1, fli+1 {
		hCtrFreq := hCtrInc * (hi + 1)
		angF := -2.0 * angInc
		for y := 0; y < ag.SizeFreq; y++ {
			var xf, yf, xfn, yfn float32
			for x := 0; x < ag.SizeTime; x++ {
				xf = float32(x) - ctrTime
				yf = float32(y) - float32(hCtrFreq)
				xfn = xf / radiusTime
				yfn = yf / radiusFreq

				dist := math32.Hypot(xfn, yfn)
				val := float32(0)
				if !(ag.CircleEdge && dist > 1.0) {
					nx := xfn*math32.Cos(angF) - yfn*math32.Sin(angF)
					ny := yfn*math32.Cos(angF) + xfn*math32.Sin(angF)
					gauss := math32.Exp(-(widthHorizNorm*(nx*nx) + lenHorizNorm*(ny*ny)))
					sinVal := math32.Sin(twoPiNorm*ny + ag.PhaseOffset)
					val = gauss * sinVal
				}
				filters.Set([]int{x, y, fli}, val)
			}
		}
	}

	// fli should be ag.Horiz - 1 at this point
	for ang := 1; ang < 4; ang, fli = ang+1, fli+1 {
		angF := float32(-ang) * angInc
		var xf, yf, xfn, yfn float32
		for y := 0; y < ag.SizeFreq; y++ {
			for x := 0; x < ag.SizeTime; x++ {
				xf = float32(x) - ctrTime
				yf = float32(y) - ctrFreq
				xfn = xf / radiusTime
				yfn = yf / radiusFreq

				dist := math32.Hypot(xfn, yfn)
				val := float32(0)
				if !(ag.CircleEdge && dist > 1.0) {
					nx := xfn*math32.Cos(angF) - yfn*math32.Sin(angF)
					ny := yfn*math32.Cos(angF) + xfn*math32.Sin(angF)
					gauss := math32.Exp(-(lenNorm*(nx*nx) + widthNorm*(ny*ny)))
					sinVal := math32.Sin(twoPiNorm*ny + ag.PhaseOffset)
					val = gauss * sinVal
				}
				filters.Set([]int{x, y, fli}, val)
			}
		}
	}

	// renorm each half
	for fli := 0; fli < ag.NFilters; fli++ {
		posSum := float32(0)
		negSum := float32(0)
		for y := 0; y < ag.SizeFreq; y++ {
			for x := 0; x < ag.SizeTime; x++ {
				val := float32(filters.Value([]int{x, y, fli}))
				if val > 0 {
					posSum += val
				} else if val < 0 {
					negSum += val
				}
			}
		}
		posNorm := 1.0 / posSum
		negNorm := -1.0 / negSum
		for y := 0; y < ag.SizeFreq; y++ {
			for x := 0; x < ag.SizeTime; x++ {
				val := filters.Value([]int{x, y, fli})
				if val > 0.0 {
					val *= posNorm
				} else if val < 0.0 {
					val *= negNorm
				}
				filters.Set([]int{x, y, fli}, val)
			}
		}
	}
}
