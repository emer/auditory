// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/***************************************************************************
 *  Copyright 1991, 1992, 1993, 1994, 1995, 1996, 2001, 2002               *
 *    David R. Hill, Leonard Manzara, Craig Schock                         *
 *                                                                         *
 *  This program is free software: you can redistribute it and/or modify   *
 *  it under the terms of the GNU General Public License as published by   *
 *  the Free Software Foundation, either version 3 of the License, or      *
 *  (at your option) any later version.                                    *
 *                                                                         *
 *  This program is distributed in the hope that it will be useful,        *
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of         *
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the          *
 *  GNU General Public License for more details.                           *
 *                                                                         *
 *  You should have received a copy of the GNU General Public License      *
 *  along with this program.  If not, see <http://www.gnu.org/licenses/>.  *
 ***************************************************************************/
// 2014-09
// This file was copied from Gnuspeech and modified by Marcelo Y. Matuda.

// 2019-02
// This is a port of the Gnuspeech port to C++ by Marcelo Y. Matuda

package trm

import "math"

// kaiser window params
const Beta = float32(5.658) // kaiser window parameters
const IZeroEpsilon = 1E-21

// SAMPLE RATE CONVERSION CONSTANTS
const ZeroCrossings = 13              // SRC CUTOFF FRQ
const LpCutoff = float32(11.0 / 13.0) // 0.846 OF NYQUIST
const FilterLength = ZeroCrossings * LRange

//const N_BITS                    16
const LBits = 8
const LRange = 256 /*  must be 2^L_BITS  */
const MBits = 8
const MRange = 256 /*  must be 2^M_BITS  */
const FractionBits = LBits + MBits
const FractionRange = 65536 /*  must be 2^FRACTION_BITS  */
const FilterLimit = FilterLength - 1

const NMask = 0xFFFF0000
const LMask = 0x0000FF00
const MMask = 0x000000FF
const FractionMask = 0x0000FFFF

// ToDo - port these
//const nValue (x)                 (((x) & N_MASK) >> FRACTION_BITS)
//const lValue (x)                 (((x) & L_MASK) >> M_BITS)
//const mValue (x)                 ((x) & M_MASK)
//const fractionValue (x)          ((x) & FRACTION_MASK)

const BufferSize = 1024 /*  ring buffer size  */

type SampleRateConverter struct {
	SampleRateRatio       float32
	FillPtr               int
	EmptyPtr              int
	PadSize               int
	FillSize              int
	TimeRegisterIncrement uint
	FilterIncrement       uint
	PhaseIncrement        uint
	TimeRegister          uint
	FillCounter           int
	MaximumSampleValue    float32
	NumberSamples         int64

	H      [FilterLength]float32
	DeltaH [FilterLength]float32
	Buffer [BufferSize]float32
	// ToDo - is this correct?
	OutputData *[]float32
	//std::vector<float>& outputData_;
}

func (src *SampleRateConverter) Init(sampleRate int, outputRate int, outputData *[]float32) {
	src.OutputData = outputData
	src.InitConversion(sampleRate, float32(outputRate))

}

// SampleRateConverter resets various values of the converter
func (src *SampleRateConverter) Reset() {
	src.EmptyPtr = 0
	src.TimeRegister = 0
	src.FillCounter = 0
	src.MaximumSampleValue = 0.0
	src.NumberSamples = 0

	src.InitBuffer()
}

// InitConversion initializes all the sample rate conversion functions
func (src *SampleRateConverter) InitConversion(sampleRate int, outputRate float32) {
	src.InitFilter() // initialize filter impulse response

	src.SampleRateRatio = outputRate / float32(sampleRate)

	src.TimeRegisterIncrement = uint(math.Round(math.Pow(2.0, float64(FractionBits)) / float64(src.SampleRateRatio)))

	roundedSampleRateRatio := math.Pow(2.0, FractionBits) / float64(src.TimeRegisterIncrement)

	if src.SampleRateRatio >= 1.0 {
		src.FilterIncrement = LRange
	} else {
		src.PhaseIncrement = uint(math.Round(float64(src.SampleRateRatio) * FractionRange))
	}

	if src.SampleRateRatio >= 1.0 {
		src.PadSize = ZeroCrossings
	} else {
		src.PadSize = int(float64(ZeroCrossings)/roundedSampleRateRatio) + 1
	}

	src.InitBuffer() // initialize the ring buffer
}

// IZero Returns the value for the modified Bessel function of the first kind, order 0, as a float
func (src *SampleRateConverter) IZero(x float32) float32 {
	var sum float32 = 1.0
	var u float32 = 1.0
	var halfx float32 = x / 2.0

	n := 1

	for {
		temp := halfx / float32(n)
		n += 1
		temp *= temp
		u *= temp
		sum += u
		if u >= IZeroEpsilon*sum {
			break
		}
	}
	return sum
}

