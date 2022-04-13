// Copyright (c) 2021, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package vowels should contain the vowel names and timing information for the vowel corpus from Hillenbrand.
// See Hillenbrand, J., Getty, L. A., Clark, M. J., & Wheeler, K. (1995). Acoustic characteristics of American English vowels. The Journal of the Acoustical society of America, 97(5), 3099-3111.
// See Hillenbrand, J. M., Clark, M. J., & Nearey, T. M. (2001). Effects of consonant environment on vowel formant patterns. The Journal of the Acoustical Society of America, 109(2), 748-763.

// https://homepages.wmich.edu/~hillenbr/voweldata.html has the wav files and the documentation

package vowels

import (
	"bufio"
	"github.com/emer/auditory/speech"
	"log"
	"os"
	"strconv"
	"strings"
)

// Cats are the categories, in this case vowel sounds
var Cats = []string{"ae", "ah", "aw", "eh", "ei", "er", "ih", "iy", "oa", "oo", "uh", "uw"}

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
	for i, cv := range Cats {
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
	if idx >= 0 && idx < len(Cats) {
		cv = Cats[idx]
		ok = true
		return
	}
	return
}
