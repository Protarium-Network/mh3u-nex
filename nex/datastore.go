package nex

import (
	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastoretypes "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

// searchDataStoreObjects supplies the empty-but-successful bootstrap result
// expected by a fresh MH3U account. No matching DLC/event objects exist yet,
// and NEX explicitly permits SearchObject to return zero results.
func searchDataStoreObjects(_ error, packet nex.PacketInterface, callID uint32, _ datastoretypes.DataStoreSearchParam) (*nex.RMCMessage, *nex.Error) {
	// The retail MH3U request predates several trailing DataStoreSearchParam
	// fields. The generated modern decoder reports a short-buffer error after
	// successfully reading the legacy portion; this endpoint intentionally does
	// not use the parameters, so tolerate that decoder error.
	endpoint := packet.Sender().Endpoint()
	result := datastoretypes.NewDataStoreSearchResult()
	result.TotalCount = types.NewUInt32(0)
	result.Result = types.NewList[datastoretypes.DataStoreMetaInfo]()
	result.TotalCountType = types.NewUInt8(0)

	out := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
	result.WriteTo(out)

	response := nex.NewRMCSuccess(endpoint, out.Bytes())
	response.ProtocolID = datastore.ProtocolID
	response.MethodID = datastore.MethodSearchObject
	response.CallID = callID
	return response, nil
}
