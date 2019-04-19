// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package audio

import (
	"fmt"
	"github.com/emer/emergent/etensor"
	"math"
	"strconv"

	"github.com/chewxy/math32"
)

// AudInputSpec defines the sound input parameters for auditory processing
type AudInputSpec struct {
	WinMsec      float32 `desc:"#DEF_25 input window -- number of milliseconds worth of sound to filter at a time"`
	StepMsec     float32 `desc:"#DEF_5;10;12.5 input step -- number of milliseconds worth of sound that the input is stepped along to obtain the next window sample""`
	TrialMsec    float32 `desc:"#DEF_100 length of a full trial's worth of input -- total number of milliseconds to accumulate into a complete trial of activations to present to a network -- must be a multiple of step_msec -- input will be trial_msec / step_msec = trial_steps wide in the X axis, and number of filters in the Y axis"`
	BorderSteps  uint32  `desc:"number of steps before and after the trial window to preserve -- this is important when applying temporal filters that have greater temporal extent"`
	SampleRate   uint32  `desc:"rate of sampling in our sound input (e.g., 16000 = 16Khz) -- can initialize this from a taSound object using InitFromSound method"`
	Channels     uint16  `desc:"total number of channels to process"`
	Channel      uint16  `desc:"#CONDSHOW_ON_channels:1 specific channel to process, if input has multiple channels, and we only process one of them (-1 = process all)"`
	WinSamples   uint32  `desc:"#READ_ONLY #SHOW total number of samples to process (win_msec * .001 * sample_rate)"`
	StepSamples  uint32  `desc:"#READ_ONLY #SHOW total number of samples to step input by (step_msec * .001 * sample_rate)"`
	TrialSamples uint32  `desc:"#READ_ONLY #SHOW total number of samples in a trial  (trail_msec * .001 * sample_rate)"`
	TrialSteps   uint32  `desc:"#READ_ONLY #SHOW total number of steps in a trial  (trail_msec / step_msec)"`
	TotalSteps   uint32  `desc:"#READ_ONLY #SHOW 2*border_steps + trial_steps -- total in full window"`
}

//Init initializes the the AudInputSpec
func (ais *AudInputSpec) Initialize() {
	ais.WinMsec = 25.0
	ais.StepMsec = 5.0
	ais.TrialMsec = 100.0
	ais.BorderSteps = 12
	ais.SampleRate = 16000
	ais.Channels = 1
	ais.Channel = 0
	ais.ComputeSamples()
}

// ComputeSamples computes the sample counts based on time and sample rate
func (ais *AudInputSpec) ComputeSamples() {
	ais.WinSamples = MSecToSamples(ais.WinMsec, ais.SampleRate)
	ais.StepSamples = MSecToSamples(ais.StepMsec, ais.SampleRate)
	ais.TrialSamples = MSecToSamples(ais.TrialMsec, ais.SampleRate)
	ais.TrialSteps = uint32(math.Round(float64(ais.TrialMsec / ais.StepMsec)))
	ais.TotalSteps = 2*ais.BorderSteps + ais.TrialSteps
}

// MSecToSamples converts milliseconds to samples, in terms of sample_rate
func MSecToSamples(msec float32, rate uint32) uint32 {
	return uint32(math.Round(float64(msec) * 0.001 * float64(rate)))
}

// SamplesToMSec converts samples to milliseconds, in terms of sample_rate
func SamplesToMSec(samples uint32, rate uint32) float32 {
	return 1000.0 * float32(samples) / float32(rate)
}

// InitFromSound loads a sound and sets the AudInputSpec channel vars and sample rate
func (ais *AudInputSpec) InitFromSound(snd *Sound, nChannels uint16, channel uint16) {
	if snd == nil {
		fmt.Printf("InitFromSound: sound nil")
		return
	}

	ais.SampleRate = snd.SampleRate()
	ais.ComputeSamples()
	if nChannels < 1 {
		ais.Channels = snd.Channels()
	} else {
		ais.Channels = uint16(math32.Min(float32(nChannels), float32(ais.Channels)))
	}
	if ais.Channels > 1 {
		ais.Channel = channel
	} else {
		ais.Channel = 0
	}
}

