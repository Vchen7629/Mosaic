package service

import "sync"

// in memory mapping to keep track of
// profile sessions. maps session token to
// user id.
var (
	profileIDMap = map[int32]string{}
	sessionMap   = map[string]int32{}
	sessionMu    sync.Mutex
)

// adds the newly generated session to the mapping
func AddNewProfileSession(sessionToken string, profileID int32) {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	oldToken, exists := profileIDMap[profileID]
	if exists {
		delete(sessionMap, oldToken)
	}

	profileIDMap[profileID] = sessionToken
	sessionMap[sessionToken] = profileID
}

func FetchSessionToken(profileID int32) string {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	return profileIDMap[profileID]
}

func FetchProfileIDWithSession(sessionToken string) (int32, bool) {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	profileID, exists := sessionMap[sessionToken]
	return profileID, exists
}

func ClearSessions() {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	sessionMap = map[string]int32{}
	profileIDMap = map[int32]string{}
}
