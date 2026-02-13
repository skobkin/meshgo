# Repository Guidelines

## Project Structure & Module Organization
- `cmd/gui`: desktop app entrypoint (Fyne UI).
- `cmd/debug`: CLI debug tool for connecting to a node and inspecting frames/events.
- `internal/app`: runtime wiring, app constants, path resolution.
- `internal/ui`: tabs, widgets, and UI behavior tests.
- `internal/radio`, `internal/transport`, `internal/connectors`: protocol decode, transport, and bus topics.
- `internal/domain`: in-memory stores/models and sync orchestration.
- `internal/persistence`: SQLite schema, migrations, repositories, writer queue.
- `internal/resources/tray`: packaged icon assets.
- `internal/radio/meshtasticpb`: generated protobuf bindings used by codec logic.

## Build, Test, and Development Commands
- `go build ./...`: build all binaries and packages.
- `go test ./...`: run all unit tests.
- `go run ./cmd/gui`: start the desktop app.
- `go run ./cmd/debug --host <node-ip> --no-subscribe`: run one-shot initial config/debug flow.
- `go run ./cmd/debug --host <node-ip> --listen-for 30s`: subscribe for a bounded session.

## Windows Icon Regeneration
- Source icon: `internal/resources/ui/light/icon_64.png`.
- Regenerate ICO (multi-size): `magick internal/resources/ui/light/icon_64.png -define icon:auto-resize=64,48,32,24,16 cmd/gui/icon_windows.ico`
- Regenerate Windows resource object: `rsrc -arch amd64 -ico cmd/gui/icon_windows.ico -o cmd/gui/icon_windows_amd64.syso`
- Commit both files together when icon changes: `cmd/gui/icon_windows.ico` and `cmd/gui/icon_windows_amd64.syso`.

## Coding Style & Naming Conventions
- Language: Go (`go 1.25` in `go.mod`).
- Formatting is mandatory: run `gofmt -w` on changed Go files.
- Package names are short lowercase nouns (`ui`, `domain`, `persistence`).
- Exported identifiers: `PascalCase`; internal helpers: `camelCase`.
- Keep UI updates on Fyne’s UI thread (`fyne.Do`/`fyne.DoAndWait`) when triggered from goroutines.
- Use structured logging (`slog`) for runtime/platform operations and failures; include actionable context fields (for example operation trigger, mode, target path/key).
- Proactively suggest refactoring when code shows weak technical depth, poor readability, or unclear structure; call out concrete improvement options.

## Testing Guidelines
- Place tests next to code using `*_test.go` (example: `internal/ui/chats_tab_test.go`).
- Prefer table-driven tests for codec/domain logic.
- Run focused tests during iteration, then `go test ./...` before opening a PR.
- Coverage target is pragmatic: new logic paths should include tests, especially decode/migration/store behavior.
- Fyne race-detector rule: tests that drive real GUI interactions (`fynetest.NewTempWindow`, `fynetest.Tap`, async UI state checks) are flaky under `go test -race`; guard only those tests with `if raceDetectorEnabled { t.Skip("Fyne GUI interaction tests are not stable under the race detector") }` and keep pure logic/UI formatting tests runnable under `-race`.
- Reuse existing race flag wiring in `internal/ui/race_enabled_test.go` and `internal/ui/race_disabled_test.go`; do not add ad-hoc race checks in individual tests.

## Completion Checklist
- Before finishing work and saying it is done, run the same baseline checks as CI:
  - `go fmt ./...` (and ensure changed files are formatted)
  - `go vet ./...`
  - `golangci-lint run ./...`
  - `go test ./...`
- If `PLAN.md` exists in the repository root and is relevant to the current task, update implementation progress there before finishing:
  - Mark completed items by checking relevant task checkboxes.
  - And/or update the `Current Status` section at the beginning with a short written progress summary.
- Do not state that work is done if any of the checks above fail.

## Commit & Pull Request Guidelines
- Follow Conventional Commits used in history: `feat(ui): ...`, `fix(ui): ...`, `chore: ...`.
- Keep commits scoped and explain behavioral impact in the subject.
- PRs should include a clear summary of user-visible changes.
- PRs should include testing performed (`go test ./...`, manual GUI/debug steps).
- PRs should include screenshots/GIFs for UI changes.
- PRs should include migration notes when `internal/persistence/db.go` changes.

## Configuration & Data Paths
- Runtime files are stored under `os.UserConfigDir()/meshgo`: `config.json`, `app.db`, `app.log`.
- Avoid hard-coding node IPs in code; pass host via config or `--host` for debug runs.
