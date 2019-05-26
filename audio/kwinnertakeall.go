package audio

import (
	"fmt"

	"github.com/emer/etable/etensor"
	"github.com/emer/etable/minmax"
	"github.com/emer/leabra/leabra"
)

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

	xx1 := leabra.XX1Params{}
	xx1.Defaults()

	inhibPars := leabra.FFFBParams{}
	inhibPars.Defaults()
	inhibPars.Gi = 1.5
	inhibPars.On = true
	inhib := leabra.FFFBInhib{}

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
