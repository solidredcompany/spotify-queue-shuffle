package shuffle

type queueResponse struct {
	Queue []struct {
		URI string `json:"uri"`
	} `json:"queue"`
}
