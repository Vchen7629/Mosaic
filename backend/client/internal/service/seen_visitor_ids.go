package service

import "sync"

// in memory mapping to keep track of what
// IDs were already seen for this session
// prevents duplicate db fetch for already
// fetched briefings for a visitor
var (
	seenFaceMap = map[int32]bool{}
	seenMu		sync.Mutex
)

func GetSeenVisitors() []int32 {
	seenMu.Lock()
	defer seenMu.Unlock()

	ids := make([]int32, 0, len(seenFaceMap))
	for id := range seenFaceMap {
		ids = append(ids, id)
	}

	return ids
}

func HasSeenVisitor(visitorID int32) bool {
	seenMu.Lock()
	defer seenMu.Unlock()
	return seenFaceMap[visitorID]
}

func AddSeenVisitor(visitorID int32) {
	seenMu.Lock()
	defer seenMu.Unlock()
	
	seenFaceMap[visitorID] = true
}

func ClearSeenVisitors() {
	seenMu.Lock()
	defer seenMu.Unlock()

	seenFaceMap = map[int32]bool{}
}