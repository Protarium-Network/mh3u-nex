package nex

import (
	"fmt"
	"time"

	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	common_matchmaking "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making"
	common_matchmaking_ext "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making-ext"
	common_matchmake_extension "github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension"
	matchmake_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	common_nat "github.com/PretendoNetwork/nex-protocols-common-go/v2/nat-traversal"
	common_secure "github.com/PretendoNetwork/nex-protocols-common-go/v2/secure-connection"
	common_utility "github.com/PretendoNetwork/nex-protocols-common-go/v2/utility"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	matchmaking "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	matchmaking_ext "github.com/PretendoNetwork/nex-protocols-go/v2/match-making-ext"
	matchmakingtypes "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
	nat "github.com/PretendoNetwork/nex-protocols-go/v2/nat-traversal"
	secure "github.com/PretendoNetwork/nex-protocols-go/v2/secure-connection"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"

	"github.com/Protarium-Network/mh3u-nex/database"
	"github.com/Protarium-Network/mh3u-nex/globals"
)

var SecureServer *nex.PRUDPServer
var SecureEndpoint *nex.PRUDPEndPoint

func StartSecureServer() {
	SecureServer = nex.NewPRUDPServer()
	SecureServer.LibraryVersions.SetDefault(nex.NewLibraryVersion(globals.NEXMajor, globals.NEXMinor, globals.NEXPatch))
	SecureServer.AccessKey = globals.AccessKey
	SecureServer.PRUDPV1Settings.LegacyConnectionSignature = true
	SecureEndpoint = nex.NewPRUDPEndPoint(1)
	SecureEndpoint.IsSecureEndPoint = true
	SecureEndpoint.ServerAccount = globals.SecureServerAccount
	SecureEndpoint.AccountDetailsByPID = globals.AccountDetailsByPID
	SecureEndpoint.AccountDetailsByUsername = globals.AccountDetailsByUsername
	SecureServer.BindPRUDPEndPoint(SecureEndpoint)
	SecureEndpoint.OnData(logPacket("Secure"))
	SecureEndpoint.OnConnectionEnded(func(c *nex.PRUDPConnection) { fmt.Printf("[MH3U Secure] PID=%d disconnected\n", uint64(c.PID())) })

	globals.MatchmakingManager = common_globals.NewMatchmakingManager(SecureEndpoint, database.Postgres)
	globals.MatchmakingManager.GetUserFriendPIDs = globals.GetUserFriendPIDs
	registerSecureProtocols()

	port := envPort("PN_MH3U_SECURE_PORT", 26401)
	globals.Logger.Successf("[MH3U] Secure server listening on UDP %d", port)
	SecureServer.Listen(port)
}

