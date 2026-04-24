package diagnostics

import (
	"sync"
	"time"

	d "github.com/synesissoftware/Diagnosticism.Go"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// Common timings state shared by all broadcasters.
type HeartbeatCommon struct {
	// heartbeat timings
	dg    d.DOOMGram
	mx    sync.RWMutex
	count uint64
}

// Function to be called (via defer) at the end of a given broadcaster's
// heartbeat's HeartbeatString().
func (hc *HeartbeatCommon) OnHeartbeatCompletion(tm_start time.Time) {
	hc.mx.Lock()
	defer hc.mx.Unlock()

	hc.count++

	hc.dg.PushEventDuration(snx_lib_utils_time.Since(tm_start))
}

func (hc *HeartbeatCommon) GetCommonInfo() (count uint64, strip string) {
	hc.mx.RLock()
	defer hc.mx.RUnlock()

	count = hc.count
	strip = hc.dg.ToStrip()

	return
}
