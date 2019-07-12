package unique

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestGenUniqueKey(t *testing.T) {
	m := make(map[string]struct{})
	var mutex sync.Mutex
	var wg sync.WaitGroup
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			uk := GenUniqueKey("test")
			mutex.Lock()
			m[uk] = struct{}{}
			mutex.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()
	assert.Equal(t, 1000, len(m), "fxxk.")
}