func registerSecureProtocols() {
	secureProtocol := secure.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(secureProtocol)
	secureCommon := common_secure.NewCommonProtocol(secureProtocol)
	secureCommon.EnableInsecureRegister()
	// Reports are telemetry only. Keep this service independent from any
	// shared database so it can be upgraded/restarted alone.
	secureCommon.CreateReportDBRecord = func(types.PID, types.UInt32, types.QBuffer) error { return nil }

	utilityProtocol := utility.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(utilityProtocol)
	common_utility.NewCommonProtocol(utilityProtocol)

	natProtocol := nat.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(natProtocol)
	common_nat.NewCommonProtocol(natProtocol)

	// MH3U queries DataStore during the initial online bootstrap. Leaving the
	// protocol unregistered makes the retail client wait forever because no
	// RMC response is emitted for SearchObject.
	dataStoreProtocol := datastore.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(dataStoreProtocol)
	dataStoreProtocol.SetHandlerSearchObject(searchDataStoreObjects)

	mmProtocol := matchmaking.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(mmProtocol)
	mmCommon := common_matchmaking.NewCommonProtocol(mmProtocol)
	mmCommon.SetManager(globals.MatchmakingManager)
	mmProtocol.SetHandlerFindByOwner(findLobbies)
	mmProtocol.SetHandlerGetParticipants(emptyParticipants)
	mmProtocol.SetHandlerGetDetailedParticipants(emptyDetailedParticipants)

	mmExtProtocol := matchmaking_ext.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(mmExtProtocol)
	mmExtCommon := common_matchmaking_ext.NewCommonProtocol(mmExtProtocol)
	mmExtCommon.SetManager(globals.MatchmakingManager)

	extProtocol := matchmake_extension.NewProtocol()
	SecureEndpoint.RegisterServiceProtocol(extProtocol)
	extCommon := common_matchmake_extension.NewCommonProtocol(extProtocol)
	extCommon.SetManager(globals.MatchmakingManager)
	// Attribs[2] encodes the region. Clearing it before the search runs lets
	// MH3U and MH3G players match into the same rooms across regions.
	extCommon.CleanupMatchmakeSessionSearchCriterias = func(searchCriterias types.List[matchmakingtypes.MatchmakeSessionSearchCriteria]) {
		for i := range searchCriterias {
			if len(searchCriterias[i].Attribs) > 2 {
				searchCriterias[i].Attribs[2] = ""
			}
		}
	}
	extCommon.CleanupSearchMatchmakeSession = func(*matchmakingtypes.MatchmakeSession) {}
	// MH3U uses game-specific notification type values outside the generic
	// common handler's hard-coded 101-108 range. Retail expects an empty
	// success response here; returning Core::InvalidArgument crashes the
	// online bootstrap.
	extProtocol.SetHandlerUpdateNotificationData(updateMH3UNotificationData)
	// MH3U expects a legacy PersistentGathering whose final participation
	// count includes two system occupants. A zero value underflows the
	// population calculation in the retail world-select code.
	extProtocol.SetHandlerFindOfficialCommunity(findOfficialCommunitiesMH3U)
	extProtocol.SetHandlerFindCommunityByGatheringID(findMH3UCommunitiesByID)
	extProtocol.SetHandlerJoinCommunity(joinMH3UCommunity)
	legacy := &legacyMatchmakeProtocol{create: createMH3UMatchmakeSession, joinEx: extProtocol.JoinMatchmakeSessionEx}
	legacy.SetEndpoint(SecureEndpoint)
	extProtocol.Patches = legacy
	extProtocol.PatchedMethods = []uint32{matchmake_extension.MethodCreateMatchmakeSession, matchmake_extension.MethodJoinMatchmakeSessionEx}

	// MH3U has no local chat echo: DeliverMessage must be pushed back to all
	// participants in the current room (including the sender), then ACKed.
	messageProtocol := &mh3uMessageDeliveryProtocol{}
	SecureEndpoint.RegisterServiceProtocol(messageProtocol)

	seedOfficialCommunities()
}

func updateMH3UNotificationData(_ error, packet nex.PacketInterface, callID uint32, uiType types.UInt32, uiParam1 types.UInt32, uiParam2 types.UInt32, strParam types.String) (*nex.RMCMessage, *nex.Error) {
	fmt.Printf("[MH3U Notify] type=%d param1=%d param2=%d text=%q\n", uint32(uiType), uint32(uiParam1), uint32(uiParam2), string(strParam))
	response := nex.NewRMCSuccess(packet.Sender().Endpoint(), nil)
	response.ProtocolID = matchmake_extension.ProtocolID
	response.MethodID = matchmake_extension.MethodUpdateNotificationData
	response.CallID = callID
	return response, nil
}

func findOfficialCommunitiesMH3U(err error, packet nex.PacketInterface, callID uint32, _ types.Bool, _ types.ResultRange) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	endpoint := packet.Sender().Endpoint()
	community := newMH3UCommunity(0x101)
	communities := types.NewList[matchmakingtypes.PersistentGathering]()
	communities = append(communities, community)
	out := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
	communities.WriteTo(out)

	response := nex.NewRMCSuccess(endpoint, out.Bytes())
	response.ProtocolID = matchmake_extension.ProtocolID
	response.MethodID = matchmake_extension.MethodFindOfficialCommunity
	response.CallID = callID
	return response, nil
}

func newMH3UCommunity(gid uint32) matchmakingtypes.PersistentGathering {
	community := matchmakingtypes.NewPersistentGathering()
	community.Gathering.ID = types.NewUInt32(gid)
	community.Gathering.OwnerPID = types.NewPID(2)
	community.Gathering.HostPID = types.NewPID(2)
	community.Gathering.MinimumParticipants = types.NewUInt16(1)
	maximum, population, label := uint16(18), uint32(2), "Gathering Hall 1"
	if gid == 0x201 {
		maximum, population, label = 17, 1, "Lobby 1"
	}
	community.Gathering.MaximumParticipants = types.NewUInt16(maximum)
	community.Gathering.ParticipationPolicy = types.NewUInt32(1)
	community.Gathering.PolicyArgument = types.NewUInt32(0)
	community.Gathering.Flags = types.NewUInt32(512)
	community.Gathering.State = types.NewUInt32(0)
	community.Gathering.Description = types.NewString(label + ":" + label + ":" + label + ":" + label + ":" + label + ":" + label + ":" + label + ":" + label)
	community.CommunityType = types.NewUInt32(1)
	community.Password = types.NewString("")
	community.Attribs = types.NewList[types.UInt32]()
	for _, value := range []uint32{gid, 0, 0xFFFFFFFF, 0, 0, 0} {
		community.Attribs = append(community.Attribs, types.NewUInt32(value))
	}
	community.ApplicationBuffer = types.NewBuffer(nil)
	community.ParticipationStartDate = community.ParticipationStartDate.FromTimestamp(time.Date(2013, 1, 1, 0, 0, 0, 0, time.UTC))
	community.ParticipationEndDate = community.ParticipationEndDate.FromTimestamp(time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC))
	community.MatchmakeSessionCount = types.NewUInt32(0)
	community.ParticipationCount = types.NewUInt32(population)
	return community
}

