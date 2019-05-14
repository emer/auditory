// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package audio

import (
	"github.com/chewxy/math32"
	"github.com/emer/etable/etensor"
)

type KwtaSpec struct {
	On        bool    // turn on kwta-style inhibitory competition, implemented via feed-forward and feed-back inhibitory functions of net input and activation within the layer
	Gi        float32 // #CONDSHOW_ON_on #DEF_1.5:2 typically between 1.5-2 -- sets overall level of inhibition for feedforward / feedback inhibition at the unit group level (see lay_gi for layer level parameter)
	LayGi     float32 // #CONDSHOW_ON_on #DEF_1:2 sets overall level of inhibition for feedforward / feedback inhibition for the entire layer level -- the actual inhibition at each unit group is then the MAX of this computed inhibition and that computed for the unit group individually
	Ff        float32 // #HIDDEN #NO_SAVE #DEF_1 overall inhibitory contribution from feedforward inhibition -- computed from average netinput -- fixed to 1
	Fb        float32 // #HIDDEN #NO_SAVE #DEF_0.5 overall inhibitory contribution from feedback inhibition -- computed from average activation
	NCyc      int     // #HIDDEN #NO_SAVE #DEF_20 number of cycle iterations to perform on fffb inhib
	Cycle     int     // #HIDDEN #NO_SAVE current cycle of fffb settling
	ActDt     float32 // #HIDDEN #NO_SAVE #DEF_0.3 time constant for integrating activations -- only for FFFB inhib 
	FbDt      float32 // #HIDDEN #NO_SAVE #DEF_0.7 time constant for integrating fb inhib
	MaxDa     float32 // #HIDDEN #NO_SAVE current max delta activation for fffb settling
	MaxDaCrit float32 // #HIDDEN #NO_SAVE stopping criterion for activation change for fffb
	Ff0       float32 // #HIDDEN #NO_SAVE #DEF_0.1 feedforward zero point in terms of average netinput -- below this level, no FF inhibition is computed -- the 0.1 default should be good for most cases -- fixed to 0.1
	Gain      float32 // #CONDSHOW_ON_on #DEF_20;40;80 gain on the NXX1 activation function (based on g_e - g_e_Thr value -- i.e. the gelin version of the function)
	NVar      float32 // #CONDSHOW_ON_on #DEF_0.01 noise variance to convolve with XX1 function to obtain NOISY_XX1 function -- higher values make the function more gradual at the bottom
	GBarL     float32 // #CONDSHOW_ON_on #DEF_0.1;0.3 leak current conductance value -- determines neural response to weak inputs -- a higher value can damp the neural response
	GBarE     float32 // #HIDDEN #NO_SAVE excitatory conductance multiplier -- multiplies filter input value prior to computing membrane potential -- general target is to have max excitatory input = .5, so with 0-1 normalized inputs, this value is automatically set to .5
	ERevE     float32 // #HIDDEN #NO_SAVE excitatory reversal potential -- automatically set to default value of 1 in normalized units
	ERevL     float32 // #HIDDEN #NO_SAVE leak and inhibition reversal potential -- automatically set to 0.3 (gelin default)
	Thr       float32 // #HIDDEN #NO_SAVE firing Threshold -- automatically set to default value of .5 (gelin default)

	Nxx1Fun   etensor.Float32 // #HIDDEN #NO_SAVE #NO_INHERIT #CAT_Activation convolved gaussian and x/x+1 function as lookup table
	NoiseConv etensor.Float32 // #HIDDEN #NO_SAVE #NO_INHERIT #CAT_Activation gaussian for convolution

	GlEl           float32 // #READ_ONLY #NO_SAVE kwta.GBarL * e_rev_l -- just a compute time saver
	ERevSubThrE    float32 // #READ_ONLY #NO_SAVE #HIDDEN e_rev_e - thr -- used for compute_ithresh
	ERevSubThrI    float32 // #READ_ONLY #NO_SAVE #HIDDEN e_rev_i - thr -- used for compute_ithresh
	GblERevSubThrL float32 // #READ_ONLY #NO_SAVE #HIDDEN kwta.GBarL * (e_rev_l - thr) -- used for compute_ithresh
	ThrSubERevI    float32 // #READ_ONLY #NO_SAVE #HIDDEN thr - e_rev_i used for compute_ithresh
	ThrSubERevE    float32 // #READ_ONLY #NO_SAVE #HIDDEN thr - e_rev_e used for compute_ethresh
}

