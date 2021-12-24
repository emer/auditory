package speech

// SpeechUnit
type SpeechUnit struct {
	Name   string  `desc:"the CV (e.g. -- da, go, ku ...), or phones (g, ah, ix ...)"`
	Start  float64 `desc:"start time of this unit in a particular sequence in milliseconds"`
	End    float64 `desc:"end time of this unit in a particular sequence in milliseconds"`
	AStart float64 `desc:"start time of this unit in a particular sequence in milliseconds, adjusted for random start silence and any offset in audio"`
	AEnd   float64 `desc:"end time of this unit in a particular sequence in milliseconds, adjusted for random start silence and any offset in audio"`
	Type   string  `desc:"optional info - type of unit, phone, phoneme, word, CV (consonsant-vowel), etc"`
}