func findMH3UCommunitiesByID(err error, packet nex.PacketInterface, callID uint32, gids types.List[types.UInt32]) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	communities := types.NewList[matchmakingtypes.PersistentGathering]()
	if len(gids) == 0 {
		communities = append(communities, newMH3UCommunity(0x101))
	} else {
		for _, gid := range gids {
			if uint32(gid) == 0x101 || uint32(gid) == 0x201 {
				communities = append(communities, newMH3UCommunity(uint32(gid)))
			}
		}
	}
	fmt.Printf("[MH3U Community] resolve gids=%v results=%d\n", gids, len(communities))
	endpoint := packet.Sender().Endpoint()
	out := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
	communities.WriteTo(out)
	response := nex.NewRMCSuccess(endpoint, out.Bytes())
	response.ProtocolID = matchmake_extension.ProtocolID
	response.MethodID = matchmake_extension.MethodFindCommunityByGatheringID
	response.CallID = callID
	return response, nil
}

func joinMH3UCommunity(err error, packet nex.PacketInterface, callID uint32, gid types.UInt32, _ types.String, _ types.String) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}
	if uint32(gid) != 0x101 && uint32(gid) != 0x201 {
		return nil, nex.NewError(nex.ResultCodes.RendezVous.InvalidGID, "unknown MH3U community")
	}

	if _, nexErr := matchmake_database.UpdatePersistentGatheringParticipationCount(globals.MatchmakingManager, packet.Sender().PID(), uint32(gid)); nexErr != nil {
		return nil, nexErr
	}

	fmt.Printf("[MH3U Community] PID=%d joined gid=0x%X\n", uint64(packet.Sender().PID()), uint32(gid))
	response := nex.NewRMCSuccess(packet.Sender().Endpoint(), nil)
	response.ProtocolID = matchmake_extension.ProtocolID
	response.MethodID = matchmake_extension.MethodJoinCommunity
	response.CallID = callID
	return response, nil
}

func createMH3UMatchmakeSession(err error, packet nex.PacketInterface, callID uint32, holder matchmakingtypes.GatheringHolder, message types.String, participationCount types.UInt16) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}
	if len(message) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "MH3U room message too long")
	}

	session, ok := holder.Object.(matchmakingtypes.MatchmakeSession)
	if !ok {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "MH3U expected MatchmakeSession")
	}
	// The retail MH3U room wire has nine attributes. The common Pretendo
	// validator only accepts six and therefore rejects every valid room.
	if len(session.Attributes) != 9 || len(session.ApplicationBuffer) > 512 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "invalid MH3U room shape")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	manager := globals.MatchmakingManager
	manager.Mutex.Lock()
	defer manager.Mutex.Unlock()

	matchmake_database.EndMatchmakeSessionsParticipation(manager, connection)
	if nexErr := matchmake_database.CreateMatchmakeSession(manager, connection, &session); nexErr != nil {
		fmt.Printf("[MH3U Room] create failed: %s\n", nexErr.Error())
		return nil, nexErr
	}
	participants, nexErr := matchmake_database.JoinMatchmakeSession(manager, session, connection, uint16(participationCount), string(message))
	if nexErr != nil {
		fmt.Printf("[MH3U Room] host join failed gid=0x%X: %s\n", uint32(session.Gathering.ID), nexErr.Error())
		return nil, nexErr
	}
	session.ParticipationCount = types.NewUInt32(participants)

	endpoint := packet.Sender().Endpoint()
	out := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
	session.Gathering.ID.WriteTo(out)
	session.SessionKey.WriteTo(out)
	response := nex.NewRMCSuccess(endpoint, out.Bytes())
	response.ProtocolID = matchmake_extension.ProtocolID
	response.MethodID = matchmake_extension.MethodCreateMatchmakeSession
	response.CallID = callID
	fmt.Printf("[MH3U Room] created gid=0x%X host=%d attrs=%d participants=%d\n", uint32(session.Gathering.ID), uint64(connection.PID()), len(session.Attributes), participants)
	return response, nil
}

