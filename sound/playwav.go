// Copyright (c) 2021, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sound

import (
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/oto"
	"io"
	"os"
	"runtime"
	"sync"
)

func PlayWav(context *oto.Context, fn string, rate int) error {
	f, err := os.Open(fn)
	if err != nil {
		panic(err)
	}

	s, err := wav.DecodeWithSampleRate(rate, f)
	if err != nil {
		panic(err)
	}
	p := context.NewPlayer()
	if _, err := io.Copy(p, s); err != nil {
		return err
	}
	if err := p.Close(); err != nil {
		return err
	}
	return nil
}

func Play(fn string, rate int, channnels int, bitdepth int) error {
	c, err := oto.NewContext(rate, channnels, bitdepth, 4096)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var players []oto.Player

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := PlayWav(c, fn, rate); err != nil {
			panic(err)
		}
	}()

	wg.Wait()
	// Pin the players not to GC the players.
	runtime.KeepAlive(players)
	c.Close()
	return nil
}
