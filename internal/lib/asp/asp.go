// Package asp applies the speaker tuning the vendor's audio signal processing applies.
//
// A 1024-tap FIR carries the tuning: relative to 500 Hz it lifts 125-400 Hz by up to 27 dB and cuts
// 2-3 kHz by about 11 dB. A four-band compressor and limiter follows, and it is not optional — a
// 27 dB shelf at 160 Hz is only survivable because the band below 115 Hz is crushed 20:1 before it
// reaches the driver.
//
// The coefficients are the vendor's and are not ours to ship, so they are read off the device.
package asp

import (
	"fmt"
	"math"
	"path/filepath"
)

// VendorDir is where the tuning lives on a device.
const VendorDir = "/vendor/etc/audio-algorithms"

// Rate is the rate the tuning was designed at, which is also the only rate the playback codec takes.
const Rate = 48000

// taps is the length of the tuning filter. A file of any other length is a tuning we do not know.
const taps = 1024

// eqFiles are the vendor's volume-dependent EQ, with the volume boundary each one reaches up to. The
// six hold one filter shape and differ only in the gain in front of it, which is half of how the
// device gets louder; the attenuation in the volume curve is the other half.
var eqFiles = []struct {
	upTo float64
	name string
}{
	{0.5, "EQ_50.cfg"},
	{0.6, "EQ_60.cfg"},
	{0.7, "EQ_70.cfg"},
	{0.8, "EQ_80.cfg"},
	{0.9, "EQ_90.cfg"},
	{1.0, "EQ_100.cfg"},
}

// mbclFile is the compressor and limiter that sits under the tuning.
const mbclFile = "MBCL.cfg"

// Tuning is a loaded tuning, shared and read-only. Chain turns it into something that can process.
type Tuning struct {
	taps  []float32
	gains []float64 // what each bucket puts in front of taps, linear, in eqFiles order
	mbcl  mbcl
}

// Load reads a tuning out of a directory, normally VendorDir.
func Load(dir string) (*Tuning, error) {
	t := &Tuning{}
	for _, e := range eqFiles {
		h, err := readFloats(filepath.Join(dir, e.name), taps)
		if err != nil {
			return nil, err
		}
		if t.taps == nil {
			t.taps, t.gains = h, []float64{1}
			continue
		}

		// The filter is taken from the first file and the rest are read for their gain alone, so this
		// insists they really are the same filter: a set that is not would need a filter each.
		g, err := scaleOf(t.taps, h)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.name, err)
		}
		t.gains = append(t.gains, g)
	}

	m, err := readMBCL(filepath.Join(dir, mbclFile))
	if err != nil {
		return nil, err
	}
	t.mbcl = m
	return t, nil
}

// scaleOf reports how much bigger b is than a, and refuses anything that is not a scaled copy of it.
func scaleOf(a, b []float32) (float64, error) {
	var ab, aa float64
	for i := range a {
		ab += float64(a[i]) * float64(b[i])
		aa += float64(a[i]) * float64(a[i])
	}
	g := ab / aa

	var off, tot float64
	for i := range a {
		d := float64(b[i]) - g*float64(a[i])
		off += d * d
		tot += float64(b[i]) * float64(b[i])
	}
	if off/tot > 1e-9 {
		return 0, fmt.Errorf("a different filter, not the same one %.2f dB louder", 20*math.Log10(g))
	}
	return g, nil
}

// Makeup is the gain the vendor puts in front of the filter at this fraction of full volume: the
// first bucket the volume reaches up to, and the last of them for anything above the rest.
func (t *Tuning) Makeup(of float64) float64 {
	for i, e := range eqFiles {
		if of <= e.upTo {
			return t.gains[i]
		}
	}
	return t.gains[len(t.gains)-1]
}

// Chain is a tuning applied to one stream. It holds the filter history and the compressor's
// envelopes, so it belongs to whoever is playing and is not safe for concurrent use.
type Chain struct {
	fir  *fir
	comp *mbclState
}

// Chain builds the processing state for a stream of the given block size. Every call to Process must
// then be exactly that long: the FIR's transform is sized for it.
func (t *Tuning) Chain(block int) (*Chain, error) {
	if block <= 0 {
		return nil, fmt.Errorf("asp: a block is %d samples", block)
	}
	if t.mbcl.Bypass {
		return nil, fmt.Errorf("asp: %s asks to be bypassed", mbclFile)
	}

	comp, err := newMBCL(t.mbcl, Rate)
	if err != nil {
		return nil, err
	}
	return &Chain{fir: newFIR(t.taps, block), comp: comp}, nil
}

// Process applies the tuning to one block in place. Samples are full scale at ±1, which is what the
// compressor's thresholds are in dB of.
func (c *Chain) Process(x []float32) {
	c.fir.process(x)
	c.comp.process(x)
}

// Reset drops the filter's history and the compressor's envelopes, so the next block is processed as
// though it were the first. A chain that stopped being used has a history of whatever was playing
// then, and starting from silence is better than smearing that across what is playing now.
func (c *Chain) Reset() {
	c.fir.reset()
	c.comp.reset()
}
