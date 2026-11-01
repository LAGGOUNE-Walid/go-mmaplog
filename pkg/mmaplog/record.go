package mmaplog

import (
	"encoding/binary"
	"strings"
	"time"
)

const (
	RecordSize int64 = 256
	MsgCap           = 200
	LevelDebug uint8 = 0
	LevelInfo  uint8 = 1
	LevelWarn  uint8 = 2
	LevelError uint8 = 3
)

type Record struct {
	Timestamp int64
	Level     uint8
	_         [7]byte
	TraceID   [16]byte
	SpanID    [8]byte
	Source    [16]byte
	Message   [MsgCap]byte
}

func NewRecord(level uint8, traceID, spanID, source, msg string) Record {
	var r Record
	r.Timestamp = time.Now().UnixNano()
	r.Level = level
	copy(r.TraceID[:], []byte(traceID))
	copy(r.SpanID[:], []byte(spanID))
	copy(r.Source[:], []byte(source))
	copy(r.Message[:], []byte(msg))
	return r
}
func (r *Record) LevelString() string {
	switch r.Level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func (r *Record) MessageString() string {
	// \x00 because copy will append that
	return strings.TrimRight(string(r.Message[:]), "\x00")
}

func (r *Record) EncodeTo(dst []byte) {
	binary.LittleEndian.PutUint64(dst[0:8], uint64(r.Timestamp))
	dst[8] = r.Level
	copy(dst[16:32], r.TraceID[:])
	copy(dst[32:40], r.SpanID[:])
	copy(dst[40:56], r.Source[:])
	copy(dst[56:256], r.Message[:])
}

func DecodeRecord(src []byte) Record {
	var r Record
	r.Timestamp = int64(binary.LittleEndian.Uint64(src[0:8]))
	r.Level = src[8]
	copy(r.TraceID[:], src[16:32])
	copy(r.SpanID[:], src[32:40])
	copy(r.Source[:], src[40:56])
	copy(r.Message[:], src[56:256])
	return r
}
