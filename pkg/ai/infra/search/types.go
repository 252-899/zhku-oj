package search

type SearchInput struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k"`
}

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type Config struct {
	APIKey  string
	BaseURL string
}
