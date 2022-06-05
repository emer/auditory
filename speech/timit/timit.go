// Copyright (c) 2021, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package timit Phones of the TIMIT database. For recognition testing the full set of 61 is typically
// reduced to 39 with confusable sounds folded into a group, e.g. "sh" and "zh"
// See Speaker-Independent Phone Recognition Using Hidden Markov Models, Kai-Fu Lee and Hsiao-Wuen Hon
// in IEEE Transactions on Acoustics, Speech and Signal Processing, Vol 37, 1989 for the original
// set and collapsing to 39 phones
// Many later studies use the 39 phone set
//
package timit

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/emer/auditory/speech"
)

// Each set of PhoneCats and Phone map must have the same order!!!!

// This is the full set of timit transcribed phones. See Lee & Hon, 1989 - Speaker-Independent Phone Recognition Using Hidden Markov Models
var PhoneCats61 = []string{"iy", "ih", "eh", "ae", "ix", "ah", "ax", "ax-h", "uw", "ux", "uh", "ao", "aa", "ey",
	"ay", "oy", "aw", "ow", "l", "el", "r", "y", "w", "er", "axr", "m", "em", "n", "nx", "en", "ng",
	"eng", "ch", "jh", "dh", "b", "d", "dx", "g", "p", "t", "k", "z", "zh", "v", "f", "th", "s", "sh",
	"hh", "hv", "pcl", "tcl", "kcl", "bcl", "dcl", "gcl", "epi", "h#", "pau", "q"}

// PhoneCats41 is a reduced set of phones. Some phones are generally recognized as confusable and swaps are
// not considered errors in the published phone recognition experiments. Some research does the mapping of
// many to one (e.g. n, nx and en all to "n") when scoring the recognizer output and the PhoneCats61 set is used.
// Using the PhoneCats41 set the mapping down is done during training and test.
var PhoneCats41 = []string{"iy", "ih", "eh", "ae", "ix", "ah", "uw", "uh", "ao", "ey", "ay",
	"oy", "aw", "ow", "l", "r", "y", "w", "er", "m", "n", "ng", "ch", "jh", "dh", "b", "d",
	"dx", "g", "p", "t", "k", "z", "zh", "v", "f", "th", "s", "hh", "pcl", "q"}

// the PhoneCats10 set is a subset of phones that early results showed were more easily recognized
// and the set was used to "begin with success!"
var PhoneCats10 = []string{"ah", "ao", "dh", "er", "ix", "iy", "l", "n", "r", "s"}

var Phones10 = map[string]int{
	"ah": 0,
	"ao": 1,
	"dh": 2,
	"er": 3,
	"ix": 4,
	"iy": 5,
	"l":  6,
	"n":  7,
	"r":  8,
	"s":  9,
}

var Phones41 = map[string]int{
	"iy":   0,
	"ih":   1,
	"eh":   2,
	"ae":   3,
	"ix":   4,
	"ah":   5,
	"ax":   5,
	"ax-h": 5,
	"uw":   6,
	"ux":   6,
	"uh":   7,
	"ao":   8,
	"aa":   8,
	"ey":   9,
	"ay":   10,
	"oy":   11,
	"aw":   12,
	"ow":   13,
	"l":    14,
	"el":   14,
	"r":    15,
	"y":    16,
	"w":    17,
	"er":   18,
	"axr":  18,
	"m":    19,
	"em":   19,
	"n":    20,
	"nx":   20,
	"en":   20,
	"ng":   21,
	"eng":  21,
	"ch":   22,
	"jh":   23,
	"dh":   24,
	"b":    25,
	"d":    26,
	"dx":   27,
	"g":    28,
	"p":    29,
	"t":    30,
	"k":    31,
	"z":    32,
	"zh":   33,
	"sh":   33,
	"v":    34,
	"f":    35,
	"th":   36,
	"s":    37,
	"hh":   38,
	"hv":   38,
	"pcl":  39,
	"tcl":  39,
	"kcl":  39,
	"bcl":  39,
	"dcl":  39,
	"gcl":  39,
	"h#":   39,
	"pau":  39,
	"epi":  39,
	"q":    40,
}

var Phones61 = map[string]int{
	"iy":   0,
	"ih":   1,
	"eh":   2,
	"ae":   3,
	"ix":   4,
	"ah":   5,
	"ax":   6,
	"ax-h": 7,
	"uw":   8,
	"ux":   9,
	"uh":   10,
	"ao":   11,
	"aa":   12,
	"ey":   13,
	"ay":   14,
	"oy":   15,
	"aw":   16,
	"ow":   17,
	"l":    18,
	"el":   19,
	"r":    20,
	"y":    21,
	"w":    22,
	"er":   23,
	"axr":  24,
	"m":    25,
	"em":   26,
	"n":    27,
	"nx":   28,
	"en":   29,
	"ng":   30,
	"eng":  31,
	"ch":   32,
	"jh":   33,
	"dh":   34,
	"b":    35,
	"d":    36,
	"dx":   37,
	"g":    38,
	"p":    39,
	"t":    40,
	"k":    41,
	"z":    42,
	"zh":   43,
	"v":    44,
	"f":    45,
	"th":   46,
	"s":    47,
	"sh":   48,
	"hh":   49,
	"hv":   50,
	"pcl":  51,
	"tcl":  52,
	"kcl":  53,
	"bcl":  54,
	"dcl":  55,
	"gcl":  56,
	"epi":  57,
	"h#":   58,
	"pau":  59,
	"q":    60,
}

