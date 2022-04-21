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
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/emer/auditory/speech"
)

// PhoneList is the full list of phones. Some phones get folded together and the reduced set is the PhoneCats variable.
var PhoneList = []string{"iy", "ih", "eh", "ae", "ix", "ah", "ax", "ax-h", "uw", "ux", "uh", "ao", "aa", "ey",
	"ay", "oy", "aw", "ow", "l", "el", "r", "y", "w", "er", "axr", "m", "em", "n", "nx", "en", "ng",
	"eng", "ch", "jh", "dh", "b", "d", "dx", "g", "p", "t", "k", "z", "zh", "v", "f", "th", "s", "sh",
	"hh", "hv", "pcl", "tcl", "kcl", "bcl", "dcl", "gcl", "epi", "h#", "pau"}

var PhoneCats = []string{"iy", "ih", "eh", "ae", "ix", "ah", "ax", "ax-h", "uw", "ux", "uh", "ao", "aa", "ey",
	"ay", "oy", "aw", "ow", "l", "el", "r", "y", "w", "er", "axr", "m", "em", "n", "nx", "en", "ng",
	"eng", "ch", "jh", "dh", "b", "d", "dx", "g", "p", "t", "k", "z", "zh", "v", "f", "th", "s", "sh",
	"hh", "hv", "pcl", "tcl", "kcl", "bcl", "dcl", "gcl", "epi", "h#", "pau"}

// PhoneCats and PhoneMap must maintain same order!
//var PhoneCats = []string{"iy", "ih", "eh", "ae", "ix", "ah", "uw", "ao", "ey",
//	"ay", "ow", "l", "r", "y", "w", "er", "m", "n", "ch", "jh", "dh", "b", "d", "dx",
//	"p", "t", "k", "z", "zh", "v", "f", "th", "s", "hh", "pcl", "q"}

// PhoneCats2 is a reduced set of phones. PhoneCats2 and Phones2 must maintain same order!
//var PhoneCats2 = []string{"iy", "ih", "eh", "ae", "ix", "ah", "uw", "uh", "ao", "ey",
//	"ay", "oy", "aw", "ow", "l", "r", "y", "w", "er", "m", "n", "ng",
//	"ch", "jh", "dh", "b", "d", "dx", "g", "p", "t", "k", "z", "zh", "v", "f", "th", "s",
//	"hh", "pcl", "q"}

var Phones = map[string]int{
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
	"sh":   44,
	"v":    45,
	"f":    46,
	"th":   47,
	"s":    48,
	"hh":   49,
	"hv":   50,
	"pcl":  51,
	"tcl":  52,
	"kcl":  53,
	"bcl":  54,
	"dcl":  55,
	"gcl":  56,
	"h#":   57,
	"pau":  58,
	"epi":  59,
	"q":    60,
}

//var Phones = map[string]int{
//	"iy":   0,
//	"ih":   1,
//	"eh":   2,
//	"ae":   3,
//	"ix":   4,
//	"ah":   5,
//	"ax":   5,
//	"ax-h": 5,
//	"uw":   6,
//	"ux":   6,
//	"uh":   7,
//	"ao":   8,
//	"aa":   8,
//	"ey":   9,
//	"ay":   10,
//	"oy":   11,
//	"aw":   12,
//	"ow":   13,
//	"l":    14,
//	"el":   14,
//	"r":    15,
//	"y":    16,
//	"w":    17,
//	"er":   18,
//	"axr":  18,
//	"m":    19,
//	"em":   19,
//	"n":    20,
//	"nx":   20,
//	"en":   20,
//	"ng":   21,
//	"eng":  21,
//	"ch":   22,
//	"jh":   23,
//	"dh":   24,
//	"b":    25,
//	"d":    26,
//	"dx":   27,
//	"g":    28,
//	"p":    29,
//	"t":    30,
//	"k":    31,
//	"z":    32,
//	"zh":   33,
//	"sh":   33,
//	"v":    34,
//	"f":    35,
//	"th":   36,
//	"s":    37,
//	"hh":   38,
//	"hv":   38,
//	"pcl":  39,
//	"tcl":  39,
//	"kcl":  39,
//	"bcl":  39,
//	"dcl":  39,
//	"gcl":  39,
//	"h#":   39,
//	"pau":  39,
//	"epi":  39,
//	"q":    40,
//}

var Phones2 = map[string]int{
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
	//"uh":   7,
	"ao": 7,
	"aa": 7,
	"ey": 8,
	"ay": 9,
	//"oy":   11,
	//"aw":  12,
	"ow":  10,
	"l":   11,
	"el":  11,
	"r":   12,
	"y":   13,
	"w":   14,
	"er":  15,
	"axr": 15,
	"m":   16,
	"em":  16,
	"n":   17,
	"nx":  17,
	"en":  17,
	//"ng":  21,
	//"eng": 21,
	"ch": 18,
	"jh": 19,
	"dh": 20,
	"b":  21,
	"d":  22,
	"dx": 23,
	//"g":   28,
	"p":   24,
	"t":   25,
	"k":   26,
	"z":   27,
	"zh":  28,
	"sh":  28,
	"v":   29,
	"f":   30,
	"th":  31,
	"s":   32,
	"hh":  33,
	"hv":  33,
	"pcl": 34,
	"tcl": 34,
	"kcl": 34,
	"bcl": 34,
	"dcl": 34,
	"gcl": 34,
	"h#":  34,
	"pau": 34,
	"epi": 34,
	"q":   35,
}

// IdxFmSnd returns the slice index of the snd if found.
// id is ignored if the corpus doesn't have subsets of sounds
func IdxFmSnd(s string, id string) (v int, ok bool) {
	v, ok = Phones[s]
	return
}

// IdxFmSnd returns the slice index of the snd if found.
// id is ignored if the corpus doesn't have subsets of sounds
func IdxFmSnd2(s string, id string) (v int, ok bool) {
	v, ok = Phones2[s]
	return
}

// SndFmIdx returns the sound if found in the map of sounds of the corpus.
// id is ignored if the corpus doesn't have subsets of sounds
func SndFmIdx(idx int, id string) (phone string, ok bool) {
	phone = ""
	ok = false
	for k, v := range Phones {
		if v == idx {
			phone = k
			ok = true
			return
		}
	}
	return
}

// SndFmIdx returns the sound if found in the map of sounds of the corpus.
// id is ignored if the corpus doesn't have subsets of sounds
func SndFmIdx2(idx int, id string) (phone string, ok bool) {
	phone = ""
	ok = false
	for k, v := range Phones2 {
		if v == idx {
			phone = k
			ok = true
			return
		}
	}
	return
}

// LoadTranscription is a "no op" for timit, LoadTimes does the work of both
func LoadTranscription(fn string) ([]string, error) {
	var names []string
	return names, nil
}

// LoadTimes loads both the timing and transcription data for timit files so the names slice is unused
func LoadTimes(fn string, names []string) ([]speech.Unit, error) {
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
	for scanner.Scan() {
		t := scanner.Text()
		if t == "" {
			break
		}
		cvt := new(speech.Unit)
		units = append(units, *cvt)
		cvs := strings.Fields(t)
		f, err := strconv.ParseFloat(cvs[0], 64)
		if err == nil {
			units[i].Start = f
		}
		if cvs[1] == "h#" {
			units[i].Silence = true
		}
		if len(units) > 1 {
			if cvs[1] == "h#" { // tail silence - set unknown end as start plus one
				units[i].End = units[i].Start + 1
			}
			units[i-1].End = units[i].Start // all units up till final silence
		}
		units[i].Name = cvs[1] //
		i++
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
