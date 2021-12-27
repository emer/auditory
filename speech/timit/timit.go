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

var PhoneList = []string{"iy", "ih", "eh", "ae", "ix", "ah", "ax", "ax-h", "uw", "ux", "uh", "ao", "aa", "ey",
	"ay", "oy", "aw", "ow", "l", "el", "r", "y", "w", "er", "axr", "m", "em", "n", "nx", "en", "ng",
	"eng", "ch", "jh", "dh", "b", "d", "dx", "g", "p", "t", "k", "z", "zh", "v", "f", "th", "s", "sh",
	"hh", "hv", "cl", "pcl", "tcl", "kcl", "qcl", "vcl", "bcl", "dcl", "gcl", "epi", "sil", "h#", "#h", "pau"}

var PhoneCats = []string{"iy", "ih", "eh", "ae", "ix", "ah", "uw", "uh", "ao", "ey",
	"ay", "oy", "aw", "ow", "l", "r", "y", "w", "er", "m", "n", "ng",
	"ch", "jh", "dh", "b", "d", "dx", "g", "p", "t", "k", "z", "zh", "v", "f", "th", "s",
	"hh", "pcl", "q"}

var Phones = map[string]int{
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

// ReMap handles the mapping of mulitple sounds to a single category as some sounds are highly confusable
// and the research community follows a protocol of condensing the full set of phones down to 39, see comment at top of file.
// id is ignored if the corpus doesn't have subsets of sounds
//func ReMap(s string, id string) (v int, ok bool) {
//	v, ok = Phones[s]
//	return
//}

// IdxFmSnd returns the slice index of the snd if found.
// id is ignored if the corpus doesn't have subsets of sounds
func IdxFmSnd(s string, id string) (v int, ok bool) {
	v, ok = Phones[s]
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

// LoadTranscriptionAndTimes loads the timing and transcription data for timit files
func LoadTranscriptionAndTimes(fn string) ([]speech.SpeechUnit, error) {
	//fmt.Println("LoadTimitSeqsAndTimes")
	var units []speech.SpeechUnit

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
		if strings.Contains(t, "h#") { // silence at start or silence at end
			if len(units) == 0 { // starting silence
				continue
			} else { // silence at end
				cvs := strings.Fields(t)
				f, _ := strconv.ParseFloat(cvs[0], 64)
				units[i-1].End = f
				break // we're done!
			}
		}
		cvt := new(speech.SpeechUnit)
		units = append(units, *cvt)
		cvs := strings.Fields(t)
		f, err := strconv.ParseFloat(cvs[0], 64)
		if err == nil {
			units[i].Start = f
		}
		if len(units) > 1 {
			units[i-1].End = units[i].Start
		}
		units[i].Name = cvs[1] //
		i++
	}
	return units, nil
}
