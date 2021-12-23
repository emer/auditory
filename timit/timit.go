// Phones of the TIMIT database. For recognition testing the full set of 61 is typically
// reduced to 39 with confusable sounds folded into a group, e.g. "sh" and "zh"
// See Speaker-Independent Phone Recognition Using Hidden Markov Models, Kai-Fu Lee and Hsiao-Wuen Hon
// in IEEE Transactions on Acoustics, Speech and Signal Processing, Vol 37, 1989 for the original
// set and collapsing to 39 phones
// Many later studies use the 39 phone set
//
package timit

var PhoneList = []string{"iy", "ih", "eh", "ae", "ix", "ax", "ah", "uw", "ux", "uh", "ao", "aa", "ey",
	"ay", "oy", "aw", "ow", "l", "el", "r", "y", "w", "er", "axr", "m", "em", "n", "nx", "en", "ng",
	"eng", "ch", "jh", "dh", "b", "d", "dx", "g", "p", "t", "k", "z", "zh", "v", "f", "th", "s", "sh",
	"hh", "hv", "cl", "pcl", "tcl", "kcl", "qcl", "vcl", "bcl", "dcl", "gcl", "epi", "sil", "h#", "#h", "pau"}

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
	"hu":   7,
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
	"en":   20,
	"nx":   20,
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

func PhoneLookup(s string) (val int, ok bool) {
	val, ok = Phones[s]
	return val, ok
}
