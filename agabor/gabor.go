package agabor

import (
	"fmt"
	"image"

	"github.com/chewxy/math32"
	"github.com/emer/auditory/input"
	"github.com/emer/etable/etensor"
)

// Gabor params for auditory gabor filters: 2d Gaussian envelope times a sinusoidal plane wave --
// by default produces 2 phase asymmetric edge detector filters -- horizontal tuning is different from V1 version --
// has elongated frequency-band specific tuning, not a parallel horizontal tuning -- and has multiple of these
type Gabor struct {
	On              bool            `desc:"use this gabor filtering of the time-frequency space filtered input (time in terms of steps of the DFT transform, and discrete frequency factors based on the FFT window and input sample rate)"`
	SizeTime        int             `viewif:"On" def:"6,8,12,16,24" desc:" size of the filter in the time (horizontal) domain, in terms of steps of the underlying DFT filtering steps"`
	SizeFreq        int             `viewif:"On" def:"6,8,12,16,24" desc:" size of the filter in the frequency domain, in terms of discrete frequency factors based on the FFT window and input sample rate"`
	SpaceTime       int             `viewif:"On" desc:" spacing in the time (horizontal) domain, in terms of steps"`
	SpaceFreq       int             `viewif:"On" desc:" spacing in the frequency (vertical) domain"`
	WaveLen         float32         `viewif:"On" def:"1.5,2" desc:"wavelength of the sine waves in normalized units"`
	SigmaLen        float32         `viewif:"On" def:"0.6" desc:"gaussian sigma for the length dimension (elongated axis perpendicular to the sine waves) -- normalized as a function of filter size in relevant dimension"`
	SigmaWidth      float32         `viewif:"On" def:"0.3" desc:"gaussian sigma for the width dimension (in the direction of the sine waves) -- normalized as a function of filter size in relevant dimension"`
	HorizSigmaLen   float32         `viewif:"On" def:"0.3" desc:"gaussian sigma for the length of special horizontal narrow-band filters -- normalized as a function of filter size in relevant dimension"`
	HorizSigmaWidth float32         `viewif:"On" def:"0.1" desc:"gaussian sigma for the horizontal dimension for special horizontal narrow-band filters -- normalized as a function of filter size in relevant dimension"`
	Gain            float32         `viewif:"On" def:"2" desc:"overall gain multiplier applied after gabor filtering -- only relevant if not using renormalization (otherwize it just gets renormed awaY"`
	NHoriz          int             `viewif:"On" def:"4" desc:"number of horizontally-elongated,  pure time-domain, frequency-band specific filters to include, evenly spaced over the available frequency space for this filter set -- in addition to these, there are two diagonals (45, 135) and a vertically-elongated (wide frequency band) filter"`
	PhaseOffset     float32         `viewif:"On" def:"0,1.5708" desc:"offset for the sine phase -- default is an asymmetric sine wave -- can make it into a symmetric cosine gabor by using PI/2 = 1.5708"`
	CircleEdge      bool            `viewif:"On" def:"true" desc:"cut off the filter (to zero) outside a circle of diameter filter_size -- makes the filter more radially symmetric"`
	NFilters        int             `viewif:"On" desc:" total number of filters = 3 + NHoriz"`
	Shape           image.Point     `viewif:Gabor1.On=true" inactive:"+" desc:"overall geometry of gabor1 output (group-level geometry -- feature / unit level geometry is n_features, 2)"`
	Filters         etensor.Float32 `viewif:Gabor1.On=true" desc:"full gabor filters"`
}

