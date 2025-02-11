package auth

import (
	"log"
	"os"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

type EnvConfig struct {
	LadonHost     string
	ClientID      string
	ClientSecret  string
	Issuer        string
	SessionSecret []byte
}

const LADON_HOST_ENV_KEY = "LADON_DOMAIN"
const OIDC_ID_ENV_KEY = "OIDC_CLIENT_ID"
const OIDC_SECRET_ENV_KEY = "OIDC_CLIENT_SECRET"
const OIDC_ISSUER_ENV_KEY = "OIDC_ISSUER"
const SESSION_SECRET = "SESSION_SECRET"

func ensureEnvVar(key string) string {
	val, isSet := os.LookupEnv(key)

	if !isSet {
		log.Fatalf("%s is not set in environment", key)
	}

	return val
}

func EnvMustParse() *EnvConfig {
	sessionSecret := os.Getenv(SESSION_SECRET)

	if len(sessionSecret) != 16 {
		log.Fatalf("session secret must be 16 characters")
	} else if sessionSecret == "" {
		log.Println("ladon: no session secret set, generating one")
		sessionSecret = gonanoid.Must(16)
	}

	return &EnvConfig{
		LadonHost:     ensureEnvVar(LADON_HOST_ENV_KEY),
		ClientID:      ensureEnvVar(OIDC_ID_ENV_KEY),
		ClientSecret:  ensureEnvVar(OIDC_SECRET_ENV_KEY),
		Issuer:        ensureEnvVar(OIDC_ISSUER_ENV_KEY),
		SessionSecret: []byte(sessionSecret),
	}
}