// AudDftSpec discrete fourier transform (dft) specifications
type AudDftSpec struct {
	LogPow         bool    `desc:"#DEF_true compute the log of the power and save that to a separate table -- generaly more useful for visualization of power than raw power values"`
	LogOff         float32 `desc:"#CONDSHOW_ON_log_pow #DEF_0 add this amount when taking the log of the dft power -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs"`
	LogMin         float32 `desc:"#CONDSHOW_ON_log_pow #DEF_-100 minimum value a log can produce -- puts a lower limit on log output"`
	PreviousSmooth float32 `desc:"#DEF_0 how much of the previous step's power value to include in this one -- smooths out the power spectrum which can be artificially bumpy due to discrete window samples"`
	CurrentSmooth  float32 `desc:"#READ_ONLY #EXPERT 1 - prv_smooth -- how much of current power to include"`
}

func (ad *AudDftSpec) Initialize() {
	ad.PreviousSmooth = 0
	ad.CurrentSmooth = 1.0 - ad.PreviousSmooth
	ad.LogPow = true
	ad.LogOff = 0
	ad.LogMin = -100
}

// AudRenormSpec holds the auditory renormalization parameters
type AudRenormSpec struct {
	On          bool    `desc:"perform renormalization of this level of the auditory signal"`
	RenormMin   float32 `desc:"#CONDSHOW_ON_on minimum value to use for renormalization -- you must experiment with range of inputs to determine appropriate values"`
	RenormMax   float32 `desc:"#CONDSHOW_ON_on maximum value to use for renormalization -- you must experiment with range of inputs to determine appropriate values"`
	RenormScale float32 `desc:"#READ_ONLY 1.0 / (ren_max - ren_min)"`
}

func (ar *AudRenormSpec) Initialize() {
	ar.On = true
	ar.RenormMin = -10.0
	ar.RenormMax = 7.0
	ar.RenormScale = 1.0 / (ar.RenormMax - ar.RenormMin)
}

// AudGaborSpec params for auditory gabor filters: 2d Gaussian envelope times a sinusoidal plane wave --
// by default produces 2 phase asymmetric edge detector filters -- horizontal tuning is different from V1 version --
// has elongated frequency-band specific tuning, not a parallel horizontal tuning -- and has multiple of these
type AudGaborSpec struct {
	On              bool    `desc:"use this gabor filtering of the time-frequency space filtered input (time in terms of steps of the DFT transform, and discrete frequency factors based on the FFT window and input sample rate)"`
	SizeTime        int     `desc:"#CONDSHOW_ON_on #DEF_6;8;12;16;24 size of the filter in the time (horizontal) domain, in terms of steps of the underlying DFT filtering steps"`
	SizeFreq        int     `desc:"#CONDSHOW_ON_on #DEF_6;8;12;16;24 size of the filter in the frequency domain, in terms of discrete frequency factors based on the FFT window and input sample rate"`
	SpaceTime       int     `desc:"#CONDSHOW_ON_on spacing in the time (horizontal) domain, in terms of steps"`
	SpaceFreq       int     `desc:"#CONDSHOW_ON_on spacing in the frequency (vertical) domain"`
	WaveLen         float32 `desc:"#CONDSHOW_ON_on #DEF_1.5;2 wavelength of the sine waves in normalized units"`
	SigmaLen        float32 `desc:" #CONDSHOW_ON_on #DEF_0.6 gaussian sigma for the length dimension (elongated axis perpendicular to the sine waves) -- normalized as a function of filter size in relevant dimension"`
	SigmaWidth      float32 `desc:"#CONDSHOW_ON_on #DEF_0.3 gaussian sigma for the width dimension (in the direction of the sine waves) -- normalized as a function of filter size in relevant dimension"`
	SigmaLenHoriz   float32 `desc:"#CONDSHOW_ON_on #DEF_0.3 gaussian sigma for the length of special horizontal narrow-band filters -- normalized as a function of filter size in relevant dimension"`
	SigmaWidthHoriz float32 `desc:"#CONDSHOW_ON_on #DEF_0.1 gaussian sigma for the horizontal dimension for special horizontal narrow-band filters -- normalized as a function of filter size in relevant dimension"`
	Gain            float32 `desc:"#CONDSHOW_ON_on #DEF_2 overall gain multiplier applied after gabor filtering -- only relevant if not using renormalization (otherwize it just gets renormed away)"`
	NHoriz          int     `desc:"#CONDSHOW_ON_on #DEF_4 number of horizontally-elongated,  pure time-domain, frequency-band specific filters to include, evenly spaced over the available frequency space for this filter set -- in addition to these, there are two diagonals (45, 135) and a vertically-elongated (wide frequency band) filter"`
	PhaseOffset     float32 `desc:"#CONDSHOW_ON_on #DEF_0;1.5708 offset for the sine phase -- default is an asymmetric sine wave -- can make it into a symmetric cosine gabor by using PI/2 = 1.5708"`
	CircleEdge      bool    `desc:"#CONDSHOW_ON_on #DEF_true cut off the filter (to zero) outside a circle of diameter filter_size -- makes the filter more radially symmetric"`
	NFilters        int     `desc:"#CONDSHOW_ON_on #READ_ONLY #SHOW total number of filters = 3 + n_horiz"`
}

