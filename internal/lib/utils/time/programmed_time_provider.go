package time

import (
	"slices"
	"time"
)

// Holds provider/count pairs for both the programmed time provider and its
// builder.
type programmedTimeEntry struct {
	provider TimeProvider
	count    int64
}

type programmedTimeProvider struct {
	entries      []programmedTimeEntry // entries stored in reverse order, hence popping from the back
	countInEntry int64                 // the count in the currently active (lastmost) entry
}

func (ptp *programmedTimeProvider) Now() time.Time {
	for {
		n := len(ptp.entries)

		tp := ptp.entries[n-1].provider

		if ptp.countInEntry < 0 {
			// in infinite behaviour mode, so we just repeat forever
		} else {
			if ptp.countInEntry == 0 {
				if n == 1 {
					panic("exhausted programmed time provider")
				} else {
					ptp.entries = ptp.entries[:n-1]
					ptp.countInEntry = ptp.entries[n-2].count

					continue
				}
			} else {
				ptp.countInEntry--
			}
		}

		return tp.Now()
	}
}

// A builder for the programmed time provider.
type ProgrammedTimeProviderBuilder struct {
	entries []programmedTimeEntry
}

// Adds a provider entry with a finite count, or -1 for indefinite use.
func (builder *ProgrammedTimeProviderBuilder) Push(
	provider TimeProvider,
	count int64,
) *ProgrammedTimeProviderBuilder {
	builder.entries = append(builder.entries, programmedTimeEntry{
		provider,
		count,
	})

	return builder
}

// Builds a programmed provider and returns its finite count, or -1 if infinite.
// Panics if no entries were added or a finite sequence is exhausted.
func (builder *ProgrammedTimeProviderBuilder) Build() (r TimeProvider, provideCount int64) {

	n := len(builder.entries)

	if 0 == n {
		panic("cannot build when no providers pushed")
	}

	// make a reversed copy of the entries held by the builder

	entries := make([]programmedTimeEntry, n)

	copy(entries, builder.entries)

	slices.Reverse(entries)

	// determine the provide-count, which basically involves counting up all
	// non-negative values; stop counting and set to -1 if find any negative
	// entry

	for _, entry := range entries {
		if entry.count < 0 {
			provideCount = -1
			break
		} else {
			provideCount += entry.count
		}
	}

	r = &programmedTimeProvider{
		entries:      entries,
		countInEntry: entries[n-1].count,
	}

	return
}
