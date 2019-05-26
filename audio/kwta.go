package audio

import (
	"fmt"
	"github.com/chewxy/math32"

	"github.com/emer/etable/etensor"
	"github.com/emer/etable/minmax"
)

//////////////////////////////////////////////////////////////////////////////////////
//  From emer.leabra.Act

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

// XX1Params are the X/(X+1) rate-coded activation function parameters for leabra
// using the GeLin (g_e linear) rate coded activation function
type XX1Params struct {
	Thr          float32 `def:"0.5" desc:"threshold value Theta (Q) for firing output activation (.5 is more accurate value based on AdEx biological parameters and normalization"`
	Gain         float32 `def:"80,100,40,20" min:"0" desc:"gain (gamma) of the rate-coded activation functions -- 100 is default, 80 works better for larger models, and 20 is closer to the actual spiking behavior of the AdEx model -- use lower values for more graded signals, generally in lower input/sensory layers of the network"`
	NVar         float32 `def:"0.005,0.01" min:"0" desc:"variance of the Gaussian noise kernel for convolving with XX1 in NOISY_XX1 and NOISY_LINEAR -- determines the level of curvature of the activation function near the threshold -- increase for more graded responding there -- note that this is not actual stochastic noise, just constant convolved gaussian smoothness to the activation function"`
	VmActThr     float32 `def:"0.01" desc:"threshold on activation below which the direct vm - act.thr is used -- this should be low -- once it gets active should use net - g_e_thr ge-linear dynamics (gelin)"`
	SigMult      float32 `def:"0.33" view:"-" desc:"multiplier on sigmoid used for computing values for net < thr"`
	SigMultPow   float32 `def:"0.8" view:"-" desc:"power for computing sig_mult_eff as function of gain * nvar"`
	SigGain      float32 `def:"3" view:"-" desc:"gain multipler on (net - thr) for sigmoid used for computing values for net < thr"`
	InterpRange  float32 `def:"0.01" view:"-" desc:"interpolation range above zero to use interpolation"`
	GainCorRange float32 `def:"10" view:"-" desc:"range in units of nvar over which to apply gain correction to compensate for convolution"`
	GainCor      float32 `def:"0.1" view:"-" desc:"gain correction multiplier -- how much to correct gains"`

	SigGainNVar float32 `view:"-" desc:"sig_gain / nvar"`
	SigMultEff  float32 `view:"-" desc:"overall multiplier on sigmoidal component for values below threshold = sig_mult * pow(gain * nvar, sig_mult_pow)"`
	SigValAt0   float32 `view:"-" desc:"0.5 * sig_mult_eff -- used for interpolation portion"`
	InterpVal   float32 `view:"-" desc:"function value at interp_range - sig_val_at_0 -- for interpolation"`
}

func (xp *XX1Params) Defaults() {
	xp.Thr = 0.5
	xp.Gain = 100
	xp.NVar = 0.005
	xp.VmActThr = 0.01
	xp.SigMult = 0.33
	xp.SigMultPow = 0.8
	xp.SigGain = 3.0
	xp.InterpRange = 0.01
	xp.GainCorRange = 10.0
	xp.GainCor = 0.1
}

// XX1 computes the basic x/(x+1) function
func (xp *XX1Params) XX1(x float32) float32 { return x / (x + 1) }

// XX1GainCor computes x/(x+1) with gain correction within GainCorRange
// to compensate for convolution effects
func (xp *XX1Params) XX1GainCor(x float32) float32 {
	gainCorFact := (xp.GainCorRange - (x / xp.NVar)) / xp.GainCorRange
	if gainCorFact < 0 {
		return xp.XX1(xp.Gain * x)
	}
	newGain := xp.Gain * (1 - xp.GainCor*gainCorFact)
	return xp.XX1(newGain * x)
}

