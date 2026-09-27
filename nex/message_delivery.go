package nex

import (
	"database/sql"
	"fmt"

	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/constants"
	protocolglobals "github.com/PretendoNetwork/nex-protocols-go/v2/globals"
	messagedelivery "github.com/PretendoNetwork/nex-protocols-go/v2/message-delivery"
	pqextended "github.com/PretendoNetwork/pq-extended"

	"github.com/Protarium-Network/mh3u-nex/database"
)

// mh3uMessageDeliveryProtocol intentionally keeps the UserMessage bytes opaque.
// MH3U's legacy UserMessage wire shape differs from the generated NEX type, and
// decoding/re-encoding it corrupts the game's tab-delimited shout payload.
type mh3uMessageDeliveryProtocol struct {
	endpoint nex.EndpointInterface
}

func (p *mh3uMessageDeliveryProtocol) Endpoint() nex.EndpointInterface { return p.endpoint }
func (p *mh3uMessageDeliveryProtocol) SetEndpoint(endpoint nex.EndpointInterface) {
	p.endpoint = endpoint
}

func (p *mh3uMessageDeliveryProtocol) HandlePacket(packet nex.PacketInterface) {
	request := packet.RMCMessage()
	if request == nil || !request.IsRequest || request.ProtocolID != messagedelivery.ProtocolID || request.MethodID != messagedelivery.MethodDeliverMessage {
		return
	}

	sender := uint64(packet.Sender().PID())
	targets := mh3uChatTargets(sender)
	relayed := 0
	for _, targetPID := range targets {
		target := SecureEndpoint.FindConnectionByPID(targetPID)
		if target == nil {
			continue
		}
		if err := pushMH3UChat(packet, target, request.CallID, request.Parameters); err != nil {
			fmt.Printf("[MH3U Chat] relay sender=%d target=%d failed: %v\n", sender, targetPID, err)
			continue
		}
		relayed++
	}
	if relayed == 0 {
		fmt.Printf("[MH3U Chat] sender=%d no live target\n", sender)
	} else {
		fmt.Printf("[MH3U Chat] sender=%d bytes=%d relayed=%d\n", sender, len(request.Parameters), relayed)
	}

	// An empty success response is required to re-arm the retail client's chat
	// send gate. Without it, only the first message of a login can be sent.
	response := nex.NewRMCSuccess(packet.Sender().Endpoint(), nil)
	response.ProtocolID = messagedelivery.ProtocolID
	response.MethodID = messagedelivery.MethodDeliverMessage
	response.CallID = request.CallID
	protocolglobals.Respond(packet, response)
}

func mh3uChatTargets(sender uint64) []uint64 {
	// Prefer the most specific gathering: a normal MatchmakeSession is the room.
	var participants []uint64
	err := database.Postgres.QueryRow(`
		SELECT participants
		FROM matchmaking.gatherings
		WHERE type='MatchmakeSession' AND $1 = ANY(participants)
		ORDER BY id DESC LIMIT 1`, sender).Scan(pqextended.Array(&participants))
	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("[MH3U Chat] room lookup for sender=%d failed: %v\n", sender, err)
	}

	// While still in a hall/lobby, choose its smallest matching gathering. As a
	// final fallback, echo to the sender so their UI and send gate still work.
	if len(participants) == 0 {
		_ = database.Postgres.QueryRow(`
			SELECT participants
			FROM matchmaking.gatherings
			WHERE type='PersistentGathering' AND $1 = ANY(participants)
			ORDER BY max_participants ASC, id DESC LIMIT 1`, sender).Scan(pqextended.Array(&participants))
	}
	if len(participants) == 0 {
		participants = []uint64{sender}
	}

	seen := make(map[uint64]bool, len(participants)+1)
	result := make([]uint64, 0, len(participants)+1)
	for _, pid := range append(participants, sender) {
		if !seen[pid] {
			seen[pid] = true
			result = append(result, pid)
		}
	}
	return result
}

func pushMH3UChat(source nex.PacketInterface, target *nex.PRUDPConnection, callID uint32, parameters []byte) error {
	request := nex.NewRMCRequest(SecureEndpoint)
	request.ProtocolID = messagedelivery.ProtocolID
	request.MethodID = messagedelivery.MethodDeliverMessage
	request.CallID = 0xFFFF0000 + callID
	request.Parameters = parameters

	var out nex.PRUDPPacketInterface
	var err error
	if target.DefaultPRUDPVersion == 0 {
		out, err = nex.NewPRUDPPacketV0(SecureServer, target, nil)
	} else {
		out, err = nex.NewPRUDPPacketV1(SecureServer, target, nil)
	}
	if err != nil {
		return err
	}
	out.SetType(constants.DataPacket)
	out.AddFlag(constants.PacketFlagReliable)
	out.AddFlag(constants.PacketFlagNeedsAck)
	out.SetSourceVirtualPortStreamType(target.StreamType)
	out.SetSourceVirtualPortStreamID(SecureEndpoint.StreamID)
	out.SetDestinationVirtualPortStreamType(target.StreamType)
	out.SetDestinationVirtualPortStreamID(target.StreamID)
	out.SetPayload(request.Bytes())
	SecureServer.Send(out)
	return nil
}
