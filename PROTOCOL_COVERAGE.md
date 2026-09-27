# Protocol coverage

What this server implements for Monster Hunter 3 Ultimate, and the
MH3U-specific quirks each protocol handler works around.

## Authentication endpoint

| Protocol        | Notes |
|------------------|-------|
| Ticket Granting  | Login / LoginEx / RequestTicket. `ValidateLoginData` accepts the account-server token (or anything, in local mode). |

MH3U uses PRUDP v1's legacy connection signature
(`PRUDPV1Settings.LegacyConnectionSignature = true`) and does not write NEX
structure-header version bytes (`ByteStreamSettings.UseStructureHeader` left
at the library default, `false`).

## Secure endpoint

| Protocol | Coverage |
|----------|----------|
| Secure Connection | baseline handshake, insecure `Register` |
| Utility | baseline |
| NAT Traversal | baseline, via the common protocol |
| DataStore | `SearchObject` only, hand-written to always return zero results — the retail client's online bootstrap calls it but MH3U has no real DataStore-backed objects |
| MatchMaking | `FindByOwner` (returns the seeded Lobby), `GetParticipants` / `GetDetailedParticipants` (hand-written, always empty) |
| MatchMakingExt | via the common manager |
| MatchmakeExtension | community/gathering-hall simulation and room create/join — see below |
| Message Delivery | own protocol handler (not common-go) — see below |

### MatchmakeExtension: the Gathering Hall / Lobby simulation

MH3U's online menu expects two always-present **persistent gatherings**
rather than a dynamic community list: a "Gathering Hall" world
(`gid 0x101`, 18 max, seeded with population 2) and a child "Lobby"
(`gid 0x201`, 17 max, population 1). `nex/secure.go`'s
`seedOfficialCommunities` inserts both directly into
`matchmaking.gatherings` / `matchmaking.persistent_gatherings` on startup
(`ON CONFLICT` upsert, so it's safe to run every start), and
`FindOfficialCommunity` / `FindCommunityByGatheringID` / `JoinCommunity` are
hand-written to serve exactly these two rather than the common library's
generic community list.

The eight repeated name segments in the gathering description
(`"Gathering Hall 1:Gathering Hall 1:..."`) work around the retail EU
client's multilingual name-parser reading past the end of a shorter string.

`UpdateNotificationData` is overridden because MH3U uses game-specific
notification type values outside the common handler's hard-coded 101-108
range; the common handler would reject them, but retail expects an empty
success response, and returning `Core::InvalidArgument` there crashes the
online bootstrap.

### Legacy wire format: `CreateMatchmakeSession` / `JoinMatchmakeSessionEx`

MH3U's client omits the trailing `participationCount` u16 that the
current generated decoder for `MatchmakeExtension` methods 6
(`CreateMatchmakeSession`) and 30 (`JoinMatchmakeSessionEx`) expects, so the
generic decoder rejects an otherwise-valid packet. `nex/legacy_matchmake.go`
patches both methods at the protocol level (`Patches` /
`PatchedMethods`), decoding the legacy (shorter) parameter layout by hand and
forwarding to the real handlers with a synthesized `participationCount = 1`.

The retail room wire also carries **nine** `MatchmakeSession.Attributes`
where the common validator only accepts six — `createMH3UMatchmakeSession` in
`nex/secure.go` checks for exactly nine instead of deferring to the common
validation.

### Message Delivery: opaque chat relay

MH3U has no server-side chat echo: the client expects `DeliverMessage` to be
pushed back to every other participant of the sender's current gathering
(room first, falling back to the smallest hall/lobby the sender is in), then
ACKed to the sender. The `UserMessage` wire shape here predates the generated
NEX type, so `nex/message_delivery.go` implements its own
`ProtocolInterface` and keeps the payload bytes opaque instead of decoding
and re-encoding them (which corrupts the game's tab-delimited shout
payload). Without the empty success ACK, only the first chat message after
login can be sent — the client's send gate never re-arms.

## Not implemented

- **Ranking** — this title has no leaderboard NEX calls.
- **DataStore beyond the `SearchObject` stub** — no evidence of real
  DLC/event objects being fetched through this protocol.