func (ag *AudGaborSpec) Initialize() {
	ag.On = true
	ag.Gain = 2.0
	ag.NHoriz = 4
	ag.SizeTime = 6.0
	ag.SizeFreq = 6.0
	ag.WaveLen = 1.5
	ag.SigmaLen = 0.6
	ag.SigmaWidth = 0.3
	ag.SigmaLenHoriz = 0.3
	ag.SigmaWidthHoriz = 0.1
	ag.PhaseOffset = 0.0
	ag.CircleEdge = true
	ag.NFilters = 3 + ag.NHoriz
}

// RenderFilters generates filters into the given matrix, which is formatted as: [sz_time_steps][sz_freq][n_filters]
func (ag *AudGaborSpec) RenderFilters(filters *etensor.Float32) {
	//fltrs.SetGeom(3, sz_time, sz_freq, n_filters);
	//float
	//ctr_t = (float)(sz_time-1) / 2.0
	//f;
	//float
	//ctr_f = (float)(sz_freq-1) / 2.0
	//f;
	//float
	//ang_inc = taMath_float::pi / (float)
	//4.0
	//f;
	//float
	//radius_t = (float)(sz_time) / 2.0
	//f;
	//float
	//radius_f = (float)(sz_freq) / 2.0
	//f;
	//float
	//len_norm = 1.0
	//f / (2.0
	//f * sig_len * sig_len);
	//float
	//wd_norm = 1.0
	//f / (2.0
	//f * sig_wd * sig_wd);
	//float
	//hor_len_norm = 1.0
	//f / (2.0
	//f * sig_hor_len * sig_hor_len);
	//float
	//hor_wd_norm = 1.0
	//f / (2.0
	//f * sig_hor_wd * sig_hor_wd);
	//float
	//twopinorm = (2.0
	//f * taMath_float::pi) / wvlen;
	//float
	//hctr_inc = (float)(sz_freq-1) / (float)(n_horiz+1);
	//int
	//fli = 0;
	//for
	//(int
	//hi = 0;
	//hi < n_horiz;
	//hi++, fli++) {
	//
	//	float
	//	hctr_f = hctr_inc * (float)(hi+1);
	//	float
	//	angf = -2.0
	//	f * ang_inc;
	//	for
	//	(int
	//	y = 0;
	//	y < sz_freq;
	//	y++) {
	//		for
	//		(int
	//		x = 0;
	//		x < sz_time;
	//		x++) {
	//			float
	//			xf = (float)
	//			x - ctr_t;
	//			float
	//			yf = (float)
	//			y - hctr_f;
	//			float
	//			xfn = xf / radius_t;
	//			float
	//			yfn = yf / radius_f;
	//			float
	//			dist = taMath_float::hypot(xfn, yfn);
	//			float
	//			val = 0.0
	//			f;
	//			if (!(circle_edge && (dist > 1.0f))) {
	//			float nx = xfn * cosf(angf) - yfn * sinf(angf);
	//			float ny = yfn * cosf(angf) + xfn * sinf(angf);
	//			float gauss = expf(-(hor_wd_norm * (nx * nx) + hor_len_norm * (ny * ny)));
	//			float sin_val = sinf(twopinorm * ny + phase_off);
	//			val = gauss * sin_val;
	//			}
	//			fltrs.FastEl3d(x, y, fli) = val;
	//		}
	//	}
	//}
	//for
	//(int
	//ang = 1;
	//ang < 4;
	//ang++, fli++) {
	//	float
	//	angf = -(float)
	//	ang * ang_inc;
	//	for
	//	(int
	//	y = 0;
	//	y < sz_freq;
	//	y++) {
	//		for
	//		(int
	//		x = 0;
	//		x < sz_time;
	//		x++) {
	//			float
	//			xf = (float)
	//			x - ctr_t;
	//			float
	//			yf = (float)
	//			y - ctr_f;
	//			float
	//			xfn = xf / radius_t;
	//			float
	//			yfn = yf / radius_f;
	//			float
	//			dist = taMath_float::hypot(xfn, yfn);
	//			float
	//			val = 0.0
	//			f;
	//			if (!(circle_edge && (dist > 1.0f))) {
	//			float nx = xfn * cosf(angf) - yfn * sinf(angf);
	//			float ny = yfn * cosf(angf) + xfn * sinf(angf);
	//			float gauss = expf(-(len_norm * (nx * nx) + wd_norm * (ny * ny)));
	//			float sin_val = sinf(twopinorm * ny + phase_off);
	//			val = gauss * sin_val;
	//			}
	//			fltrs.FastEl3d(x, y, fli) = val;
	//		}
	//	}
	//}
	//
	//// renorm each half
	//for (fli = 0; fli < n_filters; fli++) {
	//	float
	//	pos_sum = 0.0
	//	f;
	//	float
	//	neg_sum = 0.0
	//	f;
	//	for
	//	(int
	//	y = 0;
	//	y < sz_freq;
	//	y++) {
	//		for
	//		(int
	//		x = 0;
	//		x < sz_time;
	//		x++) {
	//			float & val = fltrs.FastEl3d(x, y, fli);
	//			if (val > 0.0f)          {
	//				pos_sum += val;
	//			}
	//			else if (val < 0.0f)     {
	//				neg_sum += val;
	//			}
	//		}
	//	}
	//	float
	//	pos_norm = 1.0
	//	f / pos_sum;
	//	float
	//	neg_norm = -1.0
	//	f / neg_sum;
	//	for
	//	(int
	//	y = 0;
	//	y < sz_freq;
	//	y++) {
	//		for
	//		(int
	//		x = 0;
	//		x < sz_time;
	//		x++) {
	//			float & val = fltrs.FastEl3d(x, y, fli);
	//			if (val > 0.0f)          {
	//				val *= pos_norm;
	//			}
	//			else if (val < 0.0f)     {
	//				val *= neg_norm;
	//			}
	//		}
	//	}
	//}
}

