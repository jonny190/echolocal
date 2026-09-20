package diag

import (
	"fmt"
	"log/slog"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/echolocal/internal/android/logd"
	"github.com/ygelfand/echolocal/internal/layout"
)

// logPage is how much log text goes in one answer. The transport caps a message at 65515 bytes and
// JSON escaping only adds to what a line costs, so half of that leaves room to spare.
const logPage = 32 * 1024

// Logs is one page of the device's log. Pages is the whole count, so a caller knows when to stop
// without a second round trip.
type Logs struct {
	Version int      `json:"version"`
	Page    int      `json:"page"`
	Pages   int      `json:"pages"`
	Lines   []string `json:"lines"`
}

// Actions is how Home Assistant reads the log after the fact. The live stream only carries what
// happens while something is subscribed; this is what the daemon still holds either way.
func (d *Diag) Actions() []*esphome.Action {
	return []*esphome.Action{
		{
			Name:    "logs",
			Args:    []esphome.Arg{{Name: "page", Type: esphome.ArgInt}},
			Answers: true,
			Run: func(c esphome.Call) (any, error) {
				return logs(c.Int("page"))
			},
		},
	}
}

func logs(page int) (*Logs, error) {
	lines, err := logd.Dump(layout.LogTag)
	if err != nil {
		if len(lines) == 0 {
			return nil, err
		}
		slog.Warn("reading the log back was cut short", "lines", len(lines), "err", err)
	}

	text := make([]string, len(lines))
	for i, l := range lines {
		text[i] = l.Level.String() + " " + l.Text
	}

	pages := split(text)
	if page < 0 || page >= len(pages) {
		return nil, fmt.Errorf("page %d of %d", page, len(pages))
	}
	return &Logs{Version: 1, Page: page, Pages: len(pages), Lines: pages[page]}, nil
}

// split cuts the lines into pages the transport will carry, never splitting a line. Always at least
// one page: a device with nothing to say still answers.
func split(lines []string) [][]string {
	pages := [][]string{nil}
	size := 0

	for _, l := range lines {
		n := len(l) + 1
		if size > 0 && size+n > logPage {
			pages = append(pages, nil)
			size = 0
		}

		at := len(pages) - 1
		pages[at] = append(pages[at], l)
		size += n
	}
	return pages
}
