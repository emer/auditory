// Package synthcvs contains consonant vowel names and timing information for the synthesized speech generated
// with gnuspeech. These sounds are similar to the ones used by Saffran, Aslin & Newport,
// "Statistical Learning by 8-Month-Old Infants", 1996

package synthcvs

import (
	"bufio"
	"fmt"
	"github.com/emer/auditory/speech"
	"log"
	"os"
	"strconv"
	"strings"
)

// !! really 3 groups of 4, first, second and third position of trisyllabic word - keep the order!!
// The CVs_I set are those used by Saffran, Aslin & Newport
// Sets III-VI include additional consonant vowel combinations
var CVs_I = []string{"da", "go", "pa", "ti", "ro", "la", "bi", "bu", "pi", "tu", "ku", "do"}

var CVs_III = []string{"su", "ro", "pa", "ho", "ba", "lu", "go", "li", "hi", "ra", "di", "sa"}
var CVs_IV = []string{"do", "na", "hu", "ki", "ka", "to", "mo", "mu", "ru", "si", "ta", "po"}
var CVs_V = []string{"gu", "ma", "bi", "bu", "ri", "gi", "tu", "ni", "ha", "so", "ga", "bo"}
var CVs_VI = []string{"da", "ti", "nu", "lo", "ku", "no", "pi", "du", "mi", "pu", "ko", "la"}

// LoadCVSeq reads in a list of cv strings for decoding a particular sequence and returns a slice of strings
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
func LoadTimes(fn string, names []string) ([]speech.SpeechUnit, error) {
	//fmt.Println("LoadCVTimes")
	var units []speech.SpeechUnit
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
		cvt := new(speech.SpeechUnit)
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
	var cvs []string
	switch id {
	case "I":
		cvs = CVs_I
	case "III":
		cvs = CVs_III
	case "IV":
		cvs = CVs_IV
	case "V":
		cvs = CVs_V
	case "VI":
		cvs = CVs_VI
	default:
		fmt.Println("IndexFromCV: Error - fell through CV switch")
	}
	for i, cv := range cvs {
		if s == cv {
			val = i
			ok = true
		}
	}
	return val, ok
}

// SndFromIndex returns the sound if found in the slice of sounds of the corpus.
// id is ignored if the corpus doesn't have subsets of sounds
func SndFmIdx(idx int, id string) (cv string, ok bool) {
	cv = ""
	ok = false
	var cvs []string
	switch id {
	case "I":
		cvs = CVs_I
	case "III":
		cvs = CVs_III
	case "IV":
		cvs = CVs_IV
	case "V":
		cvs = CVs_V
	case "VI":
		cvs = CVs_VI
	default:
		fmt.Println("CVFromIndex: Error - fell through CV switch")
	}
	if idx >= 0 && idx < len(cvs) {
		cv = cvs[idx]
		ok = true
	}
	return cv, ok
}