// GridFilters #BUTTON #NULL_OK_0 #NULL_TEXT_0_NewDataTable plot the filters into data table and generate a grid view (reset any existing data first)
//func (ag *AudGaborSpec) GridFilters(filters *etensor.Float32, graphData *DataTable, reset bool) {
//
//	RenderFilters(fltrs); // just to make sure
//
//	String
//	name;
//	if (owner) name = owner- > GetName();
//	taProject * proj = GetMyProj();
//	if (!graph_data) {
//		graph_data = proj- > GetNewAnalysisDataTable(name+"_V1Gabor_GridFilters", true);
//	}
//	graph_data- > StructUpdate(true);
//	if (reset)
//	graph_data- > ResetData();
//	int
//	idx;
//	DataCol * nmda = graph_data- > FindMakeColName("Name", idx, VT_STRING);
//	//   nmda->SetUserData("WIDTH", 10);
//	DataCol * matda = graph_data- > FindMakeColName("Filter", idx, VT_FLOAT, 2, sz_time,
//		sz_freq);
//	float
//	maxv = taMath_float::vec_abs_max(&fltrs, idx);
//	graph_data- > SetUserData("N_ROWS", n_filters);
//	graph_data- > SetUserData("SCALE_MIN", -maxv);
//	graph_data- > SetUserData("SCALE_MAX", maxv);
//	graph_data- > SetUserData("BLOCK_HEIGHT", 0.0
//	f);
//	for
//	(int
//	i = 0;
//	i < n_filters;
//	i++) {
//		graph_data- > AddBlankRow();
//		float_MatrixPtr
//		frm;
//		frm = (float_Matrix *)
//		fltrs.GetFrameSlice(i);
//		matda- > SetValAsMatrix(frm, -1);
//		nmda- > SetValAsString("Filter: "+String(i), -1);
//	}
//
//	graph_data- > StructUpdate(false);
//	graph_data- > FindMakeGridView();
//}