// InitBuffer Initializes the ring buffer used for sample rate conversion
func (src *SampleRateConverter) InitBuffer() {
	for i := 0; i < BufferSize; i++ {
		src.Buffer[i] = 0.0
	}

	src.FillPtr = src.PadSize
	src.FillSize = BufferSize - (2 * src.PadSize)
}

// InitFilter Initializes filter impulse response and impulse delta values
func (src *SampleRateConverter) InitFilter() {
	src.H[0] = LpCutoff
	x := math.Pi / float32(LRange)

	// initialize the filter impulse response
	for i := 1; i < FilterLength; i++ {
		y := float32(i) * x
		src.H[i] = float32(math.Sin(float64(y)*float64(LpCutoff))) / y
	}

	// apply a kaiser window to the impulse response
	iBeta := 1.0 / src.IZero(Beta)
	for i := 0; i < FilterLength; i++ {
		temp := float64(i / FilterLength)
		src.H[i] = float32(src.IZero(float32(math.Sqrt(float64(1.0)-(temp*temp))))) * iBeta
	}

	for i := 0; i < FilterLimit; i++ {
		src.DeltaH[i] = src.H[i+1] - src.H[i]
	}
	src.DeltaH[FilterLimit] = 0.0 - src.H[FilterLimit]
}

