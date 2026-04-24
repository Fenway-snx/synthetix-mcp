package http

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	extentOfKnownContiguousStandardDomainCodes = 16
)

type standardDomainCloseCodeCounts struct {
	counts [extentOfKnownContiguousStandardDomainCodes]atomic.Int64
}

// Efficiently count websocket close codes.
type WebSocketCloseCodeCounts struct {
	standardDomainCloseCodeCounts    standardDomainCloseCodeCounts
	standardCount                    atomic.Int64 // == sum of all standardDomainCloseCodeCounts
	reservedCount                    atomic.Int64
	customCount                      atomic.Int64
	totalCount                       atomic.Int64 // == sum of standardCount + reservedCount + customCount
	mu                               sync.RWMutex // for control of access to nonstandardDomainCloseCodeCounts
	nonstandardDomainCloseCodeCounts map[WebSocketCloseCode]int64
}

// Pushes a count of a WebSocket Close Code.
func (w *WebSocketCloseCodeCounts) Push(closeCode WebSocketCloseCode) {
	w.totalCount.Add(1)

	if closeCode.IsReserved() {

		w.reservedCount.Add(1)

		return
	}

	if closeCode.IsStandard() {

		w.standardCount.Add(1)

		i := int(closeCode) - 1_000

		if i < len(w.standardDomainCloseCodeCounts.counts) {

			w.standardDomainCloseCodeCounts.counts[i].Add(1)

			return
		}
	}

	w.customCount.Add(1)

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.nonstandardDomainCloseCodeCounts == nil {

		w.nonstandardDomainCloseCodeCounts = make(map[WebSocketCloseCode]int64)
	}

	w.nonstandardDomainCloseCodeCounts[closeCode] = 1 + w.nonstandardDomainCloseCodeCounts[closeCode]
}

// Obtains a human-readable summary of the counts, in which only those codes
// with a non-0 count will appear.
func (w *WebSocketCloseCodeCounts) String() string {

	// NOTE: the implementation does not guarantee precise correctness at the
	// call time; rather it will be almost precisely correct almost all the
	// time.

	numReserved := w.reservedCount.Load()

	if w.customCount.Load() != 0 {

		if w.standardCount.Load() != 0 {
			// To make overall processing simpler (and more efficient), we elect
			// to write into the custom map the current standard counts. Since
			// they are not otherwise used, this breaks nothing and saves us
			// creating a copy of the map (or construing some _really_ horrible
			// logic to integrate the standard and the custom counts).

			func() {

				w.mu.Lock()
				defer w.mu.Unlock()

				for i := 0; i != len(w.standardDomainCloseCodeCounts.counts); i++ {

					if w.nonstandardDomainCloseCodeCounts == nil {

						w.nonstandardDomainCloseCodeCounts = make(map[WebSocketCloseCode]int64)
					}

					closeCode := WebSocketCloseCode(1_000 + i)

					w.nonstandardDomainCloseCodeCounts[closeCode] = w.standardDomainCloseCodeCounts.counts[i].Load()
				}
			}()
		}

		// Now we process the whole map
		w.mu.RLock()
		defer w.mu.RUnlock()

		keys := make([]int, 0, len(w.nonstandardDomainCloseCodeCounts))
		for key := range w.nonstandardDomainCloseCodeCounts {
			keys = append(keys, int(key))
		}
		sort.Ints(keys)

		var sb strings.Builder

		sb.WriteByte('{')

		for _, key := range keys {

			count := w.nonstandardDomainCloseCodeCounts[WebSocketCloseCode(key)]
			if count != 0 {

				if sb.Len() > 1 {

					sb.WriteByte(',')
				}

				sb.WriteString(fmt.Sprintf(" %d:%d", key, count))
			}
		}

		if 0 != numReserved {

			if sb.Len() > 1 {

				sb.WriteByte(';')
			}

			sb.WriteString(fmt.Sprintf(" reserved:%d", numReserved))
		}

		if sb.Len() > 1 {

			sb.WriteByte(' ')
		}

		sb.WriteByte('}')

		return sb.String()
	} else {

		var sb strings.Builder

		sb.WriteByte('{')

		for i := 0; i != extentOfKnownContiguousStandardDomainCodes; i++ {

			count := w.standardDomainCloseCodeCounts.counts[i].Load()
			if count != 0 {

				if sb.Len() > 1 {

					sb.WriteByte(',')
				}

				sb.WriteString(fmt.Sprintf(" %d:%d", 1_000+i, count))
			}
		}

		if 0 != numReserved {

			if sb.Len() > 1 {

				sb.WriteByte(';')
			}

			sb.WriteString(fmt.Sprintf(" reserved:%d", numReserved))
		}

		if sb.Len() > 1 {

			sb.WriteByte(' ')
		}

		sb.WriteByte('}')

		return sb.String()
	}
}
