// Phones of the TIMIT database. For recognition testing the full set of 61 is typically
// reduced to 39 with confusable sounds folded into a group, e.g. "sh" and "zh"
// See Speaker-Independent Phone Recognition Using Hidden Markov Models, Kai-Fu Lee and Hsiao-Wuen Hon
// in IEEE Transactions on Acoustics, Speech and Signal Processing, Vol 37, 1989 for the original
// set and collapsing to 39 phones
// Many later studies use the 39 phone set
//
package timit

var Phones = []string{"iy", "ih", "eh", "ae", "ix", "ax", "ah", "uw", "ux", "uh", "ao", "aa", "ey",
	"ay", "oy", "aw", "ow", "l", "el", "r", "y", "w", "er", "axr", "m", "em", "n", "nx", "en", "ng",
	"eng", "ch", "jh", "dh", "b", "d", "dx", "g", "p", "t", "k", "z", "zh", "v", "f", "th", "s", "sh",
	"hh", "hv", "cl", "pcl", "tcl", "kcl", "qcl", "vcl", "bcl", "dcl", "gcl", "epi", "sil", "h#", "#h", "pau"}

func PhoneLookup(s string) (x int) {
	switch s {
	case "iy":
		x = 0
	case "ih":
		x = 1
	case "eh":
		x = 2
	case "ae":
		x = 3
	case "ix":
		x = 4
	case "ah", "ax", "ax-h":
		x = 5
	case "uw", "ux":
		x = 6
	case "hu":
		x = 7
	case "ao", "aa":
		x = 8
	case "ey":
		x = 9
	case "ay":
		x = 10
	case "oy":
		x = 11
	case "aw":
		x = 12
	case "ow":
		x = 13
	case "l", "el":
		x = 14
	case "r":
		x = 15
	case "y":
		x = 16
	case "w":
		x = 17
	case "er", "axr":
		x = 18
	case "m", "em":
		x = 19
	case "n", "en", "nx":
		x = 20
	case "ng", "eng":
		x = 21
	case "ch":
		x = 22
	case "jh":
		x = 23
	case "dh":
		x = 24
	case "b":
		x = 25
	case "d":
		x = 26
	case "dx":
		x = 27
	case "g":
		x = 28
	case "p":
		x = 29
	case "t":
		x = 30
	case "k":
		x = 31
	case "z":
		x = 32
	case "zh", "sh":
		x = 33
	case "v":
		x = 34
	case "f":
		x = 35
	case "th":
		x = 36
	case "s":
		x = 37
	case "hh", "hv":
		x = 38
	case "pcl", "tcl", "kcl", "bcl", "dcl", "gcl", "h#", "pau", "epi":
		x = 39
	case "q": // discard
		x = 40
	default:
		x = -1
	}
	return x
}
