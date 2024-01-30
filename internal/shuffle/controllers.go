package shuffle

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"text/template"
	"time"
)

// A song that is added to the end of the queue to indicate which songs should be shuffled.
// This is necessary because the Spotify API includes all the songs that will play after the
// queue is done in the GET /me/player/queue endpoint, and there is no way to distinguish
// beteen the actual queue and the rest of the songs.
const delimiter = "spotify:track:5zOKuItOTZhRCGtPrDYmlj"

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

// Create a request to the Spotify API.
// A delay is added to the request in an attempt to avoid flaky behaviour.
func doRequest(client *http.Client, token string, method string, path string, query *url.Values) (*http.Response, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("https://api.spotify.com/v1%s", path), nil)

	if err != nil {
		fmt.Println("Error creating request")
	}

	if query != nil {
		req.URL.RawQuery = query.Encode()
	}

	req.Header.Add(
		"Authorization",
		fmt.Sprintf("Bearer %s", token),
	)

	res, err := client.Do(req)

	// Add a delay to the request to avoid flaky behaviour.
	time.Sleep(1000 * time.Millisecond)

	return res, err
}

func shuffleQueue(token string) error {
	client := &http.Client{}

	addToQueue(client, token, delimiter)

	var URIs []string
	var qErr error

	// Try to get the queue up to 5 times, with a delay of one second between each attempt.
	// This is necessary because it might sometimes take longer than expected for the delimiter
	// to be added to the queue.
	for i := 0; i < 5; i++ {
		URIs, qErr = getQueue(client, token)

		if qErr == nil {
			break
		}

		fmt.Printf("Error getting queue (attempt %d): %v\n", i+1, qErr)
	}

	// If there are no songs in the queue, return early since there is nothing to do.
	// This will mean that the delimiter will be left in the queue, but that is the
	// users punishment for using the website incorrectly.
	if len(URIs) == 0 {
		fmt.Println("No songs in queue")
		return nil
	}

	// Shuffle the queue.
	rand.Shuffle(len(URIs), func(i, j int) {
		URIs[i], URIs[j] = URIs[j], URIs[i]
	})

	// Re-add the songs to the queue.
	for _, s := range URIs {
		aErr := addToQueue(client, token, s)

		if aErr != nil {
			fmt.Println("Error adding song to queue:", aErr)
			return aErr
		}
	}

	// Skip the songs in the old, unshuffled, queue.
	// Break when all songs in the queue, plus the delimiter and the
	// currently playing song, have been skipped.
	i := 0
	for i != len(URIs)+2 {
		sRes, sErr := doRequest(client, token, "POST", "/me/player/next", nil)

		if sErr != nil || !(sRes.StatusCode == http.StatusNoContent || sRes.StatusCode == http.StatusAccepted) {
			fmt.Println("Error skipping song", sRes.Status, sErr)
			return sErr
		}

		i++
	}

	return nil
}

func getQueue(client *http.Client, token string) ([]string, error) {
	qRes, qErr := doRequest(client, token, "GET", "/me/player/queue", nil)

	if qErr != nil {
		return nil, qErr
	}

	if qRes.StatusCode != http.StatusOK {
		fmt.Println("Error getting queue", qRes.Status)
		return nil, errors.New("error getting queue")
	}

	// Extract URIs from the "queue" array
	var parsed queueResponse

	if err := json.NewDecoder(qRes.Body).Decode(&parsed); err != nil {
		fmt.Println("Error decoding response body:", err)
		return nil, err
	}

	var delimiterFound bool

	var URIs []string
	for _, s := range parsed.Queue {
		// Stop adding songs to the list of URIs the first time that the delimiter is seen.
		// This assumes that that particular song wasn't in the queue before we added it above,
		// but people shouldn't listen to that song, so I don't see it as an issue.
		if s.URI == delimiter {
			delimiterFound = true
			break
		}

		URIs = append(URIs, s.URI)
	}

	if !delimiterFound {
		return nil, errors.New("delimiter not found in queue")
	}

	return URIs, nil
}

func addToQueue(client *http.Client, token string, uri string) error {
	query := url.Values{
		"uri": {uri},
	}

	sRes, sErr := doRequest(client, token, "POST", "/me/player/queue", &query)

	if sErr != nil {
		return sErr
	}

	if sRes.StatusCode != http.StatusNoContent {
		fmt.Println("Error adding song to queue", sRes.Status)
		return errors.New("error adding song to queue")
	}

	return nil
}
