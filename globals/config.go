package globals

// NEX configuration for Monster Hunter 3 Ultimate (Wii U). Three title IDs
// share the same online service (JPN 0005000010117200, USA 0005000010118300,
// EUR 0005000010104d00); GameServerID is the low 32 bits of the EUR title ID,
// following the pattern every title in this project uses.
const (
	GameServerID = "10104d00"
	AccessKey    = "cb2b2f5a"

	// PRUDP library version reported by both endpoints. MH3U uses PRUDP v1's
	// legacy connection signature and the pre-structure-header NEX wire
	// format (see nex/authentication.go).
	NEXMajor = 3
	NEXMinor = 0
	NEXPatch = 5
)