func (kwta *KwtaSpec) Initialize() {
	kwta.On = true
	kwta.Mode = NOT_LOADED
	kwta.Gi = 2.0
	kwta.LayGi = 1.5
	kwta.Ff = 1.0
	kwta.Fb = 0.5
	kwta.Cycle = 0
	kwta.NCyc = 20
	kwta.ActDt = 0.3
	kwta.FbDt = 0.7
	kwta.MaxDaCrit = 0.005
	kwta.Ff0 = 0.1
	kwta.Gain = 80.0
	kwta.NVar = 0.01
	kwta.GBarL = 0.1

	// gelin defaults:
	kwta.GBarE = 0.5
	kwta.ERevE = 1.0
	kwta.ERevL = 0.3
	kwta.Thr = 0.5

	kwta.NoiseConv.x_range.min = -.05
	kwta.NoiseConv.x_range.max = .05
	kwta.NoiseConv.res = .001
	kwta.NoiseConv.UpdateAfterEdit_NoGui()

	kwta.Nxx1Fun.x_range.min = -.03
	kwta.Nxx1Fun.x_range.max = 1.0
	kwta.Nxx1Fun.res = .001
	//kwta.Nxx1Fun.UpdateAfterEdit_NoGui()

	kwta.GlEl = kwta.GBarL * kwta.ERevL
	kwta.ERevSubThrE = kwta.ERevE - kwta.Thr
	kwta.ERevSubThrI = kwta.ERevL - kwta.Thr
	kwta.GblERevSubThrL = kwta.GBarL * (kwta.ERevL - kwta.Thr)
	kwta.ThrSubERevI = kwta.Thr - kwta.ERevL
	kwta.ThrSubERevE = kwta.Thr - kwta.ERevE

	CreateNXX1Fun()
}

func (kwta *KwtaSpec) UpdateAfterEdit() {
	if (mode != NOT_LOADED) {
		if (mode != OFF)
		kwta.On = true;
		else
		kwta.On = false;
		mode = NOT_LOADED;
	}

	// these are all gelin defaults
	kwta.GBarE = 0.5
	kwta.ERevE = 1.0
	kwta.ERevL = 0.3
	kwta.Thr = 0.5

	kwta.GlEl = kwta.GBarL * kwta.ERevL;
	kwta.ERevSubThrE = kwta.ERevE - kwta.Thr;
	kwta.ERevSubThrI = kwta.ERevL - kwta.Thr;
	kwta.GblERevSubThrL = kwta.GBarL * (kwta.ERevL - kwta.Thr);
	kwta.ThrSubERevI = kwta.Thr - kwta.ERevL;
	kwta.ThrSubERevE = kwta.Thr - kwta.ERevE;
	CreateNXX1Fun();
}

