// Copyright (c) 2021, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package grafestes contains the consonant vowel names and timing information for the sound sequences used for the
// research reported in "Listening Through Voices: Infant Statistical Word Segmentation Across Multiple Speakers",
// Katherine Graf Estes & Lew-Williams, 2015.
// The sounds are spoken consonant-vowels that were spliced together from eight (?) women.
// See the paper for the details on how the sequences were contructed

package grafestes

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/emer/auditory/speech"
)

var CVs = []string{"ti", "do", "ga", "mo", "may", "bu", "pi", "ku"}
var CVsPerWord = 2 // The graf-estes experiment used 2 syllable words
var CVsPerPos = 4  // The graf-estes experiment had 4 cv possibilities per syllable position

// LoadTranscription reads in a list of cv strings for decoding a particular sequence and returns a slice of strings
func LoadTranscription(fn string) ([]string, error) {
	//fmt.Println
	var names []string
	fp2, err := os.Open(fn)
	if err != nil {
		log.Println(err)
		return names, err
	}
	defer fp2.Close() // we will be done with the file within this function
	scanner2 := bufio.NewScanner(fp2)
	scanner2.Split(bufio.ScanLines)
	s := ""
	for scanner2.Scan() {
		s = scanner2.Text()
	}
	names = strings.Split(s, " ")
	return names, nil
}

// LoadTimes loads the timing and sequence (transcription) data for CV files
func LoadTimes(fn string, names []string) ([]speech.Unit, error) {
	//fmt.Println("LoadCVTimes")
	var units []speech.Unit
	fp, err := os.Open(fn)
	if err != nil {
		log.Println(err)
		log.Println("Make sure you have the sound files rsyncd to your ccn_images directory and a link (ln -s) to ccn_images in your sim working directory")
		return units, err
	}
	defer fp.Close() // we will be done with the file within this function

	scanner := bufio.NewScanner(fp)
	scanner.Split(bufio.ScanLines)

	i := 0
	for scanner.Scan() {
		t := scanner.Text()
		if t == "" {
			break
		} else if strings.HasPrefix(t, "\\") { // lines starting with '/' are lines with frequency for start/end points
			continue
		}
		cvt := new(speech.Unit)
		units = append(units, *cvt)
		cvs := strings.Fields(t)
		f, err := strconv.ParseFloat(cvs[0], 64)
		if err == nil {
			(units)[i].Start = f * 1000 // convert to milliseconds
		}
		f, err = strconv.ParseFloat(cvs[1], 64)
		if err == nil {
			(units)[i].End = f * 1000 // convert to milliseconds
		}
		(units)[i].Name = names[i]
		i++
		if i == len(names) {
			return units, nil
		} // handles case where there may be lines after last line of start, end, name
	}
	return units, nil
}

// IdxFmSnd returns the slice index of the snd if found.
// id is ignored if the corpus doesn't have subsets of sounds
func IdxFmSnd(s string, id string) (val int, ok bool) {
	val = -1
	ok = false
	for i, cv := range CVs {
		if s == cv {
			val = i
			ok = true
			return
		}
	}
	return
}

// SndFmIdx returns the sound if found in the slice of sounds of the corpus.
// id is ignored if the corpus doesn't have subsets of sounds
func SndFmIdx(idx int, id string) (cv string, ok bool) {
	cv = ""
	ok = false
	if idx >= 0 && idx < len(CVs) {
		cv = CVs[idx]
		ok = true
		return
	}
	return
}
