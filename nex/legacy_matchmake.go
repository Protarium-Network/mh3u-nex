package nex

import (
	"fmt"

	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	protocolglobals "github.com/PretendoNetwork/nex-protocols-go/v2/globals"
	matchmakingtypes "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmakeextension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

type createHandler func(error, nex.PacketInterface, uint32, matchmakingtypes.GatheringHolder, types.String, types.UInt16) (*nex.RMCMessage, *nex.Error)
type joinExHandler func(error, nex.PacketInterface, uint32, types.UInt32, types.String, types.Bool, types.UInt16) (*nex.RMCMessage, *nex.Error)

// MH3U omits the trailing participationCount u16 from methods 6 and 30. The
// generic generated decoder expects it and rejects the otherwise-valid packet.
type legacyMatchmakeProtocol struct {
	endpoint nex.EndpointInterface
	create   createHandler
	joinEx   joinExHandler
}

func (p *legacyMatchmakeProtocol) Endpoint() nex.EndpointInterface     { return p.endpoint }
func (p *legacyMatchmakeProtocol) SetEndpoint(e nex.EndpointInterface) { p.endpoint = e }

func (p *legacyMatchmakeProtocol) HandlePacket(packet nex.PacketInterface) {
	req := packet.RMCMessage()
	if req == nil || !req.IsRequest || req.ProtocolID != matchmakeextension.ProtocolID {
		return
	}
	in := nex.NewByteStreamIn(req.Parameters, packet.Sender().Endpoint().LibraryVersions(), packet.Sender().Endpoint().ByteStreamSettings())
	var response *nex.RMCMessage
	var rmcErr *nex.Error
	switch req.MethodID {
	case matchmakeextension.MethodCreateMatchmakeSession:
		gathering := matchmakingtypes.NewGatheringHolder()
		message := types.NewString("")
		if err := gathering.ExtractFrom(in); err != nil {
			p.fail(packet, fmt.Errorf("create gathering: %w", err))
			return
		}
		if err := message.ExtractFrom(in); err != nil {
			p.fail(packet, fmt.Errorf("create message: %w", err))
			return
		}
		response, rmcErr = p.create(nil, packet, req.CallID, gathering, message, types.NewUInt16(1))
	case matchmakeextension.MethodJoinMatchmakeSessionEx:
		gid, message, ignore := types.NewUInt32(0), types.NewString(""), types.NewBool(false)
		if err := gid.ExtractFrom(in); err != nil {
			p.fail(packet, fmt.Errorf("join gid: %w", err))
			return
		}
		if err := message.ExtractFrom(in); err != nil {
			p.fail(packet, fmt.Errorf("join message: %w", err))
			return
		}
		if err := ignore.ExtractFrom(in); err != nil {
			p.fail(packet, fmt.Errorf("join block flag: %w", err))
			return
		}
		response, rmcErr = p.joinEx(nil, packet, req.CallID, gid, message, ignore, types.NewUInt16(1))
	default:
		return
	}
	if rmcErr != nil {
		protocolglobals.RespondError(packet, matchmakeextension.ProtocolID, rmcErr)
		return
	}
	protocolglobals.Respond(packet, response)
}

func (p *legacyMatchmakeProtocol) fail(packet nex.PacketInterface, err error) {
	protocolglobals.Logger.Errorf("[MH3U] legacy matchmaking decode: %v", err)
	protocolglobals.RespondError(packet, matchmakeextension.ProtocolID, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error()))
}