func seedOfficialCommunities() {
	// One World (0x101) and one child Lobby (0x201). The eight repeated name
	// segments avoid the retail EU client's multilingual name-parser overread.
	name := "Gathering Hall 1:Gathering Hall 1:Gathering Hall 1:Gathering Hall 1:Gathering Hall 1:Gathering Hall 1:Gathering Hall 1:Gathering Hall 1"
	for _, row := range []struct {
		id   int
		desc string
		max  int
	}{{0x101, name, 18}, {0x201, "Lobby 1", 17}} {
		_, err := database.Postgres.Exec(`INSERT INTO matchmaking.gatherings
			(id,owner_pid,host_pid,min_participants,max_participants,participation_policy,policy_argument,flags,state,description,registered,type,started_time,participants)
			VALUES($1,2,2,1,$3,1,0,512,0,$2,true,'PersistentGathering',NOW(),array[]::numeric(10)[])
			ON CONFLICT(id) DO UPDATE SET description=EXCLUDED.description,max_participants=EXCLUDED.max_participants,registered=true`, row.id, row.desc, row.max)
		if err != nil {
			globals.Logger.Errorf("[MH3U] seed gathering: %v", err)
			continue
		}
		_, err = database.Postgres.Exec(`INSERT INTO matchmaking.persistent_gatherings
			(id,community_type,password,attribs,application_buffer,participation_start_date,participation_end_date)
			VALUES($1,2,'',ARRAY[$1,0,4294967295,0,0,0]::bigint[],'',TIMESTAMP '2013-01-01',TIMESTAMP '2099-12-31')
			ON CONFLICT(id) DO UPDATE SET community_type=2,attribs=EXCLUDED.attribs`, row.id)
		if err != nil {
			globals.Logger.Errorf("[MH3U] seed persistent gathering: %v", err)
		}
	}
	_, _ = database.Postgres.Exec(`SELECT setval(pg_get_serial_sequence('matchmaking.gatherings','id'), GREATEST((SELECT MAX(id) FROM matchmaking.gatherings), 513), true)`)
}

func findLobbies(err error, packet nex.PacketInterface, callID uint32, _ types.PID, _ types.ResultRange) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}
	holders := types.NewList[matchmakingtypes.GatheringHolder]()
	holder := matchmakingtypes.NewGatheringHolder()
	holder.Object = newMH3UCommunity(0x201)
	holders = append(holders, holder)
	fmt.Printf("[MH3U Community] returning lobby gid=0x201\n")
	out := nex.NewByteStreamOut(SecureEndpoint.LibraryVersions(), SecureEndpoint.ByteStreamSettings())
	holders.WriteTo(out)
	response := nex.NewRMCSuccess(SecureEndpoint, out.Bytes())
	response.ProtocolID = matchmaking.ProtocolID
	response.MethodID = matchmaking.MethodFindByOwner
	response.CallID = callID
	return response, nil
}

func emptyParticipants(err error, _ nex.PacketInterface, callID uint32, _ types.UInt32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}
	out := nex.NewByteStreamOut(SecureEndpoint.LibraryVersions(), SecureEndpoint.ByteStreamSettings())
	types.NewList[types.PID]().WriteTo(out)
	r := nex.NewRMCSuccess(SecureEndpoint, out.Bytes())
	r.ProtocolID = matchmaking.ProtocolID
	r.MethodID = matchmaking.MethodGetParticipants
	r.CallID = callID
	return r, nil
}

func emptyDetailedParticipants(err error, _ nex.PacketInterface, callID uint32, _ types.UInt32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}
	out := nex.NewByteStreamOut(SecureEndpoint.LibraryVersions(), SecureEndpoint.ByteStreamSettings())
	types.NewList[matchmakingtypes.ParticipantDetails]().WriteTo(out)
	r := nex.NewRMCSuccess(SecureEndpoint, out.Bytes())
	r.ProtocolID = matchmaking.ProtocolID
	r.MethodID = matchmaking.MethodGetDetailedParticipants
	r.CallID = callID
	return r, nil
}
