# nohang Go Port

Port of [nohang](https://github.com/hakavlad/nohang) from Python 3 to Go, zero external dependencies (stdlib only).

## Project Structure

```
nohang-master-golang/
├── cmd/
│   ├── nohang/          # Main daemon
│   ├── oom-sort/        # Sort processes by OOM badness
│   ├── psi2log/         # PSI metrics logging
│   └── psi-top/         # Interactive cgroup PSI monitor
├── internal/
│   ├── action/          # Victim cache, signals, external commands
│   ├── badness/         # OOM badness calculation with regex overrides
│   ├── config/          # Config parser (key=value format with @directives)
│   ├── kmsg/            # /dev/kmsg monitor for kernel OOM events
│   ├── monitor/         # Threshold evaluation (mem/swap/zram/PSI)
│   ├── notifier/        # GUI notifications via notify-send
│   ├── proc/            # /proc reader (meminfo, psi, process, zram)
│   └── stats/           # Corrective action statistics
├── conf/                # Configuration files
├── systemd/             # systemd service files (.service)
├── openrc/              # OpenRC init scripts
├── deb/                 # Debian packaging
├── man/                 # Man pages
├── Makefile             # Build, install, uninstall
└── go.mod
```

## Binaries

| Binary    | Description |
|-----------|-------------|
| `nohang`  | OOM prevention daemon |
| `oom-sort`| List processes sorted by oom_score/badness |
| `psi2log` | Periodic PSI metrics logging |
| `psi-top` | Hierarchical cgroup PSI pressure view |

## Commands

```bash
# Build everything
make
go build ./...

# Validate config
./nohang --check -c conf/nohang/nohang.conf
./nohang --check -c conf/nohang/nohang-desktop.conf

# Manual run (root)
sudo ./nohang -c conf/nohang/nohang.conf --monitor
sudo ./nohang -c conf/nohang/nohang-desktop.conf --monitor

# View options
./nohang -h
./oom-sort -h
./psi2log -h
./psi-top -h
```

## Installing as a Service

```bash
sudo make install                 # build + install binaries + systemd + configs
sudo systemctl enable --now nohang        # server
sudo systemctl enable --now nohang-desktop # desktop
systemctl status nohang
journalctl -u nohang-desktop -f
```

## Configuration

Format identical to the original: `key = value` with `@` directives:

- `@check_kmsg` / `@debug_kmsg` — boolean flags
- `@SOFT_ACTION_RE_NAME`, `@SOFT_ACTION_RE_CGROUP_V1`, `@SOFT_ACTION_RE_CGROUP_V2` — corrective action regex
- `@BADNESS_ADJ_RE_NAME`, `@BADNESS_ADJ_RE_CMDLINE`, `@BADNESS_ADJ_RE_CGROUP_V1`, `@BADNESS_ADJ_RE_CGROUP_V2` — badness override regex

Environment variables: `NOANG_CONFIG_PATH`, `NOANG_LOG_FILE`, `NOANG_SYSLOG`.

## Differences from the Original Python

| Aspect     | Python | Go |
|------------|--------|----|
| Dependencies | >10 PyPI packages | 0 (stdlib) |
| `-m` (memload) | yes | not implemented |
| Daemon mode | systemd/openrc | systemd/openrc |

## Notes

- The `-m` flag (memload) was a debug tool in the original Python; not implemented in Go.
- All flags accept `-c` and `--config` interchangeably.
- The binary must run as root to access `/proc`, `/dev/kmsg`, and send signals.

-* AI was used in this project -*
