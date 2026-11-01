# mmaplog

A zero-copy memory-mapped log file engine in Go. Records are written directly
into a `MAP_SHARED` mapping of the log file, so appending is a slice copy and
the kernel handles durability — no per-write syscalls.

## Features

- Fixed 256-byte records: timestamp, level, trace/span ID, source, message
- `MAP_SHARED` writes visible to other processes and durable on disk
- Auto-growing file: unmap → `Truncate` → remap (doubling strategy)
- Concurrent-safe appends (atomic offset, RWMutex)
- CLI with `append`, `read`, and `bench` subcommands

## Install

```bash
go install ./cmd/mmaplog
```

Or build the binary:

```bash
go build -o mmaplog ./cmd/mmaplog
```

## CLI

```bash
# Append a record
./mmaplog append app.log "System booted" INFO 0000000000000000 00000000 app-server

# Read all records
./mmaplog read app.log

# Benchmark: 100k concurrent writes across 10 goroutines
./mmaplog bench bench.log 100000
```

Append arguments: `append <file> <message> [level] [traceID] [spanID] [source]`

## Library usage

```go
import "mmaplog/pkg/mmaplog"

lg, err := mmaplog.Open("app.log")
if err != nil { ... }
defer lg.Close()

lg.Append(mmaplog.NewRecord(mmaplog.LevelInfo, "trace", "span", "svc", "hello"))
```

## Record format

| Field    | Offset | Size  |
|----------|--------|-------|
| Timestamp | 0      | 8     |
| Level     | 8      | 1     |
| padding   | 9      | 7     |
| TraceID   | 16     | 16    |
| SpanID    | 32     | 8     |
| Source    | 40     | 16    |
| Message   | 56     | 200   |

Total: **256 bytes** per record.

## Notes

- `syscall.Mmap` takes its length as `int`, which is 32 bits on 32-bit
  platforms. `Open` refuses files larger than `math.MaxInt` to avoid a
  silent truncation of the mapping at the 2 GiB boundary.
- On 64-bit platforms this limit is effectively 8 GiB (the kernel's
  `vm.max_map_count` / address-space limits apply first in practice).