//
///******************************************************************************
// *
// *  function:  dataFill
// *
// *  purpose:   Fills the ring buffer with a single sample, increments
// *             the counters and pointers, and empties the buffer when
// *             full.
// *
// ******************************************************************************/
//void
//SampleRateConverter::dataFill(float data)
//{
///*  PUT THE DATA INTO THE RING BUFFER  */
//buffer_[fillPtr_] = data;
//
///*  INCREMENT THE FILL POINTER, MODULO THE BUFFER SIZE  */
//srIncrement(&fillPtr_, BUFFER_SIZE);
//
///*  INCREMENT THE COUNTER, AND EMPTY THE BUFFER IF FULL  */
//if (++fillCounter_ >= fillSize_) {
//dataEmpty();
///* RESET THE FILL COUNTER  */
//fillCounter_ = 0;
//}
//}
//
///******************************************************************************
// *
// *  function:  dataEmpty
// *
// *  purpose:   Converts available portion of the input signal to the
// *             new sampling rate, and outputs the samples to the
// *             sound struct.
// *
// ******************************************************************************/
//void
//SampleRateConverter::dataEmpty()
//{
///*  CALCULATE END POINTER  */
//int endPtr = fillPtr_ - padSize_;
//
///*  ADJUST THE END POINTER, IF LESS THAN ZERO  */
//if (endPtr < 0) {
//endPtr += BUFFER_SIZE;
//}
//
///*  ADJUST THE ENDPOINT, IF LESS THEN THE EMPTY POINTER  */
//if (endPtr < emptyPtr_) {
//endPtr += BUFFER_SIZE;
//}
//
///*  UPSAMPLE LOOP (SLIGHTLY MORE EFFICIENT THAN DOWNSAMPLING)  */
//if (sampleRateRatio_ >= 1.0) {
//while (emptyPtr_ < endPtr) {
///*  RESET ACCUMULATOR TO ZERO  */
//float output = 0.0;
//
///*  CALCULATE INTERPOLATION VALUE (STATIC WHEN UPSAMPLING)  */
//float interpolation = (float) mValue(timeRegister_) / (float) M_RANGE;
//
///*  COMPUTE THE LEFT SIDE OF THE FILTER CONVOLUTION  */
//int index = emptyPtr_;
//for (int filterIndex = lValue(timeRegister_);
//filterIndex < FILTER_LENGTH;
//srDecrement(&index, BUFFER_SIZE), filterIndex += filterIncrement_) {
//output += (buffer_[index] *
//(h_[filterIndex] + (deltaH_[filterIndex] * interpolation)));
//}
//
///*  ADJUST VALUES FOR RIGHT SIDE CALCULATION  */
//timeRegister_ = ~timeRegister_;
//interpolation = (float) mValue(timeRegister_) / (float) M_RANGE;
//
///*  COMPUTE THE RIGHT SIDE OF THE FILTER CONVOLUTION  */
//index = emptyPtr_;
//srIncrement(&index,BUFFER_SIZE);
//for (int filterIndex = lValue(timeRegister_);
//filterIndex < FILTER_LENGTH;
//srIncrement(&index, BUFFER_SIZE), filterIndex += filterIncrement_) {
//output += (buffer_[index] *
//(h_[filterIndex] + (deltaH_[filterIndex] * interpolation)));
//}
//
///*  RECORD MAXIMUM SAMPLE VALUE  */
//float absoluteSampleValue = fabs(output);
//if (absoluteSampleValue > maximumSampleValue_) {
//maximumSampleValue_ = absoluteSampleValue;
//}
//
///*  INCREMENT SAMPLE NUMBER  */
//numberSamples_++;
//
///*  SAVE THE SAMPLE  */
//outputData_.push_back(static_cast<float>(output));
//
///*  CHANGE TIME REGISTER BACK TO ORIGINAL FORM  */
//timeRegister_ = ~timeRegister_;
//
///*  INCREMENT THE TIME REGISTER  */
//timeRegister_ += timeRegisterIncrement_;
//
///*  INCREMENT THE EMPTY POINTER, ADJUSTING IT AND END POINTER  */
//emptyPtr_ += nValue(timeRegister_);
//
//if (emptyPtr_ >= BUFFER_SIZE) {
//emptyPtr_ -= BUFFER_SIZE;
//endPtr -= BUFFER_SIZE;
//}
//
///*  CLEAR N PART OF TIME REGISTER  */
//timeRegister_ &= (~N_MASK);
//}
//} else {
///*  DOWNSAMPLING CONVERSION LOOP  */
//
//while (emptyPtr_ < endPtr) {
//
///*  RESET ACCUMULATOR TO ZERO  */
//float output = 0.0;
//
///*  COMPUTE P PRIME  */
//unsigned int phaseIndex = (unsigned int) rint(
//((float) fractionValue(timeRegister_)) * sampleRateRatio_);
//
///*  COMPUTE THE LEFT SIDE OF THE FILTER CONVOLUTION  */
//int index = emptyPtr_;
//unsigned int impulseIndex;
//while ((impulseIndex = (phaseIndex >> M_BITS)) < FILTER_LENGTH) {
//float impulse = h_[impulseIndex] + (deltaH_[impulseIndex] *
//(((float) mValue(phaseIndex)) / (float) M_RANGE));
//output += (buffer_[index] * impulse);
//srDecrement(&index, BUFFER_SIZE);
//phaseIndex += phaseIncrement_;
//}
//
///*  COMPUTE P PRIME, ADJUSTED FOR RIGHT SIDE  */
//phaseIndex = (unsigned int) rint(
//((float) fractionValue(~timeRegister_)) * sampleRateRatio_);
//
///*  COMPUTE THE RIGHT SIDE OF THE FILTER CONVOLUTION  */
//index = emptyPtr_;
//srIncrement(&index, BUFFER_SIZE);
//while ((impulseIndex = (phaseIndex >> M_BITS)) < FILTER_LENGTH) {
//float impulse = h_[impulseIndex] + (deltaH_[impulseIndex] *
//(((float) mValue(phaseIndex)) / (float) M_RANGE));
//output += (buffer_[index] * impulse);
//srIncrement(&index, BUFFER_SIZE);
//phaseIndex += phaseIncrement_;
//}
//
///*  RECORD MAXIMUM SAMPLE VALUE  */
//float absoluteSampleValue = fabs(output);
//if (absoluteSampleValue > maximumSampleValue_) {
//maximumSampleValue_ = absoluteSampleValue;
//}
//
///*  INCREMENT SAMPLE NUMBER  */
//numberSamples_++;
//
///*  SAVE THE SAMPLE  */
//outputData_.push_back(static_cast<float>(output));
//
///*  INCREMENT THE TIME REGISTER  */
//timeRegister_ += timeRegisterIncrement_;
//
///*  INCREMENT THE EMPTY POINTER, ADJUSTING IT AND END POINTER  */
//emptyPtr_ += nValue(timeRegister_);
//if (emptyPtr_ >= BUFFER_SIZE) {
//emptyPtr_ -= BUFFER_SIZE;
//endPtr -= BUFFER_SIZE;
//}
//
///*  CLEAR N PART OF TIME REGISTER  */
//timeRegister_ &= (~N_MASK);
//}
//}
//}
//
///******************************************************************************
// *
// *  function:  srIncrement
// *
// *  purpose:   Increments the pointer, keeping it within the range
// *             0 to (modulus-1).
// *
// ******************************************************************************/
//void
//SampleRateConverter::srIncrement(int *pointer, int modulus)
//{
//if (++(*pointer) >= modulus) {
//(*pointer) -= modulus;
//}
//}
//
///******************************************************************************
// *
// *  function:  srDecrement
// *
// *  purpose:   Decrements the pointer, keeping it within the range
// *             0 to (modulus-1).
// *
// ******************************************************************************/
//void
//SampleRateConverter::srDecrement(int *pointer, int modulus)
//{
//if (--(*pointer) < 0) {
//(*pointer) += modulus;
//}
//}
//
///******************************************************************************
// *
// *  function:  flushBuffer
// *
// *  purpose:   Pads the buffer with zero samples, and flushes it by
// *             converting the remaining samples.
// *
// ******************************************************************************/
//void
//SampleRateConverter::flushBuffer()
//{
///*  PAD END OF RING BUFFER WITH ZEROS  */
//for (int i = 0; i < padSize_ * 2; i++) {
//dataFill(0.0);
//}
//
///*  FLUSH UP TO FILL POINTER - PADSIZE  */
//dataEmpty();
//}
