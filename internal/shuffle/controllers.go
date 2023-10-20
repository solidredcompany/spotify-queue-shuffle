package shuffle

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"text/template"
)

func HandleHome(w http.ResponseWriter, r *http.Request) {
	// Check if the user is logged in. Redirect to the sign in page if they are not.
	_, tErr := r.Cookie("access_token")

	if tErr != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
	}

	t, err := template.ParseFiles("web/templates/home.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t.Execute(w, nil)
}

// The logic for when a user does a POST request to /shuffle.
func HandleShuffle(w http.ResponseWriter, r *http.Request) {
	// Check if the user is logged in. Redirect to the sign in page if they are not.
	token, tErr := r.Cookie("access_token")

	if tErr != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
	}

	switch r.Method {
	case "POST":
		shuffleErr := shuffleQueue(token.Value)

		if shuffleErr != nil {
			http.Error(w, shuffleErr.Error(), http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}
}

func shuffleQueue(token string) error {
	client := &http.Client{}

	// Get the user's current queue.
	req, _ := http.NewRequest("GET", "https://api.spotify.com/v1/me/player/queue", nil)
	req.Header.Add(
		"Authorization",
		fmt.Sprintf("Bearer %s", token),
	)

	qRes, qErr := client.Do(req)

	if qErr != nil {
		return qErr
	}

	if qRes.StatusCode != http.StatusOK {
		fmt.Println("Error getting queue", qRes.Status)
		return errors.New("Error getting queue")
	}

	// Extract URIs from the "queue" array
	var parsed queueResponse

	if err := json.NewDecoder(qRes.Body).Decode(&parsed); err != nil {
		fmt.Println("Error decoding response body:", err)
		return err
	}

	var URIs []string
	for _, uri := range parsed.Queue {
		URIs = append(URIs, uri.URI)
	}

	// Shuffle the queue.
	rand.Shuffle(len(URIs), func(i, j int) {
		URIs[i], URIs[j] = URIs[j], URIs[i]
	})

	// Re-add the songs to the queue.
	for _, s := range URIs {
		req, _ := http.NewRequest("POST", "https://api.spotify.com/v1/me/player/queue", nil)
		req.URL.Query().Add("uri", s)
		req.Header.Add(
			"Authorization",
			fmt.Sprintf("Bearer %s", token),
		)

		sRes, sErr := client.Do(req)

		if sErr != nil {
			return sErr
		}

		if sRes.StatusCode != http.StatusNoContent {
			fmt.Println("Error adding song to queue", sRes.Status)
			return errors.New("Error adding song to queue")
		}
	}

	return nil
}