//Initialize initializes the Gabor
func (ga *Gabor) Initialize(steps int, melFilters int) {
	ga.On = true
	ga.Gain = 2.0
	ga.NHoriz = 4
	ga.SizeTime = 6.0
	ga.SizeFreq = 6.0
	ga.SpaceTime = 2.0
	ga.SpaceFreq = 2.0
	ga.WaveLen = 2.0
	ga.SigmaLen = 0.6
	ga.SigmaWidth = 0.3
	ga.HorizSigmaLen = 0.3
	ga.HorizSigmaWidth = 0.1
	ga.PhaseOffset = 0.0
	ga.CircleEdge = true
	ga.NFilters = 3 + ga.NHoriz // 3 is number of angle filters

	ga.SetShape(steps, melFilters)
	ga.InitFilters()

}

// SetShape sets the shape of a gabor based on parameters of gabor, mel filters and input
func (ga *Gabor) SetShape(trialSteps int, nFilters int) { // nFilters is Mel.MelFBank.NFilters
	ga.Shape.X = ((trialSteps - 1) / ga.SpaceTime) + 1
	ga.Shape.Y = ((nFilters - ga.SizeFreq - 1) / ga.SpaceFreq) + 1
}

// InitFilters
func (ga *Gabor) InitFilters() {
	ga.Filters.SetShape([]int{ga.NFilters, ga.SizeFreq, ga.SizeTime}, nil, nil)
	ga.RenderFilters(&ga.Filters)
}

// RenderFilters generates filters into the given matrix, which is formatted as: [ga.SizeTime_steps][ga.SizeFreq][n_filters]
func (ga *Gabor) RenderFilters(filters *etensor.Float32) {
	ctrTime := (float32(ga.SizeTime) - 1) / 2.0
	ctrFreq := (float32(ga.SizeFreq) - 1) / 2.0
	angInc := math32.Pi / 4.0
	radiusTime := float32(ga.SizeTime / 2.0)
	radiusFreq := float32(ga.SizeFreq / 2.0)

	lenNorm := 1.0 / (2.0 * ga.SigmaLen * ga.SigmaLen)
	widthNorm := 1.0 / (2.0 * ga.SigmaWidth * ga.SigmaWidth)
	lenHorizNorm := 1.0 / (2.0 * ga.HorizSigmaLen * ga.HorizSigmaLen)
	widthHorizNorm := 1.0 / (2.0 * ga.HorizSigmaWidth * ga.HorizSigmaWidth)

	twoPiNorm := (2.0 * math32.Pi) / ga.WaveLen
	hCtrInc := (ga.SizeFreq - 1) / (ga.NHoriz + 1)

	fli := 0
	for hi := 0; hi < ga.NHoriz; hi, fli = hi+1, fli+1 {
		hCtrFreq := hCtrInc * (hi + 1)
		angF := -2.0 * angInc
		for y := 0; y < ga.SizeFreq; y++ {
			var xf, yf, xfn, yfn float32
			for x := 0; x < ga.SizeTime; x++ {
				xf = float32(x) - ctrTime
				yf = float32(y) - float32(hCtrFreq)
				xfn = xf / radiusTime
				yfn = yf / radiusFreq

				dist := math32.Hypot(xfn, yfn)
				val := float32(0)
				if !(ga.CircleEdge && dist > 1.0) {
					nx := xfn*math32.Cos(angF) - yfn*math32.Sin(angF)
					ny := yfn*math32.Cos(angF) + xfn*math32.Sin(angF)
					gauss := math32.Exp(-(widthHorizNorm*(nx*nx) + lenHorizNorm*(ny*ny)))
					sinVal := math32.Sin(twoPiNorm*ny + ga.PhaseOffset)
					val = gauss * sinVal
				}
				filters.Set([]int{fli, x, y}, val)
			}
		}
	}

	// fli should be ga.Horiz - 1 at this point
	for ang := 1; ang < 4; ang, fli = ang+1, fli+1 {
		angF := float32(-ang) * angInc
		var xf, yf, xfn, yfn float32
		for y := 0; y < ga.SizeFreq; y++ {
			for x := 0; x < ga.SizeTime; x++ {
				xf = float32(x) - ctrTime
				yf = float32(y) - ctrFreq
				xfn = xf / radiusTime
				yfn = yf / radiusFreq

				dist := math32.Hypot(xfn, yfn)
				val := float32(0)
				if !(ga.CircleEdge && dist > 1.0) {
					nx := xfn*math32.Cos(angF) - yfn*math32.Sin(angF)
					ny := yfn*math32.Cos(angF) + xfn*math32.Sin(angF)
					gauss := math32.Exp(-(lenNorm*(nx*nx) + widthNorm*(ny*ny)))
					sinVal := math32.Sin(twoPiNorm*ny + ga.PhaseOffset)
					val = gauss * sinVal
				}
				filters.Set([]int{fli, y, x}, val)
			}
		}
	}

	// renorm each half
	for fli := 0; fli < ga.NFilters; fli++ {
		posSum := float32(0)
		negSum := float32(0)
		for y := 0; y < ga.SizeFreq; y++ {
			for x := 0; x < ga.SizeTime; x++ {
				val := float32(filters.Value([]int{fli, y, x}))
				if val > 0 {
					posSum += val
				} else if val < 0 {
					negSum += val
				}
			}
		}
		posNorm := 1.0 / posSum
		negNorm := -1.0 / negSum
		for y := 0; y < ga.SizeFreq; y++ {
			for x := 0; x < ga.SizeTime; x++ {
				val := filters.Value([]int{fli, y, x})
				if val > 0.0 {
					val *= posNorm
				} else if val < 0.0 {
					val *= negNorm
				}
				filters.Set([]int{fli, y, x}, val)
			}
		}
	}
}

