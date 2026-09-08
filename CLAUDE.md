# EchoLocal

Pure-Go replacement firmware for the 2nd-generation Amazon Echo Dot that turns it into a Home Assistant
voice satellite speaking the ESPHome native API. See README.md for what it does and how it is installed.

## Layout

- `cmd/echod` runs on the Dot. `cmd/echoctl` is the host CLI. `cmd/mkmanifest` writes update manifests.
- `internal/hardware` drives the LEDs, microphones, speaker, buttons and BLE.
- `internal/feature` is one folder per thing Home Assistant sees. Each registers itself as a component.
- `internal/lib` is the audio pipeline: echo cancellation, denoising, wake word inference.
- `internal/android` is the Android system underneath: wifi, firewall, init services.
- `internal/config` is the settings store, one struct per feature, persisted as JSON.
- `internal/layout` is paths and identity constants shared by device and host.

## Building and checking

Go 1.26 or newer. The device build is static, no cgo, for arm64.

```sh
make build-echod          # cross-compile for the Dot
make build-echod TAGS=noasm
make test                 # host tests
make test-race            # needs cgo
make vet
make lint                 # golangci-lint, see .golangci.yml
gofmt -l .
```

CI runs vet, test, test with race, both device builds and gofmt. Lint is not in CI and has
pre-existing findings outside the Sendspin code.

`internal/component/all/all_test.go` holds a sorted list of every Home Assistant entity id. Adding an
entity means adding it there.

## Sendspin

`internal/feature/sendspin` makes the Dot a Sendspin player for Music Assistant. The protocol layer is
implemented here against the spec rather than taken from the sendspin-go library, whose releases have
no encryption. Only the library's clock filter is used. `docs/sendspin-encryption.md` explains the
design, what is stored where, and the known gaps. `guides/pairing-with-music-assistant.md` is the
user-facing guide.

Points worth knowing before touching it:

- The room offers only the pairing token method. The code-based methods were renamed between the
  aiosendspin version Music Assistant ships and the current spec, and servers reject a hello naming an
  unknown method.
- Audio chunks are read with an 8-byte header. The spec has since added a 4-byte send_ahead field
  that no server sends yet.
- Fragment reassembly accepts both the spec's encoding and the older one Music Assistant uses.
- Credentials live in `/data/misc/echolocal/sendspin.json`. Losing the identity key makes the Dot a
  new device to every server.
- `session_test.go` contains a small Sendspin server used to drive the room end to end.

## Conventions

- Comments explain why, in plain prose. No em dashes in documentation.
- Documentation goes in `docs/`, user guides in `guides/`. Keep both current with code changes.
- Changes should stay inside the feature they concern before touching shared code.
- Work goes up as pull requests.
