package storage

type URLRecord struct {
	ID       string `json:"uuid" format:"uuid"`
	Original string `json:"original_url"`
	Short    string `json:"short_url"`
}
