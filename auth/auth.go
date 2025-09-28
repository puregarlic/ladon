package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/zitadel/logging"
	"github.com/zitadel/oidc/v3/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

const SESSION_NAME = "_ladon_session"

var (
	ErrNoSession      = errors.New("ladon: no session cookie set")
	ErrSessionExpired = errors.New("ladon: session expired")
)

func State() string {
	return gonanoid.Must()
}

type AuthManager struct {
	Db            *badger.DB
	Env           *EnvConfig
	CookieHandler *httphelper.CookieHandler
	RelyingParty  rp.RelyingParty
	Log           *slog.Logger
	HttpClient    *http.Client
}

func NewAuthManager(logger *slog.Logger) *AuthManager {
	env := EnvMustParse()

	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		panic(err)
	}

	cookieHandler := httphelper.NewCookieHandler(env.SessionSecret, env.SessionSecret, httphelper.WithUnsecure())

	client := &http.Client{
		Timeout: time.Minute,
	}

	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(5 * time.Second)),
		rp.WithHTTPClient(client),
		rp.WithLogger(logger),
		rp.WithSigningAlgsFromDiscovery(),
	}

	logging.EnableHTTPClient(client,
		logging.WithClientGroup("client"),
	)

	ctx := logging.ToContext(context.TODO(), logger)
	provider, err := rp.NewRelyingPartyOIDC(
		ctx,
		env.Issuer,
		env.ClientID,
		env.ClientSecret,
		fmt.Sprintf("%s/callback", env.LadonHost),
		[]string{"openid profile"},
		options...,
	)
	if err != nil {
		logger.Error("ladon: failed to instantiate relying party client")
		panic(err)
	}

	return &AuthManager{
		Db:            db,
		Env:           env,
		CookieHandler: cookieHandler,
		RelyingParty:  provider,
		Log:           logger,
		HttpClient:    client,
	}
}

func (a *AuthManager) HandleLogin() http.Handler {
	return rp.AuthURLHandler(
		State,
		a.RelyingParty,
	)
}

func (a *AuthManager) HandleCallback() http.Handler {
	return rp.CodeExchangeHandler(
		func(
			w http.ResponseWriter,
			r *http.Request,
			tokens *oidc.Tokens[*oidc.IDTokenClaims],
			state string,
			rp rp.RelyingParty,
		) {
			data, err := json.Marshal(tokens)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			id := tokens.IDTokenClaims.Subject

			fmt.Println("subject:", id)

			if err := a.Db.Update(func(txn *badger.Txn) error {
				e := badger.NewEntry([]byte(id), data).WithTTL(time.Duration(tokens.ExpiresIn) * time.Second)
				err := txn.SetEntry(e)
				return err
			}); err != nil {
				http.Error(w, "failed to cache user session", http.StatusInternalServerError)
			}

			if err := a.CookieHandler.SetCookie(w, SESSION_NAME, id); err != nil {
				http.Error(w, "failed to set session cookie", http.StatusInternalServerError)
				return
			}

			w.Header().Add("Location", "/")
			w.WriteHeader(http.StatusFound)
		},
		a.RelyingParty,
	)
}

func (a *AuthManager) HandleLogout() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.CookieHandler.DeleteCookie(w, SESSION_NAME)

		w.Header().Add("Location", "/")
		w.WriteHeader(http.StatusFound)
	})
}

func (a *AuthManager) GetSession(r *http.Request) (*oidc.IDTokenClaims, error) {
	id, err := a.CookieHandler.CheckCookie(r, SESSION_NAME)
	if errors.Is(err, http.ErrNoCookie) {
		return nil, ErrNoSession
	} else if err != nil {
		return nil, err
	}

	tokens := &oidc.Tokens[*oidc.IDTokenClaims]{}
	if err := a.Db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(id))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, tokens)
		})
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	claims, err := rp.VerifyTokens[*oidc.IDTokenClaims](
		context.TODO(),
		tokens.AccessToken,
		tokens.IDToken,
		a.RelyingParty.IDTokenVerifier(),
	)
	if errors.Is(err, oidc.ErrExpired) {
		return nil, ErrSessionExpired
	} else if err != nil {
		return nil, err
	}

	return claims, nil
}

func (a *AuthManager) DeleteSession(w http.ResponseWriter) {
	a.CookieHandler.DeleteCookie(w, SESSION_NAME)
}