// NoisyXX1 computes the Noisy x/(x+1) function -- directly computes close approximation
// to x/(x+1) convolved with a gaussian noise function with variance nvar.
// No need for a lookup table -- very reasonable approximation for standard range of parameters
// (nvar = .01 or less -- higher values of nvar are less accurate with large gains,
// but ok for lower gains)
func (xp *XX1Params) NoisyXX1(x float32) float32 {
	if x < 0 { // sigmoidal for < 0
		return xp.SigMultEff / (1 + math32.Exp(-(x * xp.SigGainNVar)))
	} else if x < xp.InterpRange {
		interp := 1 - ((xp.InterpRange - x) / xp.InterpRange)
		return xp.SigValAt0 + interp*xp.InterpVal
	} else {
		return xp.XX1GainCor(x)
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//  From emer.leabra.Inhib

type FFFBParams struct {
	On       bool    `desc:"enable this level of inhibition"`
	Gi       float32 `min:"0" def:"1.8" desc:"[1.5-2.3 typical, can go lower or higher as needed] overall inhibition gain -- this is main parameter to adjust to change overall activation levels -- it scales both the the ff and fb factors uniformly"`
	FF       float32 `viewif:"On" min:"0" def:"1" desc:"overall inhibitory contribution from feedforward inhibition -- multiplies average netinput (i.e., synaptic drive into layer) -- this anticipates upcoming changes in excitation, but if set too high, it can make activity slow to emerge -- see also ff0 for a zero-point for this value"`
	FB       float32 `viewif:"On" min:"0" def:"1" desc:"overall inhibitory contribution from feedback inhibition -- multiplies average activation -- this reacts to layer activation levels and works more like a thermostat (turning up when the 'heat' in the layer is too high)"`
	FBTau    float32 `viewif:"On" min:"0" def:"1.4,3,5" desc:"time constant in cycles, which should be milliseconds typically (roughly, how long it takes for value to change significantly -- 1.4x the half-life) for integrating feedback inhibitory values -- prevents oscillations that otherwise occur -- the fast default of 1.4 should be used for most cases but sometimes a slower value (3 or higher) can be more robust, especially when inhibition is strong or inputs are more rapidly changing"`
	MaxVsAvg float32 `viewif:"On" def:"0,0.5,1" desc:"what proportion of the maximum vs. average netinput to use in the feedforward inhibition computation -- 0 = all average, 1 = all max, and values in between = proportional mix between average and max (ff_netin = avg + ff_max_vs_avg * (max - avg)) -- including more max can be beneficial especially in situations where the average can vary significantly but the activity should not -- max is more robust in many situations but less flexible and sensitive to the overall distribution -- max is better for cases more closely approximating single or strictly fixed winner-take-all behavior -- 0.5 is a good compromise in many cases and generally requires a reduction of .1 or slightly more (up to .3-.5) from the gi value for 0"`
	FF0      float32 `viewif:"On" def:"0.1" desc:"feedforward zero point for average netinput -- below this level, no FF inhibition is computed based on avg netinput, and this value is subtraced from the ff inhib contribution above this value -- the 0.1 default should be good for most cases (and helps FF_FB produce k-winner-take-all dynamics), but if average netinputs are lower than typical, you may need to lower it"`
	FBDt     float32 `inactive:"+" view:"-" desc:"rate = 1 / tau"`
}

func (fb *FFFBParams) Update() {
	fb.FBDt = 1 / fb.FBTau
}

// FFFBInhib contains values for computed FFFB inhibition
type FFFBInhib struct {
	FFi    float32 `desc:"computed feedforward inhibition"`
	FBi    float32 `desc:"computed feedback inhibition (total)"`
	Gi     float32 `desc:"overall value of the inhibition -- this is what is added into the unit Gi inhibition level (along with any synaptic unit-driven inhibition)"`
	GiOrig float32 `desc:"original value of the inhibition (before any  group effects set in)"`
	LayGi  float32 `desc:"for pools, this is the layer-level inhibition that is MAX'd with the pool-level inhibition to produce the net inhibition"`
}

func (fi *FFFBInhib) Init() {
	fi.FFi = 0
	fi.FBi = 0
	fi.Gi = 0
	fi.GiOrig = 0
	fi.LayGi = 0
}

func (fb *FFFBParams) Defaults() {
	fb.Gi = 1.8
	fb.FF = 1
	fb.FB = 1
	fb.FBTau = 1.4
	fb.MaxVsAvg = 0
	fb.FF0 = 0.1
	fb.Update()
}

// FFInhib returns the feedforward inhibition value based on average and max excitatory conductance within
// relevant scope
func (fb *FFFBParams) FFInhib(avgGe, maxGe float32) float32 {
	ffNetin := avgGe + fb.MaxVsAvg*(maxGe-avgGe)
	var ffi float32
	if ffNetin > fb.FF0 {
		ffi = fb.FF * (ffNetin - fb.FF0)
	}
	return ffi
}

// FBInhib computes feedback inhibition value as function of average activation
func (fb *FFFBParams) FBInhib(avgAct float32) float32 {
	fbi := fb.FB * avgAct
	return fbi
}

// FBUpdt updates feedback inhibition using time-integration rate constant
func (fb *FFFBParams) FBUpdt(fbi *float32, newFbi float32) {
	*fbi += fb.FBDt * (newFbi - *fbi)
}

// Inhib is full inhibition computation for given pool activity levels and inhib state
func (fb *FFFBParams) Inhib(avgGe, maxGe, avgAct float32, inh *FFFBInhib) {
	if !fb.On {
		inh.Init()
		return
	}

	ffi := fb.FFInhib(avgGe, maxGe)
	fbi := fb.FBInhib(avgAct)

	inh.FFi = ffi
	fb.FBUpdt(&inh.FBi, fbi)

	inh.Gi = fb.Gi * (ffi + inh.FBi)
	inh.GiOrig = inh.Gi
}

//////////////////////////////////////////////////////////////////////////////////////
//  kwta calculation based on emer.leabra code

type kWinnerTakeAll struct {
	Chans   Chans
	ActPars ActParams
}

func (kwta *kWinnerTakeAll) Initialize() {
	kwta.ActPars.Gbar.E = 1.0
	kwta.ActPars.Gbar.L = 0.2
	kwta.ActPars.Gbar.I = 1.0
	kwta.ActPars.Gbar.K = 1.0
	kwta.ActPars.Erev.E = 1.0
	kwta.ActPars.Erev.L = 0.3
	kwta.ActPars.Erev.I = 0.25
	kwta.ActPars.Erev.K = 0.1
	// these really should be calculated - see update method in Act
	kwta.ActPars.ErevSubThr.E = 0.5
	kwta.ActPars.ErevSubThr.L = -0.19999999
	kwta.ActPars.ErevSubThr.I = -0.25
	kwta.ActPars.ErevSubThr.K = -0.4
}

func (kwta *kWinnerTakeAll) CalcActs(raw, new *etensor.Float32) {
	kwta.Initialize()

	xx1 := XX1Params{}
	xx1.Defaults()

	inhibPars := FFFBParams{}
	inhibPars.Defaults()
	inhibPars.Gi = 1.5
	inhibPars.On = true
	inhib := FFFBInhib{}

	values := raw.Values // these are ge
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
		avgMaxGe.CalcAvg()
		inhibPars.Inhib(avgMaxGe.Avg, avgMaxGe.Max, avgMaxAct.Avg, &inhib)

		// geThr values is negative and shouldn't be

		geThr := float32((kwta.ActPars.Gbar.I*inhib.Gi*kwta.ActPars.ErevSubThr.I + kwta.ActPars.Gbar.L*kwta.ActPars.ErevSubThr.L) / kwta.ActPars.ErevSubThr.E)
		fmt.Printf("geAvg: %v, geMax: %v, actMax: %v, Gi: %v, geThr: %v\n", avgMaxGe.Avg, avgMaxGe.Max, avgMaxAct.Max, inhib.Gi, geThr)
		var ge float32
		for i, act := range acts {
			ge = kwta.ActPars.Gbar.E * act

			// temporary for debug
			geThr = .2
			nwAct := xx1.NoisyXX1(ge*float32(kwta.ActPars.Gbar.E) - geThr) // act is ge
			acts[i] = nwAct
			avgMaxAct.UpdateVal(nwAct, i)
			avgMaxGe.UpdateVal(ge, i)
		}
	}
	for i, act := range acts {
		new.SetFloat1D(i, float64(act))
	}
}
