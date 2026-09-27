package main

import (
	"encoding/hex"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"

	"github.com/PretendoNetwork/plogger-go"

	"github.com/Protarium-Network/mh3u-nex/database"
	"github.com/Protarium-Network/mh3u-nex/globals"
)

func requiredEnv(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		globals.Logger.Errorf("%s environment variable not set", name)
		os.Exit(1)
	}
	return value
}

func requiredPort(name string) {
	value := requiredEnv(name)
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		globals.Logger.Errorf("%s is not a valid UDP port: %s", name, value)
		os.Exit(1)
	}
}

func init() {
	globals.Logger = plogger.NewLogger()

	if err := godotenv.Load(); err != nil {
		globals.Logger.Warning("Error loading .env file")
	}

	if kerberos := strings.TrimSpace(os.Getenv("PN_MH3U_KERBEROS_PASSWORD")); kerberos != "" {
		globals.KerberosPassword = kerberos
	} else {
		globals.Logger.Warningf("PN_MH3U_KERBEROS_PASSWORD not set, using default %q", globals.KerberosPassword)
	}

	globals.LocalAuthMode = os.Getenv("PN_MH3U_LOCAL_MODE") == "1"
	if globals.LocalAuthMode {
		globals.Logger.Warning("Local mode is enabled; use it only in an isolated preservation environment")
	} else {
		var err error
		globals.NEXTokenAESKey, err = hex.DecodeString(requiredEnv("PN_MH3U_NEX_TOKEN_AES_KEY"))
		if err != nil || len(globals.NEXTokenAESKey) != 32 {
			globals.Logger.Critical("PN_MH3U_NEX_TOKEN_AES_KEY must be a 64-character hexadecimal AES-256 key")
			os.Exit(1)
		}
		globals.NEXPasswordSecret, err = hex.DecodeString(requiredEnv("PN_MH3U_NEX_PASSWORD_SECRET"))
		if err != nil || len(globals.NEXPasswordSecret) < 32 {
			globals.Logger.Critical("PN_MH3U_NEX_PASSWORD_SECRET must be at least 32 bytes encoded as hexadecimal")
			os.Exit(1)
		}
	}

	globals.InitAccounts()

	requiredPort("PN_MH3U_AUTH_PORT")
	requiredPort("PN_MH3U_SECURE_PORT")
	requiredEnv("PN_MH3U_SECURE_HOST")

	database.ConnectPostgres()
}
