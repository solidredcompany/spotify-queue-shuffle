package shuffle

// This intentionally only includes the fields that are useful for this website.
type queueResponse struct {
	Queue []struct {
		URI string `json:"uri"`
	} `json:"queue"`
}
