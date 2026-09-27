package globals

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/plogger-go"
)

var Logger *plogger.Logger

// KerberosPassword is the shared secret for the Quazal Authentication /
// Rendez-Vous system accounts (PIDs 1 and 2). Overridden by
// PN_MH3U_KERBEROS_PASSWORD.
var KerberosPassword = "password"

// NEXTokenAESKey is the AES-256 key an external account server uses to seal
// the NEX login token. Only consulted when LocalAuthMode is false.
var NEXTokenAESKey []byte

// NEXPasswordSecret is the HMAC secret used to derive a stable per-PID NEX
// password without storing credentials. Only consulted when LocalAuthMode is
// false; must match the account server's secret.
var NEXPasswordSecret []byte

// LocalAuthMode enables a self-contained preservation deployment: accounts
// come from settings.json and the login token is not validated. Keep it
// disabled for any shared/public-facing deployment.
var LocalAuthMode bool

var AuthenticationServer *nex.PRUDPServer
var AuthenticationEndpoint *nex.PRUDPEndPoint
var SecureServer *nex.PRUDPServer
var SecureEndpoint *nex.PRUDPEndPoint

// MatchmakingManager backs the online matchmaking protocols. It is created in
// nex.StartSecureServer once the secure endpoint exists.
var MatchmakingManager *common_globals.MatchmakingManager

// GetUserFriendPIDs is used by friend-scoped matchmaking searches. There is
// no friends system on this server, so it always returns an empty list -
// public halls and rooms stay functional without leaking any other game's
// friend data.
func GetUserFriendPIDs(pid uint32) []uint32 {
	return []uint32{}
}