// MelFBankSpec contains mel frequency feature bank sampling parameters
type MelFBankSpec struct {
	On       bool    `desc:"perform mel-frequency filtering of the fft input"`
	LoHz     float32 `desc:"#DEF_120;300 #CONDSHOW_ON_on low frequency end of mel frequency spectrum"`
	HiHz     float32 `desc:"#DEF_10000;8000 #CONDSHOW_ON_on high frequency end of mel frequency spectrum -- must be <= sample_rate / 2 (i.e., less than the Nyquist frequency)"`
	NFilters uint32  `desc:"#DEF_32;26 #CONDSHOW_ON_on number of Mel frequency filters to compute"`
	LogOff   float32 `desc:"#CONDSHOW_ON_on #DEF_0 on add this amount when taking the log of the Mel filter sums to produce the filter-bank output -- e.g., 1.0 makes everything positive -- affects the relative contrast of the outputs"`
	LogMin   float32 `desc:"#CONDSHOW_ON_on #DEF_-10 minimum value a log can produce -- puts a lower limit on log output"`
	LoMel    float32 `desc:"#READ_ONLY #SHOW #CONDSHOW_ON_on low end of mel scale in mel units"`
	HiMel    float32 `desc:"#READ_ONLY #SHOW #CONDSHOW_ON_on high end of mel scale in mel units"`
}

// FreqToMel converts frequency to mel scale
func FreqToMel(freq float32) float32 {
	return 1127.0 * math32.Log(1.0+freq/700.0)
}

// FreqToMel converts mel scale to frequency
func MelToFreq(mel float32) float32 {
	return 700.0 * (math32.Exp(mel/1127.0) - 1.0)
}

// FreqToBin converts frequency into FFT bin number, using parameters of number of FFT bins and sample rate
func FreqToBin(freq, nFft, sampleRate float32) uint32 {
	return uint32(math32.Floor(((nFft + 1) * freq) / sampleRate))
}

func (mfb *MelFBankSpec) Initialize() {
	mfb.On = true
	mfb.LoHz = 120.0
	mfb.HiHz = 10000.0
	mfb.NFilters = 32
	mfb.LogOff = 0.0
	mfb.LogMin = -10.0
	mfb.LoMel = FreqToMel(mfb.LoHz)
	mfb.HiMel = FreqToMel(mfb.HiHz)
}

// MelCepstrumSpec holds the mel frequency sampling parameters
type MelCepstrumSpec struct {
	On     bool `desc:"perform cepstrum discrete cosine transform (dct) of the mel-frequency filter bank features"`
	NCoeff int  `desc:"#CONDSHOW_ON_on #DEF_13 number of mfcc coefficients to output -- typically 1/2 of the number of filterbank features"`
}

func (mc *MelCepstrumSpec) Initialize() {
	mc.On = true
	mc.NCoeff = 13
}

