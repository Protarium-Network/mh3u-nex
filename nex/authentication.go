// Package nex implements the Wii U Monster Hunter 3 Ultimate NEX server.
package nex

import (
	"fmt"
	"os"
	"strconv"

	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/constants"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_ticket_granting "github.com/PretendoNetwork/nex-protocols-common-go/v2/ticket-granting"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting"

	"github.com/Protarium-Network/mh3u-nex/globals"
)

var AuthenticationServer *nex.PRUDPServer
var AuthenticationEndpoint *nex.PRUDPEndPoint

func StartAuthenticationServer() {
	AuthenticationServer = nex.NewPRUDPServer()
	AuthenticationServer.LibraryVersions.SetDefault(nex.NewLibraryVersion(globals.NEXMajor, globals.NEXMinor, globals.NEXPatch))
	AuthenticationServer.AccessKey = globals.AccessKey
	// Publicly reverse-engineered retail parameter: MH3U uses PRUDP v1's
	// legacy connection signature and the pre-structure-header NEX wire
	// format (ByteStreamSettings.UseStructureHeader left at the library
	// default, false).
	AuthenticationServer.PRUDPV1Settings.LegacyConnectionSignature = true

	AuthenticationEndpoint = nex.NewPRUDPEndPoint(1)
	AuthenticationEndpoint.ServerAccount = globals.AuthenticationServerAccount
	AuthenticationEndpoint.AccountDetailsByPID = globals.AccountDetailsByPID
	AuthenticationEndpoint.AccountDetailsByUsername = globals.AccountDetailsByUsername
	AuthenticationServer.BindPRUDPEndPoint(AuthenticationEndpoint)
	AuthenticationEndpoint.OnData(logPacket("Auth"))

	protocol := ticket_granting.NewProtocol()
	AuthenticationEndpoint.RegisterServiceProtocol(protocol)
	common := common_ticket_granting.NewCommonProtocol(protocol)

	securePort := envPort("PN_MH3U_SECURE_PORT", 26401)
	secureURL := types.NewStationURL("")
	secureURL.SetURLType(constants.StationURLPRUDPS)
	secureURL.SetAddress(envString("PN_MH3U_SECURE_HOST", "localhost"))
	secureURL.SetPortNumber(uint16(securePort))
	secureURL.SetConnectionID(1)
	secureURL.SetPrincipalID(types.NewPID(2))
	secureURL.SetStreamID(1)
	secureURL.SetStreamType(constants.StreamTypeRVSecure)
	secureURL.SetType(uint8(constants.StationURLFlagPublic))
	common.ValidateLoginData = globals.ValidateLoginData
	common.SecureStationURL = secureURL
	common.BuildName = types.NewString("")
	common.SecureServerAccount = globals.SecureServerAccount

	port := envPort("PN_MH3U_AUTH_PORT", 26400)
	globals.Logger.Successf("[MH3U] Authentication server listening on UDP %d", port)
	AuthenticationServer.Listen(port)
}

func logPacket(side string) func(nex.PacketInterface) {
	return func(packet nex.PacketInterface) {
		request := packet.RMCMessage()
		if request != nil {
			fmt.Printf("[MH3U %s] PID=%d protocol=0x%02X method=0x%02X\n", side, uint64(packet.Sender().PID()), request.ProtocolID, request.MethodID)
		}
	}
}

func envPort(name string, fallback int) int {
	if parsed, err := strconv.Atoi(os.Getenv(name)); err == nil && parsed > 0 {
		return parsed
	}
	return fallback
}

func envString(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
