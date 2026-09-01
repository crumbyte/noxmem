package pprofx

import (
	"runtime"
)

// Difference between Windows epoch (1601) and Unix epoch (1970)
// in nanoseconds: 116444736000000000 * 100ns intervals.
const windowsToUnixEpochNs = 11644473600000000000

type GCPause struct {
	Duration uint64
	EndTime  uint64
}

// GCPauses returns a list of GCPause instances with the predefined size where
// each item represents a single GC pause duration and the corresponding pause
// end time. The retrieval starts from the latest pause determined by the total
// number of GC cycles, down to the number defined in the size argument.
func GCPauses(stats runtime.MemStats, size int) []GCPause {
	lastGCIdx := int(stats.NumGC+255) % 256

	gcPauseMap := make([]GCPause, 0, size)

	for i, idx := 0, lastGCIdx; i < size; i, idx = i+1, idx-1 {
		if idx < 0 {
			idx = len(stats.PauseNs) - 1
		}

		pauseNS := stats.PauseNs[idx]
		pauseEnd := stats.PauseEnd[idx]

		if pauseEnd == 0 {
			break
		}

		if pauseEnd > windowsToUnixEpochNs {
			pauseEnd -= windowsToUnixEpochNs
		}

		gcPauseMap = append(
			gcPauseMap,
			GCPause{
				Duration: pauseNS,
				EndTime:  pauseEnd,
			},
		)
	}

	return gcPauseMap
}

func MaxGCPause(currenPause uint64, stats runtime.MemStats) uint64 {
	for i := range stats.PauseNs {
		currenPause = max(currenPause, stats.PauseNs[i])
	}

	return currenPause
}
