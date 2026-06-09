package telemetry

import (
	"strconv"
	"sync/atomic"
	"time"
)

var traceSequence atomic.Uint64

// NewTraceID 生成进程内唯一的链路标识。
func NewTraceID() string {
	sequence := traceSequence.Add(1)
	return strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.FormatUint(sequence, 36)
}
