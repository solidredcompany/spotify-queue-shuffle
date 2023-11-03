package shuffle

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
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

	// A song that is added to the end of the queue to indicate which songs should be shuffled.
	// This is necessary because the Spotify API includes all the songs that will play after the
	// queue is done in the GET /me/player/queue endpoint, and there is no way to distinguish
	// beteen the actual queue and the rest of the songs.
	delimiter := "spotify:track:5zOKuItOTZhRCGtPrDYmlj"

	addToQueue(client, token, delimiter)

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
	for _, s := range parsed.Queue {
		// Stop adding songs to the list of URIs the first time that the delimiter is seen.
		// This assumes that that particular song wasn't in the queue before we added it above,
		// but people shouldn't listen to that song, so I don't see it as an issue.
		if s.URI == delimiter {
			break
		}

		URIs = append(URIs, s.URI)
	}

	// If there are no songs in the queue, return early since there is nothing to do.
	// This will mean that the delimiter will be left in the queue, but that is the
	// users punishment for using the website incorrectly.
	if len(URIs) == 0 {
		return nil
	}

	// Shuffle the queue.
	rand.Shuffle(len(URIs), func(i, j int) {
		URIs[i], URIs[j] = URIs[j], URIs[i]
	})

	// Re-add the songs to the queue.
	for _, s := range URIs {
		addToQueue(client, token, s)
	}

	// Skip the songs in the old, unshuffled, queue.
	// Break when all songs in the queue, plus the delimiter and the
	// currently playing song, have been skipped.
	i := 0
	for i != len(URIs)+2 {
		i++

		req, err := http.NewRequest("POST", "https://api.spotify.com/v1/me/player/next", nil)

		if err != nil {
			fmt.Println("Error creating request")
		}

		req.Header.Add(
			"Authorization",
			fmt.Sprintf("Bearer %s", token),
		)

		sRes, sErr := client.Do(req)

		if sErr != nil || sRes.StatusCode != http.StatusNoContent {
			fmt.Println("Error skipping song", sRes.Status)
		}
	}

	return nil
}

func addToQueue(client *http.Client, token string, uri string) error {
	req, err := http.NewRequest("POST", "https://api.spotify.com/v1/me/player/queue", nil)

	if err != nil {
		fmt.Println("Error creating request")
	}

	req.URL.RawQuery = url.Values{
		"uri": {uri},
	}.Encode()

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

	return nil
}
