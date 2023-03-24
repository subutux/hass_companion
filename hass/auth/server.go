package auth

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"

	"github.com/pkg/browser"
)

func StartHttpServer() (Credentials, error) {
	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
		Addr:    ":9999",
	}

	creds := Credentials{}

	log.Printf("Starting server on %s", server.Addr)

	mux.HandleFunc("/hass/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		log.Printf("Received code %s for %s", code, state)
		creds.Server = state
		creds.Token = code
		creds.ClientId = "http://localhost:9999"
		io.WriteString(w, "Authentication successfull! You may close this window.\n")

		go func() {
			if err := server.Shutdown(context.Background()); err != nil {
				log.Fatal(err)
			}
		}()
	})

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return Credentials{}, err
	}
	log.Printf("Finished")

	return creds, nil
}

func Initiate(hass_endpoint string) (Credentials, error) {

	url, err := url.Parse(hass_endpoint)
	if err != nil {
		return Credentials{}, err
	}
	url.Path = path.Join(url.Path, "/auth/authorize")
	params := url.Query()
	params.Add("client_id", "http://localhost:9999")
	params.Add("redirect_uri", "http://localhost:9999/hass/callback")
	params.Add("state", hass_endpoint)
	url.RawQuery = params.Encode()
	log.Printf("Opening browser to %s", url.String())

	browser.OpenURL(url.String())
	if err != nil {
		return Credentials{}, err
	}
	return StartHttpServer()
}