// IdxFmSnd returns the slice index of the snd if found.
// id is ignored if the corpus doesn't have subsets of sounds
func IdxFmSnd(s string, id string) (v int, ok bool) {
	v = -1
	ok = false
	if id == "Phones10" {
		v, ok = Phones10[s]
	} else if id == "Phones41" {
		v, ok = Phones41[s]
	} else if id == "Phones61" {
		v, ok = Phones61[s]
	} else {
		fmt.Println("IdxFmSnd: phone set id does not match any existing phone set")
	}
	return
}

// SndFmIdx returns the sound if found in the map of sounds of the corpus.
// id is ignored if the corpus doesn't have subsets of sounds
func SndFmIdx(idx int, id string) (phone string, ok bool) {
	phone = ""
	ok = false
	if id == "Phones10" {
		for k, v := range Phones10 {
			if v == idx {
				phone = k
				ok = true
			}
		}
	} else if id == "Phones41" {
		for k, v := range Phones41 {
			if v == idx {
				phone = k
				ok = true
			}
		}
	} else if id == "Phones61" {
		for k, v := range Phones61 {
			if v == idx {
				phone = k
				ok = true
			}
		}
	} else {
		fmt.Println("IdxFmSnd: phone set id does not match any existing phone set")
	}
	return
}

// LoadTranscription is a "no op" for timit, LoadTimes does the work of both
func LoadTranscription(fn string) ([]string, error) {
	var names []string
	return names, nil
}

// IsStop
func IsStop(s string) bool {
	if s == "b" || s == "d" || s == "g" || s == "k" || s == "p" || s == "t" {
		return true
	}
	return false
}

// LoadTimes loads both the timing and transcription data for timit files so the names slice is unused.
// If fuse is true stop consonants and the paired closure are combined into a single sound entry. The
// duration is the combination of the closure and the consonant (b d g k p t)
func LoadTimes(fn string, names []string, fuse bool) ([]speech.Unit, error) {
	//fmt.Println("LoadTimitSeqsAndTimes")
	var units []speech.Unit

	// load the sound start/end times shipped with the TIMIT database
	fp, err := os.Open(fn)
	if err != nil {
		log.Println(err)
		log.Println("Make sure you have the sound files rsyncd to ccn_images directory and a link (ln -s) to ccn_images in your sim working directory")
		return units, err
	}
	defer fp.Close() // we will be done with the file within this function

	scanner := bufio.NewScanner(fp)
	scanner.Split(bufio.ScanLines)

	i := 0
	prvClosure := false // was the preceding snd a closure?
	closure := ""
	for scanner.Scan() {
		t := scanner.Text()
		if t == "" {
			break
		}
		cvs := strings.Fields(t)
		time := cvs[0]
		snd := cvs[1]

		if prvClosure == false || prvClosure == true && snd != string(closure[0]) {
			prvClosure = false
			closure = ""
			//if prvClosure == false && IsStop(snd) == false {
			cvt := new(speech.Unit)
			units = append(units, *cvt)
			f, err := strconv.ParseFloat(time, 64)
			if err == nil {
				units[i].Start = f
			}

			if fuse == true && strings.HasSuffix(snd, "cl") {
				prvClosure = true
				//fmt.Println("the closure is: ", snd)
				closure = snd
				c := strings.TrimSuffix(snd, "cl")
				units[i].Name = c               // name it with non-closure consonant (i.e. bcl -> b, gcl -> g)
				units[i-1].End = units[i].Start // all units up till final silence
				i++
				continue // skip the rest
			}
			if snd == "h#" {
				units[i].Silence = true
			}

			if len(units) > 1 {
				if snd == "h#" { // tail silence - set unknown end as start plus one
					units[i].End = units[i].Start + 1
				}
				units[i-1].End = units[i].Start // all units up till final silence
			}
			units[i].Name = snd //
			i++
		} else {
			prvClosure = false // reset
			//fmt.Println("prv is closure, current is:", snd)

		}
	}
	return units, nil
}

// LoadText retrieves the full text of the timit transcription
func LoadText(fn string) (string, error) {
	fp, err := os.Open(fn)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer fp.Close() // we will be done with the file within this function

	s := ""
	scanner := bufio.NewScanner(fp)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		s = scanner.Text()
	}
	// format is 'start time' 'space' 'end time' 'space' text
	cutset := "0123456789"
	s = strings.TrimLeft(s, cutset)
	s = strings.TrimLeft(s, " ")
	s = strings.TrimLeft(s, cutset)
	s = strings.TrimLeft(s, " ")
	return s, nil
}