// Conv processes input using filters that operate over an entire trial of samples
func Conv(ch int, spec Gabor, input input.Input, raw *etensor.Float32, filters int, melData etensor.Float32) {
	tHalfSz := spec.SizeTime / 2
	tOff := tHalfSz - input.BorderSteps
	tMin := tOff
	if tMin < 0 {
		tMin = 0
	}
	tMax := input.TrialSteps - tMin

	fMin := int(0)
	//fMax := ap.Mel.MelFBank.NFilters - spec.SizeFreq
	fMax := filters - spec.SizeFreq

	maxTimeIdx := raw.Dim(2) // dim 0 is channel
	tIdx := 0
	for s := tMin; s < tMax; s, tIdx = s+spec.SpaceTime, tIdx+1 {
		inSt := s - tOff
		if tIdx > maxTimeIdx {
			fmt.Printf("GaborFilter: time index %v out of range: %v", tIdx, maxTimeIdx)
			break
		}

		maxFreqIdx := raw.Dim(1) // dim 0 is channel
		fIdx := 0
		for flt := fMin; flt < fMax; flt, fIdx = flt+spec.SpaceFreq, fIdx+1 {
			if fIdx > maxFreqIdx {
				fmt.Printf("GaborFilter: freq index %v out of range: %v", tIdx, maxFreqIdx)
				break
			}
			nf := spec.NFilters
			for fi := int(0); fi < nf; fi++ {
				fSum := float32(0.0)
				for ff := int(0); ff < spec.SizeFreq; ff++ {
					for ft := int(0); ft < spec.SizeTime; ft++ {
						fVal := spec.Filters.Value([]int{fi, ff, ft})
						iVal := melData.Value([]int{inSt + ft, flt + ff, ch})
						fSum += fVal * iVal
					}
				}
				pos := fSum >= 0.0
				act := spec.Gain * math32.Abs(fSum)
				if pos {
					raw.SetFloat([]int{ch, fIdx, tIdx, 0, fi}, float64(act))
					raw.SetFloat([]int{ch, fIdx, tIdx, 1, fi}, 0)
				} else {
					raw.SetFloat([]int{ch, fIdx, tIdx, 0, fi}, 0)
					raw.SetFloat([]int{ch, fIdx, tIdx, 1, fi}, float64(act))
				}
			}
		}
	}
}
