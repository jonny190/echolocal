package speaker

import (
	"context"
	"sync"
	"testing"
	"time"
)

// counted is a Background that keeps what it was last told, so the tests can check the driver left it
// able to play rather than merely that it called something.
type counted struct {
	mu   sync.Mutex
	down bool
}

func (b *counted) Stand(down bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.down = down
}

// held is whether the background is still standing down, which is the thing that matters: a
// background left standing down never plays again, and nothing reports an error about it.
func (b *counted) held() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.down
}

func TestBackgroundPlaysAgainAfterOneSound(t *testing.T) {
	d := driver()
	bg := &counted{}
	d.Yields(bg)

	waitFor(t, d.Claim("one", func(context.Context, *Player) error { return nil }))

	if bg.held() {
		t.Fatal("still standing down after the only sound finished")
	}
}

// One claim displacing another is the ordinary case — a wake tone, then the reply that follows it —
// and it used to leak a hold every time. Each claim suspended, only the last one to finish resumed,
// and the background was left stood down for the life of the process with nothing logged.
func TestBackgroundPlaysAgainAfterOneSoundDisplacesAnother(t *testing.T) {
	d := driver()
	bg := &counted{}
	d.Yields(bg)

	first := d.Claim("first", func(ctx context.Context, _ *Player) error {
		<-ctx.Done()
		return nil
	})

	second := d.Claim("second", func(context.Context, *Player) error { return nil })

	waitFor(t, second)
	waitFor(t, first)

	if bg.held() {
		t.Fatal("still standing down after both sounds finished")
	}
}

// Silencing clears the current claim itself, so the claim's own release finds nothing to release and
// the background has to be let go here instead.
func TestBackgroundPlaysAgainAfterSilence(t *testing.T) {
	d := driver()
	bg := &counted{}
	d.Yields(bg)

	c := d.Claim("held", func(ctx context.Context, _ *Player) error {
		<-ctx.Done()
		return nil
	})

	// Let the errand reach its wait before taking the speaker away.
	time.Sleep(50 * time.Millisecond)
	d.Silence()
	waitFor(t, c)

	if bg.held() {
		t.Fatal("still standing down after being silenced")
	}
}

// A run of sounds must not accumulate anything, however many of them there were.
func TestBackgroundSurvivesAStreamOfSounds(t *testing.T) {
	d := driver()
	bg := &counted{}
	d.Yields(bg)

	for range 20 {
		waitFor(t, d.Claim("one", func(context.Context, *Player) error { return nil }))
	}

	if bg.held() {
		t.Fatal("still standing down after twenty sounds")
	}
}
