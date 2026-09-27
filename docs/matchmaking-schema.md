# The matchmaking / tracking schema

`nex-protocols-common-go` contains all the SQL for online matchmaking, but it
**expects the `matchmaking` and `tracking` schemas to already exist** - it
never runs `CREATE TABLE`. Pretendo applies that schema out of band.

So `database/init_postgres.go` creates it here. Every column type was read
back from what the library's queries `Scan` into; this is a reconstruction,
not an official schema. Unlike most sibling servers in this project, MH3U has
no ranking or DataStore tables - `SearchObject` always answers an
empty-but-successful result (see `nex/datastore.go`), so this is the only
schema this server needs.

## `matchmaking` schema

| Table                      | Purpose |
|-----------------------------|---------|
| `gatherings`                | Base row for every gathering (matchmake session or persistent gathering). Holds owner/host PID, participant limits, policy, flags, state, `participants bigint[]`, `registered`, `started_time`. |
| `matchmake_sessions`        | 1:1 with `gatherings` for `type='MatchmakeSession'` - the player-created rooms. Game mode, `attribs bigint[]`, open-participation flag, matchmake system type, application buffer, progress score, session key, option, matchmake param, passwords, `refer_gid`, codeword. |
| `persistent_gatherings`     | 1:1 with `gatherings` for `type='PersistentGathering'` - the two seeded Gathering Hall / Lobby communities (`nex/secure.go`'s `seedOfficialCommunities`). Community type, password, `attribs`, application buffer, participation window. |
| `community_participations`  | Per-user participation counter for a persistent gathering. PK `(user_pid, gathering_id)`. |
| `notifications`             | Last notification event per `(source_pid, type)`, `active` flag. |

`matchmake_sessions.id` and `persistent_gatherings.id` are FKs to
`gatherings.id` with `ON DELETE CASCADE`.

## `tracking` schema

Append-only event logs, one table per event: `register_gathering`,
`unregister_gathering`, `join_gathering`, `leave_gathering`,
`disconnect_gathering`, `change_host`, `change_owner`, `notification_data`,
`participate_community`. Each has a `bigserial` id, a `date`, a `source_pid`
and event-specific columns.

## If matchmaking misbehaves

If a matchmaking call fails with a SQL error, compare the failing query in
`nex-protocols-common-go`'s `matchmake-extension/database` package against
the DDL here - a column type mismatch is the most likely cause.
