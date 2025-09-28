//go:generate go tool templ generate
//go:generate go tool go-tw -i tailwind.css -o static/styles.css

package main

import (
	"embed"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"

	_ "github.com/joho/godotenv/autoload"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sblinch/kdl-go"
	"github.com/zitadel/logging"

	"github.com/a-h/templ"

	"ladon/auth"
	"ladon/views"
)

//go:embed static
var content embed.FS

func ServeRoot(am *auth.AuthManager) http.Handler {
	f, err := os.Open("./data/links.kdl")
	if err != nil {
		am.Log.Error("ladon: failed to open KDL config")
		panic(err)
	}

	doc, err := kdl.Parse(f)
	if err != nil {
		am.Log.Error("ladon: failed to pase KDL config")
		panic(err)
	}

	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			claims, err := am.GetSession(r)

			if errors.Is(err, auth.ErrNoSession) {
				templ.Handler(views.Authenticate()).ServeHTTP(w, r)
				return
			} else if errors.Is(err, auth.ErrSessionExpired) {
				am.DeleteSession(w)
				am.HandleLogin().ServeHTTP(w, r)
				return
			} else if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			templ.Handler(views.Links(claims.PreferredUsername, doc)).ServeHTTP(w, r)
		},
	)
}

func main() {
	logger := slog.New(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		}),
	)

	am := auth.NewAuthManager(logger)

	// Handle static assets
	fs := http.FileServer(http.FS(content))
	http.Handle("/static/", fs)

	// Serve pages
	http.Handle("/", ServeRoot(am))

	// Handle authentication
	http.Handle("/login", am.HandleLogin())
	http.Handle("/logout", am.HandleLogout())
	http.Handle("/callback", am.HandleCallback())

	mw := logging.Middleware(
		logging.WithLogger(logger),
		logging.WithGroup("server"),
		logging.WithIDFunc(func() slog.Attr {
			return slog.String("id", gonanoid.Must())
		}),
	)

	log.Println("Listening on port 4000")
	if err := http.ListenAndServe(":4000", mw(http.DefaultServeMux)); err != nil {
		panic(err)
	}
}