func (kwta *KwtaSpec) CreateNXX1Fun() {
	// first create the gaussian noise convolver
	kwta.Nxx1Fun.x_range.max = 1.0
	kwta.Nxx1Fun.res = .001 // needs same fine res to get the noise transitions
	kwta.Nxx1Fun.UpdateAfterEdit()
	ns_rng := 3.0 * kwta.NVar // range factor based on noise level -- 3 sd
	ns_rng = math32.Max(ns_rng, kwta.Nxx1Fun.res)
	kwta.Nxx1Fun.x_range.min = -ns_rng
	kwta.NoiseConv.x_range.min = -ns_rng
	kwta.NoiseConv.x_range.max = ns_rng
	kwta.NoiseConv.res = kwta.Nxx1Fun.res
	kwta.NoiseConv.UpdateAfterEdit_NoGui()
	kwta.NoiseConv.AllocForRange()
	effNVar := math32.Max(kwta.NVar, 1.0e-6) // just too lazy to do proper conditional for 0..
	v := effNVar * effNVar
	for i := 0
	i < kwta.NoiseConv.size
	i++
	{
		x := kwta.NoiseConv.Xval(i)
		kwta.NoiseConv[i] = math32.Expe(-((x * x) / v // shouldn't there be a factor of 1/2 here..?
	}

	// normalize it
	sum := 0.0
	for
	i := 0
	i < kwta.NoiseConv.size
	i++
	{
		sum += kwta.NoiseConv[i]
	}
	for i := 0
	i < kwta.NoiseConv.size
	i++
	{
		kwta.NoiseConv[i] /= sum
	}

	// then create the initial function
	FunLookup
	fun;
	fun.x_range.min = kwta.Nxx1Fun.x_range.min + kwta.NoiseConv.x_range.min;
	fun.x_range.max = kwta.Nxx1Fun.x_range.max + kwta.NoiseConv.x_range.max;
	fun.res = kwta.Nxx1Fun.res;
	fun.UpdateAfterEdit_NoGui();
	fun.AllocForRange();
	for i := 0; i < fun.size; i++ {
		x := fun.Xval(i);
		val := 0.0
		if x > 0.0 {
		val = (kwta.Gain * x) / ((kwta.Gain * x) + 1.0)
		fun[i] = val;
	}

	kwta.Nxx1Fun.Convolve(fun, kwta.NoiseConv); // does alloc
}

func (kwta *KwtaSpec) Compute_FFFB(inputs, outputs, gcIMat etensor.Float32) {

	int
	gxs = inputs.dim(0);
	int
	gys = inputs.dim(1);
	int
	ixs = inputs.dim(2);
	int
	iys = inputs.dim(3);
	if (gxs == 0 || gys == 0 || ixs == 0 || iys == 0)
	return;
	float
	normval = 1.0
	f / (gxs * gys);
	float
	lay_normval = 1.0
	f / (ixs * iys);
	float
	dtc = 1.0
	f - fb_dt;
	float
	lay_avg_netin = 0.0
	f;
	float
	lay_avg_act = 0.0
	f;
	float
	max_gi = 0.0
	f;
	for
	(int
	iy = 0;
	iy < iys;
	iy++) {
		for
		(int
		ix = 0;
		ix < ixs;
		ix++) {
			float
			avg_netin = 0.0
			f;
			if (cycle == 0) {
				for
				(int
				gy = 0;
				gy < gys;
				gy++) {
					for
					(int
					gx = 0;
					gx < gxs;
					gx++) {
						avg_netin += inputs.FastEl4d(gx, gy, ix, iy);
					}
				}
				avg_netin *= normval;
				gc_i_mat.FastEl3d(ix, iy, 2) = avg_netin;
			} else {
				avg_netin = gc_i_mat.FastEl3d(ix, iy, 2);
			}
			lay_avg_netin += avg_netin;
			float
			avg_act = 0.0
			f;
			for
			(int
			gy = 0;
			gy < gys;
			gy++) {
				for
				(int
				gx = 0;
				gx < gxs;
				gx++) {
					avg_act += outputs.FastEl4d(gx, gy, ix, iy);
				}
			}
			avg_act *= normval;
			lay_avg_act += avg_act;
			float
			nw_ffi = FFInhib(avg_netin);
			float
			nw_fbi = FBInhib(avg_act);
			float & fbi = gc_i_mat.FastEl3d(ix, iy, 1);
			fbi = fb_dt*nw_fbi + dtc*fbi;
			float
			nw_gi = gi * (nw_ffi + nw_fbi);
			gc_i_mat.FastEl3d(ix, iy, 0) = nw_gi;
			max_gi = MAX(max_gi, nw_gi);
		}
	}
	lay_avg_netin *= lay_normval;
	lay_avg_act *= lay_normval;
	float
	nw_ffi = FFInhib(lay_avg_netin);
	float
	nw_fbi = FBInhib(lay_avg_act);
	float & fbi = gc_i_mat.FastEl3d(0, 0, 3); // 3 = extra guy for layer
	fbi = fb_dt*nw_fbi + dtc*fbi;
	float
	lay_i = kwta.LayGi * (nw_ffi + nw_fbi);
	for
	(int
	iy = 0;
	iy < iys;
	iy++) {
		for
		(int
		ix = 0;
		ix < ixs;
		ix++) {
			float
			gig = gc_i_mat.FastEl3d(ix, iy, 0);
			gig = MAX(gig, lay_i);
			gc_i_mat.FastEl3d(ix, iy, 0) = gig;
		}
	}
}

bool V1KwtaSpec::Compute_Inhib(float_Matrix& inputs, float_Matrix& outputs,
float_Matrix& gc_i_mat) {
if (TestError(inputs.dims() != 4, "Compute_Kwta",
"input matrix must have 4 dimensions: gp x,y, outer (image) x,y"))
return false;

if (!kwta.On) return false;
outputs.InitVals(0.0f);
cycle = 0;
int ixs = inputs.dim(2);
int iys = inputs.dim(3);
gc_i_mat.SetGeom(3, ixs, iys, 4); // extra copy to hold onto fb inhib for temp integ, and for the avg_netin
gc_i_mat.InitVals(0.0f);
for (int i = 0; i<n_cyc; i++) {
Compute_FFFB(inputs, outputs, gc_i_mat);
Compute_Act(inputs, outputs, gc_i_mat);
cycle++;
if (max_da < max_da_crit)
break;
}
return true;
}

bool V1KwtaSpec::Compute_Inhib_Extra(float_Matrix& inputs, float_Matrix& outputs,
float_Matrix& gc_i_mat, float_Matrix& extra_inh) {
if (TestError(inputs.dims() != 4, "Compute_Kwta",
"input matrix must have 4 dimensions: gp x,y, outer (image) x,y"))
return false;

if (!kwta.On) return false;
outputs.InitVals(0.0f);
cycle = 0;
int ixs = inputs.dim(2);
int iys = inputs.dim(3);
gc_i_mat.SetGeom(3, ixs, iys, 4);
gc_i_mat.InitVals(0.0f);
for (int i = 0; i<n_cyc; i++) {
Compute_FFFB(inputs, outputs, gc_i_mat);
Compute_Act_Extra(inputs, outputs, gc_i_mat, extra_inh);
cycle++;
if (max_da < max_da_crit)
break;
}
return true;
}

void V1KwtaSpec::Compute_Act(float_Matrix& inputs, float_Matrix& outputs,
float_Matrix& gc_i_mat) {
int gxs = inputs.dim(0);
int gys = inputs.dim(1);
int ixs = inputs.dim(2);
int iys = inputs.dim(3);

float dtc = (1.0f - act_dt);
max_da = 0.0f;
for (int iy =0; iy < iys; iy++) {
for (int ix = 0; ix < ixs; ix++) {
float gig = gc_i_mat.FastEl2d(ix, iy);
for (int gy = 0; gy < gys; gy++) {
for (int gx = 0; gx < gxs; gx++) {
float raw = inputs.FastEl4d(gx, gy, ix, iy);
float ge = GBarE * raw;
float act = Compute_ActFmIn(ge, gig);
float& out = outputs.FastEl4d(gx, gy, ix, iy);
float da = fabsf(act - out);
max_da = MAX(da, max_da);
out = act_dt * act + dtc * out;
}
}
}
}
}

void V1KwtaSpec::Compute_Act_Extra(float_Matrix& inputs, float_Matrix& outputs,
float_Matrix& gc_i_mat, float_Matrix& extra_inh) {
int gxs = inputs.dim(0);
int gys = inputs.dim(1);
int ixs = inputs.dim(2);
int iys = inputs.dim(3);

float dtc = (1.0f - act_dt);
max_da = 0.0f;
for (int iy =0; iy < iys; iy++) {
for (int ix = 0; ix < ixs; ix++) {
float gig = gc_i_mat.FastEl2d(ix, iy);
for (int gy = 0; gy < gys; gy++) {
for (int gx = 0; gx < gxs; gx++) {
float raw = inputs.FastEl4d(gx, gy, ix, iy);
float ge = GBarE * raw;
float ei = extra_inh.FastEl4d(gx, gy, ix, iy);
float eig = gi * FFInhib(extra_inh.FastEl4d(gx, gy, ix, iy));
float gi_eff = MAX(gig, eig);
float act = Compute_ActFmIn(ge, gi_eff);
float& out = outputs.FastEl4d(gx, gy, ix, iy);
float da = fabsf(act - out);
max_da = MAX(da, max_da);
out = act_dt * act + dtc * out;
}
}
}
}
}
