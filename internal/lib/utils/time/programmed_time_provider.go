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

// Pushes an entry into the builder.
//
// Parameters:
//   - provider - instance of `TimeProvider` that will be used while this
//     entry is active;
//   - count - number of times that this entry is active, or -1 to keep
//     this entry active forever;
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

// Builds the programmed time provider instance from the current state of
// the builder (which may be used thereafter).
//
// Returns:
//   - the time provider instance;
//   - the provide-count if finite; will be -1 if infinite;
//
// Note:
// Will panic if `#Push()` has not been called.
//
// Note:
// The produced time provider will panic when `#Now()` is called if it has
// "run out" of time providers, which will happen when it has been called
// `provideCount` times, unless `provideCount` is -1, which indicates
// infinite provision. Hence, it is highly recommended to add an infinite
// provider (by passing `count` as -1) on the final call to `#Push()`.
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
