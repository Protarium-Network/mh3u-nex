package database

import (
	"os"

	"github.com/Protarium-Network/mh3u-nex/globals"
)

// initPostgres creates the schema this server needs on first run. Every
// statement is idempotent, so it is safe to run on every start.
//
// MH3U has no ranking or DataStore backing (SearchObject always answers an
// empty-but-successful result - see nex/datastore.go), so unlike most sibling
// servers this database holds only the matchmaking/tracking schema.
//
// The matchmaking.* / tracking.* tables are hand-authored from the SQL in
// nex-protocols-common-go (which expects the schema to already exist and
// never creates it). Column types follow what those queries scan into. See
// docs/matchmaking-schema.md.
func initPostgres() {
	mustExec := func(label, query string) {
		if _, err := Postgres.Exec(query); err != nil {
			globals.Logger.Criticalf("%s: %s", label, err.Error())
			os.Exit(1)
		}
	}

	mustExec("matchmaking schema", `CREATE SCHEMA IF NOT EXISTS matchmaking`)
	mustExec("tracking schema", `CREATE SCHEMA IF NOT EXISTS tracking`)

	mustExec("matchmaking.gatherings", `CREATE TABLE IF NOT EXISTS matchmaking.gatherings (
		id                   serial PRIMARY KEY,
		owner_pid            bigint,
		host_pid             bigint,
		min_participants     integer,
		max_participants     integer,
		participation_policy bigint,
		policy_argument      bigint,
		flags                bigint,
		state                bigint,
		description          text,
		type                 text,
		participants         bigint[] NOT NULL DEFAULT '{}',
		registered           boolean  NOT NULL DEFAULT true,
		started_time         timestamp
	)`)
	mustExec("matchmaking.gatherings indexes", `
		CREATE INDEX IF NOT EXISTS matchmaking_gatherings_registered_type_idx
			ON matchmaking.gatherings (registered, type);
		CREATE INDEX IF NOT EXISTS matchmaking_gatherings_participants_idx
			ON matchmaking.gatherings USING gin (participants)
	`)

	mustExec("matchmaking.matchmake_sessions", `CREATE TABLE IF NOT EXISTS matchmaking.matchmake_sessions (
		id                      integer PRIMARY KEY REFERENCES matchmaking.gatherings(id) ON DELETE CASCADE,
		game_mode               bigint,
		attribs                 bigint[] NOT NULL DEFAULT '{}',
		open_participation      boolean,
		matchmake_system_type   bigint,
		application_buffer      bytea,
		progress_score          integer,
		session_key             bytea,
		option_zero             bigint,
		matchmake_param         bytea,
		user_password           text,
		refer_gid               bigint,
		user_password_enabled   boolean,
		system_password_enabled boolean,
		codeword                text
	)`)

	mustExec("matchmaking.persistent_gatherings", `CREATE TABLE IF NOT EXISTS matchmaking.persistent_gatherings (
		id                       integer PRIMARY KEY REFERENCES matchmaking.gatherings(id) ON DELETE CASCADE,
		community_type           bigint,
		password                 text,
		attribs                  bigint[] NOT NULL DEFAULT '{}',
		application_buffer       bytea,
		participation_start_date timestamp,
		participation_end_date   timestamp
	)`)

	mustExec("matchmaking.community_participations", `CREATE TABLE IF NOT EXISTS matchmaking.community_participations (
		user_pid            bigint,
		gathering_id        bigint,
		participation_count integer NOT NULL DEFAULT 0,
		PRIMARY KEY (user_pid, gathering_id)
	)`)

	mustExec("matchmaking.notifications", `CREATE TABLE IF NOT EXISTS matchmaking.notifications (
		source_pid bigint,
		type       bigint,
		param_1    bigint,
		param_2    bigint,
		param_str  text,
		active     boolean NOT NULL DEFAULT true,
		PRIMARY KEY (source_pid, type)
	)`)

	for _, ddl := range []string{
		`CREATE TABLE IF NOT EXISTS tracking.register_gathering (
			id bigserial PRIMARY KEY, date timestamp, source_pid bigint, gathering_id bigint)`,
		`CREATE TABLE IF NOT EXISTS tracking.unregister_gathering (
			id bigserial PRIMARY KEY, date timestamp, source_pid bigint, gathering_id bigint)`,
		`CREATE TABLE IF NOT EXISTS tracking.join_gathering (
			id bigserial PRIMARY KEY, date timestamp, source_pid bigint, gathering_id bigint,
			new_participants bigint[], total_participants bigint[])`,
		`CREATE TABLE IF NOT EXISTS tracking.leave_gathering (
			id bigserial PRIMARY KEY, date timestamp, source_pid bigint, gathering_id bigint,
			total_participants bigint[])`,
		`CREATE TABLE IF NOT EXISTS tracking.disconnect_gathering (
			id bigserial PRIMARY KEY, date timestamp, source_pid bigint, gathering_id bigint,
			total_participants bigint[])`,
		`CREATE TABLE IF NOT EXISTS tracking.change_host (
			id bigserial PRIMARY KEY, date timestamp, source_pid bigint, gathering_id bigint,
			old_host_pid bigint, new_host_pid bigint)`,
		`CREATE TABLE IF NOT EXISTS tracking.change_owner (
			id bigserial PRIMARY KEY, date timestamp, source_pid bigint, gathering_id bigint,
			old_owner_pid bigint, new_owner_pid bigint)`,
		`CREATE TABLE IF NOT EXISTS tracking.notification_data (
			id bigserial PRIMARY KEY, date timestamp, source_pid bigint, type bigint,
			param_1 bigint, param_2 bigint, param_str text)`,
		`CREATE TABLE IF NOT EXISTS tracking.participate_community (
			id bigserial PRIMARY KEY, date timestamp, source_pid bigint, community_gid bigint,
			gathering_id bigint, participation_count integer)`,
	} {
		mustExec("tracking table", ddl)
	}

	globals.Logger.Success("Postgres schema ready")
}