type AuditoryProc struct {
	//	enum SaveMode {               // how to add new data to the data table
	//	NONE_SAVE,                  // don't save anything at all -- overrides any more specific save guys and prevents any addition or modification to the data table
	//	FIRST_ROW,                  // always overwrite the first row -- does EnforceRows(1) if rows = 0
	//	ADD_ROW,                    // always add a new row and write to that, preserving a history of inputs over time -- should be reset at some interval!
	//};
	//
	//
	//	DataTableRef  data_table;     // data table for saving filter results for viewing and applying to networks etc
	//	SaveMode      save_mode;      // how to add new data to the data table
	Input       AudInputSpec    `desc:"specifications of the raw auditory input"`
	Dft         AudDftSpec      `desc:"specifications for how to compute the discrete fourier transform (DFT, using FFT)"`
	MelFBank    MelFBankSpec    `desc:"specifications of the mel feature bank frequency sampling of the DFT (FFT) of the input sound"`
	FBankRenorm AudRenormSpec   `desc:"#CONDSHOW_ON_mel_fbank.on renormalization parmeters for the mel_fbank values -- performed prior to further processing"`
	Gabor1      AudGaborSpec    `desc:"#CONDSHOW_ON_mel_fbank.on full set of frequency / time gabor filters -- first size"`
	Gabor2      AudGaborSpec    `desc:"#CONDSHOW_ON_mel_fbank.on full set of frequency / time gabor filters -- second size"`
	Gabor3      AudGaborSpec    `desc:"#CONDSHOW_ON_mel_fbank.on full set of frequency / time gabor filters -- third size"`
	Mfcc        MelCepstrumSpec `desc:"#CONDSHOW_ON_mel_fbank.on specifications of the mel cepstrum discrete cosine transform of the mel fbank filter features"`
	//	V1KwtaSpec    gabor_kwta;     // #CONDSHOW_ON_gabor1.on k-winner-take-all inhibitory dynamics for the time-gabor output

	// Filters
	DftSize        uint32 `desc:"#READ_ONLY #NO_SAVE full size of fft output -- should be input.win_samples"`
	DftUse         uint32 `desc:"#READ_ONLY #NO_SAVE number of dft outputs to actually use -- should be dft_size / 2 + 1"`
	MelNFiltersEff uint32 `desc:"#READ_ONLY #NO_SAVE effective number of mel filters: mel.n_filters + 2"`

	MelPtsMel        etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel_n_filters_eff] scale points in mel units (mels)"`
	MetPtsHz         etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel_n_filters_eff] mel scale points in hz units"`
	MelPtsBin        etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel_n_filters_eff] mel scale points in fft bins"`
	MelFilterMaxBins uint32          `desc:"#READ_ONLY #NO_SAVE maximum number of bins for mel filter -- number of bins in highest filter"`
	MelFilters       etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel_filt_max_bins][mel.n_filters] the actual filters for actual number of mel filters"`

	Gabor1Filters etensor.Float32 `desc:"#READ_ONLY #NO_SAVE full gabor filters"`
	Gabor2Filters etensor.Float32 `desc:"#READ_ONLY #NO_SAVE full gabor filters"`
	Gabor3Filters etensor.Float32 `desc:"#READ_ONLY #NO_SAVE full gabor filters"`

	// Outputs
	FirstStep     bool        `desc:"#READ_ONLY #NO_SAVE #SHOW is this the first step of processing -- turns of prv smoothing of dft power"`
	InputPos      uint32      `desc:"#READ_ONLY #NO_SAVE #SHOW current position in the sound_full input -- in terms of sample number"`
	TrialStartPos uint32      `desc:" #READ_ONLY #NO_SAVE #SHOW starting position of the current trial -- in terms of sample number"`
	TrialEndPos   uint32      `desc:"#READ_ONLY #NO_SAVE #SHOW ending position of the current trial -- in terms of sample number"`
	XYNGeom       gabor1_geom // #CONDSHOW_ON_gabor1.on #READ_ONLY #SHOW overall geometry of gabor1 output (group-level geometry -- feature / unit level geometry is n_features, 2)
	XYNGeom       gabor2_geom // #CONDSHOW_ON_gabor2.on #READ_ONLY #SHOW overall geometry of gabor1 output (group-level geometry -- feature / unit level geometry is n_features, 2)
	XYNGeom       gabor3_geom // #CONDSHOW_ON_gabor3.on #READ_ONLY #SHOW overall geometry of gabor1 output (group-level geometry -- feature / unit level geometry is n_features, 2)

	SoundFull etensor.Float32 `desc:"#READ_ONLY #NO_SAVE the full sound input obtained from the sound input"`
	WindowIn  etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [input.win_samples] the raw sound input, one channel at a time"`

	DftOut etensor.ComplexFloat32 `desc:"#READ_ONLY #NO_SAVE [2, dft_size] discrete fourier transform (fft) output complex representation"`

	DftPowerOut         etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [dft_use] power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPowerOut      etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [dft_use] log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftPowerTrialOut    etensor.Float32 `desc"#READ_ONLY #NO_SAVE [dft_use][input.total_steps][input.channels] full trial's worth of power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	DftLogPowerTrialOut etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [dft_use][input.total_steps][input.channels] full trial's worth of log power of the dft, up to the nyquist limit frequency (1/2 input.win_samples)"`
	MelFBankOut         etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel.n_filters] mel scale transformation of dft_power, using triangular filters, resulting in the mel filterbank output -- the natural log of this is typically applied"`
	MelFBankTrialOut    etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [mel.n_filters][input.total_steps][input.channels] full trial's worth of mel feature-bank output -- only if using gabors"`
	GaborGci            etensor.Float32 `desc:"#READ_ONLY #NO_SAVE inhibitory conductances, for computing kwta"`
	Gabor1TrialRaw      etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] raw output of gabor1 -- full trial's worth of gabor steps"`
	Gabor1TrialOut      etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] post-kwta output of full trial's worth of gabor steps"`
	Gabor2TrialRaw      etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] raw output of gabor1 -- full trial's worth of gabor steps"`
	Gabor2TrialOut      etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] post-kwta output of full trial's worth of gabor steps"`
	Gabor3TrialRaw      etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] raw output of gabor1 -- full trial's worth of gabor steps"`
	Gabor3TrialOut      etensor.Float32 `desc:"#READ_ONLY #NO_SAVE [gabor.n_filters*2][mel.n_filters][input.trial_steps][input.channels] post-kwta output of full trial's worth of gabor steps"`
	MfccDctOut          etensor.Float32 `desc:"#READ_ONLY #NO_SAVE discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
	MfccDctTrialOut     etensor.Float32 `desc:"#READ_ONLY #NO_SAVE full trial's worth of discrete cosine transform of the log_mel_filter_out values, producing the final mel-frequency cepstral coefficients"`
}

func (ap *AuditoryProc) LoadSound(snd *Sound) bool {
	var needsInit = false
	if ap.NeedsInit() {
		needsInit = true
	}

	if snd == nil || !snd.IsValid() {
		fmt.Printf("LoadSound: sound nil or invalid")
		return false
	}

	if snd.SampleRate() != ap.Input.SampleRate {
		fmt.Printf("LoadSound: sample rate does not match sound -- re-initializing with new rate of: %v", strconv.Itoa(int(snd.SampleRate())))
		ap.Input.SampleRate = snd.SampleRate()
		needsInit = true
	}

	if needsInit {
		ap.Init()
	}

	if ap.Input.Channels > 1 {
		snd.SoundToMatrix(soundFull, -1)
	} else {
		snd.SoundToMatrix(soundFull, Input.Channel)
	}
	snd.StartNewSound()
	return true
}

func (ap *AuditoryProc) StartNewSound() bool {
	ap.FirstStep = true
	ap.InputPos = 0
	ap.TrialStartPos = 0
	ap.TrialEndPos = ap.TrialStartPos + ap.Input.TrialSamples
	ap.DftPowerTrialOut.InitVals(0.0)
	if ap.Dft.LogPow {
		ap.DftLogPowerTrialOut.InitVals(0.0)
	}

	if ap.MelFBank.On {
		ap.MelFBankTrialOut.InitVals(0.0)
		if ap.Gabor1.On {
			ap.Gabor1TrialRaw.InitVals(0.0)
			ap.Gabor1TrialOut.InitVals(0.0)
		}
		if ap.Gabor2.On {
			ap.Gabor2TrialRaw.InitVals(0.0)
			ap.Gabor2TrialOut.InitVals(0.0)

		}
		if ap.Gabor3.On {
			ap.Gabor3TrialRaw.InitVals(0.0)
			ap.Gabor3TrialOut.InitVals(0.0)
		}
		if ap.Mfcc.On {
			ap.MfccDctTrialOut.InitVals(0.0)
		}
	}
	return true
}

func (ap *AuditoryProc) NeedsInit() bool {
	if ap.DftSize != ap.Input.WinSamples || ap.MelNFiltersEff != ap.MelFBank.NFilters+2 {
		return true
	}
	return false

}

func (ap *AuditoryProc) Init() bool {
	ap.UpdateConfig()
	ap.InitFilters()
	ap.InitOutMatrix()
	ap.InitDataTable()
	ap.InitSound()
	return true
}
