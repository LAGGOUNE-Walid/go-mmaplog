package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"mmaplog/pkg/mmaplog"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "append":
		if len(os.Args) < 4 {
			fmt.Println("Usage: mmaplog append <file> <message> [level] [traceID] [spanID] [source]")
			return
		}
		path := os.Args[2]
		msg := os.Args[3]
		level := mmaplog.LevelInfo
		traceID := "0000000000000000"
		spanID := "00000000"
		source := "mmaplog"
		if len(os.Args) >= 5 {
			switch os.Args[4] {
			case "DEBUG":
				level = mmaplog.LevelDebug
			case "WARN":
				level = mmaplog.LevelWarn
			case "ERROR":
				level = mmaplog.LevelError
			}
		}
		if len(os.Args) >= 6 {
			traceID = os.Args[5]
		}
		if len(os.Args) >= 7 {
			spanID = os.Args[6]
		}
		if len(os.Args) >= 8 {
			source = os.Args[7]
		}

		lg, err := mmaplog.Open(path)
		if err != nil {
			panic(err)
		}
		defer lg.Close()

		rec := mmaplog.NewRecord(level, traceID, spanID, source, msg)
		if err := lg.Append(rec); err != nil {
			panic(err)
		}
		fmt.Println("Record appended successfully!")

	case "read":
		if len(os.Args) < 3 {
			fmt.Println("Usage: mmaplog read <file>")
			return
		}
		path := os.Args[2]
		lg, err := mmaplog.Open(path)
		if err != nil {
			panic(err)
		}
		defer lg.Close()

		records := lg.ReadAll()
		fmt.Printf("--- Log Records (%d total) ---\n", len(records))
		for i, r := range records {
			t := time.Unix(0, r.Timestamp).Format("2006-01-02 15:04:05.000")
			fmt.Printf("[%04d] %s [%-5s] %s\n", i+1, t, r.LevelString(), r.MessageString())
		}

	case "bench":
		if len(os.Args) < 4 {
			fmt.Println("Usage: mmaplog bench <file> <num_records>")
			return
		}
		path := os.Args[2]
		count, _ := strconv.Atoi(os.Args[3])

		_ = os.Remove(path) // fresh file
		lg, err := mmaplog.Open(path)
		if err != nil {
			panic(err)
		}
		defer lg.Close()

		fmt.Printf("Benchmarking %d concurrent writes across 10 goroutines...\n", count)
		start := time.Now()

		var wg sync.WaitGroup
		workers := 10
		perWorker := count / workers

		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				rec := mmaplog.NewRecord(mmaplog.LevelInfo, "0000000000000000", "00000000", "bench", fmt.Sprintf("worker-%d benchmark message", id))
				for i := 0; i < perWorker; i++ {
					_ = lg.Append(rec)
				}
			}(w)
		}

		wg.Wait()
		duration := time.Since(start)

		opsPerSec := float64(count) / duration.Seconds()
		fmt.Printf("Completed %d writes in %v (~%.2f writes/sec)\n", count, duration, opsPerSec)

	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("mmaplog - Zero-Copy Memory Mapped Log File")
	fmt.Println("\nCommands:")
	fmt.Println("  append <file> <msg> [level]   Append a log record")
	fmt.Println("  read   <file>                 Display all log records")
	fmt.Println("  bench  <file> <count>         Benchmark concurrent append throughput")
}
