# Gickup Web UI

This fork adds a local Web UI wrapper for the original Gickup backup tool.  
For core backup features and configuration details, please refer to the upstream project.

- Upstream: https://github.com/cooperspencer/gickup

## What This Fork Adds

- A local Web UI to configure, validate, and run backups
- Live dashboard for status, history, logs, and Prometheus metrics
- YAML file picker + form-based config builder

## Requirements

- Go 1.22+
- Node.js (for building the frontend)

## Build the gickup

```bash
go build .
```

## Build the Web UI

```bash
cd webui/frontend
npm install
npm run build
```

## Build the Web UI Backend

### Windows

```bash
cd webui
go build -o ../gickup-web.exe ./backend
```

### Linux

```bash
cd webui
go build -o ../gickup-web ./backend
```

## Run

```bash
# windows
./gickup-web.exe --conf conf.yml --listen :3780
```

or

```bash
# linux
./gickup-web --conf conf.yml --listen :3780
```

Open: `http://localhost:3780`

## Web UI Usage

### Config Page

- Use the left-side form to fill GitHub + Local destination + Cron + Prometheus.
- Optional settings are under **Extend Settings**.
- Click **Load From YAML** to select an existing `.yml/.yaml` file.
- The right panel shows **Live YAML** and **Validation Output**.
- **Save Config** writes `conf.yml`.
- **Validate** runs a dry run using the current YAML.
- **Download** exports the YAML.

### Dashboard Page

- Start/stop backups and check live status
- View run history
- View recent logs (ANSI color supported)
- View Prometheus metrics (summary + details)

## Notes

- Prometheus metrics require `metrics.prometheus.listen_addr` and `endpoint` in `conf.yml`.
- Prometheus is only served when `cron` is configured (because Gickup stays running).

## License

Copyright 2026 Aya

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.