# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Purpose

A deliberately simple Go web app used as a teaching prop in cloud-native trainings (Docker, Kubernetes, Istio, Helm, ...). Its "features" exist so trainers can demonstrate platform behavior: liveness/readiness probes, startup/graceful-shutdown delays, memory/CPU leaks, config via file/env/ConfigMap, persistent volumes, and the downward API. Changes should keep behavior predictable and observable from logs, since trainees watch the output. Images are published to `quay.io/kubermatic-labs/training-application`.

## Commands

All Go code lives in `src/` (module `github.com/cloudnativetrainings/training-application`); the makefile `cd`s there for you.

```bash
make build              # go build -> ./training-application (repo root)
make run                # build and run natively; reads ./training-application.conf
make lint               # golangci-lint in src/
make docker-lint        # hadolint docker/Dockerfile
make docker-lint-all    # hadolint all Dockerfiles (B ignores DL3025, distroless ignores DL3006 on purpose)
make docker-build       # build + lint, then build the default image
make docker-run         # run image with -m=10m --cpus=.5 (tight limits make leak demos visible)
make docker-build-all   # build default, -A, -B, -distroless images
make helm-push          # package & push helm-chart/ to oci://quay.io/kubermatic-labs/helm-charts/
```

There are no tests yet (see `todo.md`). Versions are set at the top of the makefile: `BUILD_VERSION` (image tag) and `HELM_CHART_VERSION` (`helm-chart/Chart.yaml` keeps `0.0.0`; the real version is passed at package time).

## Architecture

Single `package main`, wired together in `src/main.go`:

- **Global shared state**: one `*appConfig` (`src/config.go`) is created in `main` and shared by pointer between the HTTP server, the stdin CLI, the lifecycle handler, and the persister. Runtime toggles (`alive`, `ready`, `rootEnabled`, `rootDelaySeconds`) are just fields mutated on this struct — there is no locking.
- **Config resolution** (`getAppConfig*Value`): env var (only for `name`/`version`/`message`/`color` → `APP_NAME`, `APP_VERSION`, `APP_MESSAGE`, `APP_COLOR`) > properties file (`magiconair/properties`, `key = value`) > default. Config file path is `./training-application.conf` unless started with exactly `--configFilePath <path>`. `catMode` fetches an image URL from thecatapi.com at init time.
- **Startup sequence**: load config → optionally switch logging to file only → sleep `startUpDelaySeconds` → start stdin CLI and signal handler goroutines → optionally start persister → set `ready = true` → `ListenAndServe`. Readiness is false until this point.
- **Lifecycle** (`handleLifecycle` in `main.go`): on SIGTERM/SIGINT sets `ready = false`, sleeps `tearDownDelaySeconds`, then `os.Exit(0)`. This is the graceful-shutdown demo.
- **HTTP server** (`src/server.go`): `net/http` ServeMux on `applicationPort` (8080). `/` renders `src/root.html`, which is embedded via `//go:embed` (so Dockerfiles must `COPY src/root.html`). `/liveness` → 200/500, `/readiness` → 200/503. `request_info.go` / `response_info.go` format request/response details for logging and the page.
- **Stdin CLI** (`src/cli.go`): line-based commands (`set ready|unready|alive|dead`, `leak mem|cpu`, `delay / N`, `enable|disable /`, `request <url>`, `init`, `config`, `help`). In containers this requires `tty`/`stdin` and `docker attach` / `kubectl attach -it`. When adding a command, also update `createHelpText()` and the command table in `README.md`.
- **Persister** (`src/persister.go`): if `persistMetaInfo = true` and `./data/` exists, periodically writes `WORKER_NODE_NAME`, `POD_NAME`, `POD_IP` (downward API env vars) to `./data/metainfo.txt`. Used for PVC demos.

## Docker image variants (intentional differences — don't "fix" them)

- `docker/Dockerfile`: alpine builder + alpine runtime (default tag `X.Y.Z`).
- `Dockerfile-A`: bookworm builder + ubuntu runtime, exec-form `ENTRYPOINT` (tag `-A`).
- `Dockerfile-B`: same as A but **shell-form** `ENTRYPOINT`, so the app isn't PID 1 and doesn't receive SIGTERM — used to teach signal handling (tag `-B`).
- `Dockerfile-distroless`: `CGO_ENABLED=0` static build on `distroless/static` (tag `-distroless`).

All copy `training-application.conf` next to the binary, which is the default config location.

## Helm chart

`helm-chart/` renders values into a ConfigMap that is mounted over `/app/training-application.conf` via `subPath`, injects downward-API env vars, and mounts a PVC at `/app/data/` when `persistMetaInfo` is true. Resource names are hardcoded (`my-app`, `training-application-pvc`).

## CI

GitHub Actions on push to `main` only (path-filtered): `build_and_push_images.yaml` runs `make build` + `make docker-push-all` (multi-arch amd64/arm64 via buildx); `build_and_push_chart.yaml` runs `make helm-push`. Lint steps are currently commented out. Any matching push to `main` re-pushes the current `BUILD_VERSION` tags, overwriting existing images on quay. Bump the version in the makefile when releasing.
