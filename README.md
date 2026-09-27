# Monster Hunter 3 Ultimate (Wii U) — NEX server

A preservation-oriented NEX server for **Monster Hunter 3 Ultimate** on Wii U
(`game_server_id` `10104d00`, shared across every SKU of the game — the three
original regions (JPN `0005000010117200`, USA `0005000010118300`, EUR
`0005000010104d00`) plus three further title IDs from other releases of the
game (`0005000010111b00`, `0005000010133600`, `0005000010132f00`)). It speaks
the game's PRUDP authentication and secure protocols so the online Gathering
Hall / Lobby flow, room creation and in-room chat work again.

Built on the [Pretendo Network](https://github.com/PretendoNetwork) NEX
libraries (`nex-go`, `nex-protocols-go`, `nex-protocols-common-go`) — the
same stack as this org's other servers.

## Recovered configuration

| Field           | Value |
|-----------------|-------|
| Game server ID  | `10104d00` |
| Access key      | `cb2b2f5a` |
| NEX SDK version | `3.0.5` |
| Wire format     | PRUDP v1, legacy connection signature, no structure headers |

## Scope

- **Ticket Granting** — login / secure-server handoff.
- **Secure Connection**, **Utility**, **NAT Traversal** — baseline
  secure-endpoint handshake and P2P setup.
- **DataStore** — stubbed. `SearchObject` always answers an
  empty-but-successful result; the retail client's online bootstrap queries
  it but MH3U has no real DLC/event object backing.
- **MatchMaking / MatchMakingExt / MatchmakeExtension** — the actual online
  flow: two seeded persistent gatherings (a "Gathering Hall" world and a
  child "Lobby"), room creation and joining, and several MH3U-specific wire
  quirks patched at the protocol level rather than in the common library
  (see [PROTOCOL_COVERAGE.md](PROTOCOL_COVERAGE.md)).
- **Message Delivery** — in-room chat, relayed as opaque bytes to every other
  participant of the sender's current gathering (own protocol handler, not a
  common-go implementation).

## Database

One isolated PostgreSQL database holding only the `matchmaking` / `tracking`
schema (no ranking, no DataStore objects — MH3U doesn't use either). The
schema is created on first start; see
[docs/matchmaking-schema.md](docs/matchmaking-schema.md).

## Running

### Local preservation mode

```bash
cp .env.example .env                     # PN_MH3U_LOCAL_MODE=1 by default
cp settings.example.json settings.json   # add your console's PID + NEX password
docker compose up --build
```

In local mode there is no account server: player NEX passwords come from
`settings.json` and the login token is accepted unconditionally. Use it only
on an isolated network.

### Shared mode

Set `PN_MH3U_LOCAL_MODE` to anything but `1` and provide
`PN_MH3U_NEX_TOKEN_AES_KEY` (64 hex chars) and `PN_MH3U_NEX_PASSWORD_SECRET`
(≥32 bytes hex), matching your account server.

### Without Docker

```bash
go build -o mh3u-nex .
./mh3u-nex
```

### Pointing a console at it

On a `Protarium-Network/account-server` deployment, register the game server
ID against all six known title IDs so any SKU's console gets routed here:

```
PN_GAME_SERVERS=10104d00=<host>:26400:0005000010117200,0005000010118300,0005000010104d00,0005000010111b00,0005000010133600,0005000010132f00
```

`PN_MH3U_SECURE_HOST` **must be short** (~15 chars) — the retail binary
truncates it into a fixed-size buffer.

## License

AGPL-3.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE). No proprietary
Nintendo or Capcom code or assets are included.
