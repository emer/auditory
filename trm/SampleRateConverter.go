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
const Beta = float32(5.658)
const IZeroEpsilon = 1E-21

// Sample rate conversion constants
const ZeroCrossings = 13              // SRC CUTOFF FRQ
const LpCutoff = float32(11.0 / 13.0) // 0.846 OF NYQUIST
const FilterLength = ZeroCrossings * LRange

//const N_BITS                    16
const LBits = 8
const LRange = 256 // must be 2^L_BITS
const MBits = 8
const MRange = 256 // must be 2^M_BITS
const FractionBits = LBits + MBits
const FractionRange = 65536 // must be 2^FRACTION_BITS
const FilterLimit = FilterLength - 1

const NMask uint32 = 0xFFFF0000
const LMask uint32 = 0x0000FF00
const MMask uint32 = 0x000000FF
const FractionMask uint32 = 0x0000FFFF
const BufferSize = 1024 // ring buffer size

type SampleRateConverter struct {
	SampleRateRatio       float32
	FillPtr               uint32
	EmptyPtr              uint32
	PadSize               uint32
	FillSize              uint32
	TimeRegisterIncrement uint
	FilterIncrement       uint
	PhaseIncrement        uint
	TimeRegister          uint32
	FillCounter           uint32
	MaximumSampleValue    float64
	NumberSamples         int64

	H      [FilterLength]float32
	DeltaH [FilterLength]float32
	Buffer [BufferSize]float32
	// ToDo - is this correct?
	OutputData []float32
	//std::vector<float>& outputData_;
}

func (src *SampleRateConverter) Init(sampleRate int, outputRate int, outputData *[]float32) {
	src.OutputData = append(src.OutputData, *outputData...)
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
		src.PadSize = uint32(float64(ZeroCrossings)/roundedSampleRateRatio) + 1
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

// DataFill fills the ring buffer with a single sample, increments the counters and pointers,
// and empties the buffer when full
func (src *SampleRateConverter) DataFill(data float32) {
	src.Buffer[src.FillPtr] = data
	SrIncrement(&src.FillPtr, BufferSize)
	src.FillCounter += 1
	if src.FillCounter >= src.FillSize {
		src.DataEmpty()
		src.FillCounter = 0
	}
}

// DataEmpty converts available portion of the input signal to the new sampling rate,
// and outputs the samples to the sound struct.
func (src *SampleRateConverter) DataEmpty() {
	endPtr := src.FillPtr - src.PadSize

	if endPtr < 0 {
		endPtr += BufferSize
	}
	if endPtr < src.EmptyPtr {
		endPtr += BufferSize
	}
	// upsample loop (slightly more efficient than downsampling
	if src.SampleRateRatio >= 1.0 {
		for src.EmptyPtr < endPtr {
			output := float32(0.0)
			interpolation := float32(MValue(src.TimeRegister)) / float32(MRange)

			// compute the left side of the filter convolution
			index := src.EmptyPtr
			for fidx := LValue(src.TimeRegister); fidx < FilterLength; fidx += uint32(src.FilterIncrement) {
				SrDecrement(&index, BufferSize)
				output += (src.Buffer[index]*src.H[fidx] + (src.DeltaH[fidx] * interpolation))
			}

			// adjust values for right side calculation
			src.TimeRegister ^= src.TimeRegister // inverse of each bit
			interpolation = float32(MValue(src.TimeRegister)) / float32(MRange)

			// compute the right side of the filter convolution
			index = src.EmptyPtr
			SrIncrement(&index, BufferSize)
			for fidx := LValue(src.TimeRegister); fidx < FilterLength; fidx += uint32(src.FilterIncrement) {
				SrDecrement(&index, BufferSize)
				output += (src.Buffer[index]*src.H[fidx] + (src.DeltaH[fidx] * interpolation))
			}

			// record maximum sample value
			absoluteSampleValue := math.Abs(float64(output))
			if absoluteSampleValue > src.MaximumSampleValue {
				src.MaximumSampleValue = absoluteSampleValue
			}

			src.NumberSamples += 1

			// save the sample
			src.OutputData = append(src.OutputData, output)

			// change time register back to original form
			src.TimeRegister ^= src.TimeRegister

			// increment the empty pointer, adjusting it and end pointer
			src.EmptyPtr += NValue(src.TimeRegister)
			if src.EmptyPtr >= BufferSize {
				src.EmptyPtr -= BufferSize
				endPtr -= BufferSize
			}

			// clear n part of time register
			src.TimeRegister &= ^NMask
		}
	}
}

// SrIncrement increments the buffer position keeping it within the range 0 to (modulus - 1)
func SrIncrement(pos *uint32, modulus uint32) {
	*pos += 1
	if *pos >= modulus {
		*pos -= modulus
	}
}

// SrDecrement decrements the buffer position keeping it within the range 0 to (modulus - 1)
func SrDecrement(pos *uint32, modulus uint32) {
	*pos -= 1
	if *pos < 0 {
		*pos += modulus
	}
}

// FlushBuffer pads the buffer with zero samples, and flushes it by converting the remaining samples
func (src *SampleRateConverter) FlushBuffer() {
	for i := 0; i < int(src.PadSize*2); i++ {
		src.DataFill(0.0)
	}
	src.DataEmpty()
}

func NValue(x uint32) uint32 {
	y := x
	y &= NMask
	y = y >> FractionBits
	return y
}

func LValue(x uint32) uint32 {
	y := x
	y &= LMask
	y = y >> MBits
	return y
}

func MValue(x uint32) uint32 {
	y := x
	y &= MMask
	return y
}

func FractionValue(x uint32) uint32 {
	y := x
	y &= FractionMask
	return y
}
