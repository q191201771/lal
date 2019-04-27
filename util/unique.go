package util

import (
	"fmt"
	"sync/atomic"
)

var globalId uint64

func GenUniqueId() uint64 {
	return atomic.AddUint64(&globalId, 1)
}

func GenUniqueKey(prefix string) string {
	return fmt.Sprintf("%s%d", prefix, GenUniqueId())
}
