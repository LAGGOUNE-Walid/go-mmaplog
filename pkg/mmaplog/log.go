package mmaplog

import (
	"encoding/binary"
	"errors"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
)

const DefaultInitialSize = 64 * 1024 // 64kb

type Log struct {
	file   *os.File
	data   []byte
	offset int64
	mu     sync.RWMutex
}

func Open(path string) (*Log, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	size := info.Size()
	if size <= DefaultInitialSize {
		size = DefaultInitialSize
		if err := f.Truncate(size); err != nil {
			f.Close()
			return nil, err
		}
	}
	// syscall.Mmap takes length as int, which is 32 bits even on 64-bit
	// platforms. Casting a >2GiB size silently truncates the mapping.
	if size > math.MaxInt {
		f.Close()
		return nil, errors.New("mmaplog: file size exceeds maximum mappable length")
	}
	mapped, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		f.Close()
		return nil, err
	}
	var existingOffset int64 = 0
	for existingOffset+RecordSize <= size {
		// read timestamp
		ts := binary.LittleEndian.Uint64(mapped[existingOffset : existingOffset+8])
		// empty slot
		if ts == 0 {
			break
		}
		// not empty slot
		existingOffset += RecordSize
	}
	return &Log{
		file:   f,
		data:   mapped,
		offset: existingOffset,
	}, nil
}

func (l *Log) Grow(needed int64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	current := int64(len(l.data))
	if current >= needed {
		return nil
	}
	newSize := current * 2
	for newSize <= needed {
		newSize *= 2
	}
	if err := syscall.Munmap(l.data); err != nil {
		return err
	}
	if err := l.file.Truncate(newSize); err != nil {
		return err
	}
	newData, err := syscall.Mmap(
		int(l.file.Fd()),
		0,
		int(newSize),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)
	if err != nil {
		return err
	}

	l.data = newData
	return nil
}

func (l *Log) Append(rec Record) error {
	newOffset := atomic.AddInt64(&l.offset, RecordSize)
	start := newOffset - RecordSize
	l.mu.RLock()
	capLen := int64(len(l.data))
	l.mu.RUnlock()
	if newOffset > capLen {
		if err := l.Grow(newOffset); err != nil {
			return err
		}
	}
	l.mu.RLock()
	rec.EncodeTo(l.data[start:newOffset])
	l.mu.RUnlock()
	return nil
}

func (l *Log) ReadAll() []Record {
	l.mu.RLock()
	defer l.mu.RUnlock()
	total := atomic.LoadInt64(&l.offset)
	count := total / RecordSize
	records := make([]Record, 0, count)
	for i := int64(0); i < total; i += RecordSize {
		records = append(records, DecodeRecord(l.data[i:i+RecordSize]))
	}
	return records
}

func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	syscall.Munmap(l.data)
	l.file.Sync()
	return l.file.Close()
}
