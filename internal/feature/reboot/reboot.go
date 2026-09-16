// Package reboot is the one thing Home Assistant can ask for that ends this process for good: a
// reboot of the device. It is a button the dashboard shows, and pressing it is the same as pulling
// the plug — the whole device comes back, and echod with it.
package reboot

import (
	"sync"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/lib/safe"
	"github.com/ygelfand/echolocal/internal/update"
)

func init() {
	component.Register(component.Device, Get, component.Order(90))
}

var (
	once   sync.Once
	shared *Reboot
)

// Reboot asks init to bring the whole device back, which unwinds the services it started rather than
// dropping the device where it stands.
type Reboot struct {
	button *esphome.Button
}

func Get() *Reboot {
	once.Do(func() { shared = build() })
	return shared
}

func build() *Reboot {
	r := &Reboot{}

	r.button = &esphome.Button{
		Base: esphome.Base{
			ObjectID: "restart",
			Name:     "Restart",
			Icon:     "mdi:restart",
			Category: esphome.CategoryDiagnostic,
		},
		DeviceClass: "restart",
		OnPress: func() {
			safe.Go("reboot", func() {
				update.Reboot("pressed in Home Assistant")
			})
		},
	}

	return r
}

func (r *Reboot) Name() string { return "reboot" }

func (r *Reboot) Entities() []esphome.Entity {
	return []esphome.Entity{r.button}
}
