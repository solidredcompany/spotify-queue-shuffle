package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/solidredcompany/solid-red/websites/queue-shuffle/internal/utils"
)

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("web/templates/login.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t.Execute(w, nil)
}

func HandleAuthenticate(w http.ResponseWriter, r *http.Request) {
	state := utils.GenerateRandomString(16)
	scope := "user-read-playback-state user-modify-playback-state"

	q := url.Values{
		"response_type": {"code"},
		"client_id":     {os.Getenv("SPOTIFY_CLIENT_ID")},
		"scope":         {scope},
		"redirect_uri":  {os.Getenv("SPOTIFY_REDIRECT_URI")},
		"state":         {state},
	}

	http.SetCookie(w, &http.Cookie{Name: "spotify_auth_state", Value: state})

	u := fmt.Sprintf("%s?%s", "https://accounts.spotify.com/authorize", q.Encode())
	http.Redirect(w, r, u, http.StatusFound)
}

func HandleRedirect(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	error := r.URL.Query().Get("error")

	if error != "" {
		http.Error(w, error, http.StatusBadRequest)
		return
	}

	stateCookie, err := r.Cookie("spotify_auth_state")

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if state == "" || state != stateCookie.Value {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	// Remove the state cookie.
	http.SetCookie(w, &http.Cookie{Name: "spotify_auth_state", MaxAge: -1})

	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {os.Getenv("SPOTIFY_REDIRECT_URI")},
	}

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add(
		"Authorization",
		fmt.Sprintf(
			"Basic %s",
			base64.StdEncoding.EncodeToString(
				[]byte(fmt.Sprintf("%s:%s", os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"))),
			),
		),
	)

	res, err := client.Do(req)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	var token TokenResponse

	if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
		fmt.Println("Error decoding response body:", err)
		return
	}

	if token.AccessToken != "" && token.RefreshToken != "" {
		http.SetCookie(w, &http.Cookie{Name: "access_token", Value: token.AccessToken})
		http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: token.RefreshToken})
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
