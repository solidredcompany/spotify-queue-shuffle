package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
)

const (
	clientId     = "0d1e2eb6bfdb406d90dbe9369bedd7af"
	clientSecret = "421a74412dea42b18ac45a3c9c4fb400"
	redirectUri  = "http://localhost:8080/callback"
)

func handleHome(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("web/templates/home.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t.Execute(w, nil)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateRandomString(16)
	scope := "user-read-private user-read-email"

	q := url.Values{
		"response_type": {"code"},
		"client_id":     {clientId},
		"scope":         {scope},
		"redirect_uri":  {redirectUri},
		"state":         {state},
	}

	http.SetCookie(w, &http.Cookie{Name: "spotify_auth_state", Value: state})

	u := fmt.Sprintf("%s?%s", "https://accounts.spotify.com/authorize", q.Encode())
	http.Redirect(w, r, u, http.StatusFound)
}

func handleRedirect(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	error := r.URL.Query().Get("error")

	if error != "access_denied" {
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
		"redirect_uri": {redirectUri},
	}

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add(
		"Authorization",
		fmt.Sprintf(
			"Basic %s",
			base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", clientId, clientSecret))),
		),
	)

	res, err := client.Do(req)

	http.Error(w, err.Error(), http.StatusInternalServerError)

	var responseBody map[string]interface{}
	decoder := json.Decoder(res.Body)

	if err := decoder.Decode(&responseBody); err != nil {
		fmt.Println("Error decoding response body:", err)
		return
	}

	access_token, access_token_exists := responseBody["access_token"].(string)
	refresh_token, refresh_token_exists := responseBody["refresh_token"].(string)

	if access_token_exists && refresh_token_exists {
		http.SetCookie(w, &http.Cookie{Name: "access_token", Value: access_token})
		http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: refresh_token})
	}
}

func generateRandomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func main() {
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets/"))))

	http.HandleFunc("/", handleHome)
	http.HandleFunc("/login", handleLogin)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
