# auditory

Auditory is the our repository for audition processing code in Go (golang) focused on filtering speech wav files via mel filters. A further step using gabors provides filtering for input to neural networks. The processing code is split into 4 packages, sound, mel, dft and agabor, that can be used independently. A fifth package, trm, is a work in progress port of Gnuspeech. Example code is in examples/processspeech.

The `sound` package contains code for loading a wav file into a buffer and then converting to a floating point tensor. There are functions for trimming and padding.

The 'dft' package does a fourier transform and computes the power spectrum on the sound samples passed in.

The 'mel' package creates a set of mel filter banks and applies them to the power data to create a spectrogram.

The 'agabor' package produces an edge detector that detects oriented contrast transitions between light and dark which can be convolved with the output of the mel processing.
