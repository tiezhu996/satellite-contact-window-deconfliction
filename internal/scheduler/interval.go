package scheduler

import (
	"sort"
	"time"

	"satellite-contact-window-deconfliction/backend/internal/model"
)

type Interval struct {
	Window model.ContactWindow
	Start  time.Time
	End    time.Time
}

type sweepEvent struct {
	At     time.Time
	Start  bool
	Window model.ContactWindow
}

func BuildInterval(window model.ContactWindow, before, after time.Duration) Interval {
	return Interval{Window: window, Start: window.StartAt.Add(-before), End: window.EndAt.Add(after)}
}

func Overlaps(left, right Interval) bool {
	return left.Start.Before(right.End) && right.Start.Before(left.End)
}

func SweepCapacity(windows []model.ContactWindow, capacity int, buffer time.Duration) [][]model.ContactWindow {
	if capacity < 1 {
		capacity = 1
	}
	events := make([]sweepEvent, 0, len(windows)*2)
	for _, window := range windows {
		interval := BuildInterval(window, buffer, buffer)
		events = append(events,
			sweepEvent{At: interval.Start, Start: true, Window: window},
			sweepEvent{At: interval.End, Start: false, Window: window},
		)
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].At.Equal(events[j].At) {
			if events[i].Start != events[j].Start {
				return !events[i].Start
			}
			return events[i].Window.ID < events[j].Window.ID
		}
		return events[i].At.Before(events[j].At)
	})
	active := map[uint]model.ContactWindow{}
	groups := map[string][]model.ContactWindow{}
	for _, event := range events {
		if !event.Start {
			delete(active, event.Window.ID)
			continue
		}
		active[event.Window.ID] = event.Window
		if len(active) <= capacity {
			continue
		}
		// Allocate a fresh slice per overrun so stored groups never share a backing
		// array; reusing one corrupted earlier entries when a later overrun
		// rewrote the same memory.
		shared := make([]model.ContactWindow, 0, len(active))
		for _, id := range sortedActiveIDs(active) {
			shared = append(shared, active[id])
		}
		groups[windowSetKey(shared)] = shared
	}
	result := make([][]model.ContactWindow, 0, len(groups))
	for _, group := range groups {
		result = append(result, group)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i][0].ID == result[j][0].ID {
			return len(result[i]) < len(result[j])
		}
		return result[i][0].ID < result[j][0].ID
	})
	return mergeContainedGroups(result)
}

// mergeContainedGroups drops any group whose window set is a strict subset of
// another group's. Without this, nested overlaps reported both the larger
// cluster and its smaller subsets as separate conflicts.
func mergeContainedGroups(groups [][]model.ContactWindow) [][]model.ContactWindow {
	kept := make([][]model.ContactWindow, 0, len(groups))
	for index, candidate := range groups {
		contained := false
		for other := range groups {
			if index == other {
				continue
			}
			if isSubset(candidate, groups[other]) && len(candidate) < len(groups[other]) {
				contained = true
				break
			}
		}
		if !contained {
			kept = append(kept, candidate)
		}
	}
	return kept
}

func isSubset(left, right []model.ContactWindow) bool {
	ids := map[uint]bool{}
	for _, window := range right {
		ids[window.ID] = true
	}
	for _, window := range left {
		if !ids[window.ID] {
			return false
		}
	}
	return true
}

func sortedActiveIDs(active map[uint]model.ContactWindow) []uint {
	ids := make([]uint, 0, len(active))
	for id := range active {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func windowSetKey(windows []model.ContactWindow) string {
	key := ""
	for _, window := range windows {
		key += fmtUint(window.ID) + ":"
	}
	return key
}

func fmtUint(value uint) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 12)
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	for left, right := 0, len(digits)-1; left < right; left, right = left+1, right-1 {
		digits[left], digits[right] = digits[right], digits[left]
	}
	return string(digits)
}
