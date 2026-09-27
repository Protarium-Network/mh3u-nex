package globals

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// ValidateLoginData is the ticket-granting hook that checks the login token
// the client presents. In LocalAuthMode there is no account server issuing
// tokens, so it is accepted unconditionally; otherwise it must decrypt and
// validate against NEXTokenAESKey.
func ValidateLoginData(pid types.PID, loginData types.DataHolder) *nex.Error {
	if LocalAuthMode {
		return nil
	}

	return common_globals.ValidatePretendoLoginData(pid, loginData, NEXTokenAESKey)
}
