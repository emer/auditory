# auditory

Auditory is the our repository for audition processing code in Go (golang) focused on filtering speech wav files via mel filters. A further step using gabors provides filtering for input to neural networks. The processing code is split into 4 packages, sound, mel, dft and agabor, that can be used independently. Example code is in examples/processspeech.

# Packages

**dft**
- The 'dft' package does a fourier transform and computes the power spectrum on the sound samples passed in.

**mel**
- The 'mel' package creates a set of mel filter banks and applies them to the power data to create a spectrogram.

**agabor**
- The 'agabor' package produces an edge detector that detects oriented contrast transitions between light and dark which can be convolved with the output of the mel processing.
- There are 2 structs, FilterSet and Filter. You must create a FilterSet even if you are only adding one gabor Filter

**sound**
- sound.go contains code for loading a wav file into a buffer and then converting to a floating point tensor. There are functions for trimming and padding.
- sndenv.go is a higher level api that has code to process a sound in segments calling the sound code, mel code and gabor code
- playwav.go can be called to play a wav file

**speech**
- speech package has structs for SpeechSequence and SpeechUnit
- packages for specific sound sets (corpora) include code to load these sound files with timing information and lookup code.
  - Package timit Phones of the TIMIT database. See Speaker-Independent Phone Recognition Using Hidden Markov Models, Kai-Fu Lee and Hsiao-Wuen Hon in IEEE Transactions on Acoustics, Speech and Signal Processing, Vol 37, 1989
  - Package grafestes contains the consonant vowel names and timing information for the sound sequences used for the research reported in "Listening Through Voices: Infant Statistical Word Segmentation Across Multiple Speakers", Katherine Graf Estes & Lew-Williams, 2015.
  - Package synthcvs contains consonant vowel names and timing information for the synthesized speech generated with gnuspeech. These sounds are similar to the ones used by Saffran, Aslin & Newport, "Statistical Learning by 8-Month-Old Infants", 1996

