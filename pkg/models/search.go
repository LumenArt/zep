package models

type MemorySearchResult struct {
	Message  *Message               `json:"message"`
	Summary  *Summary               `json:"summary"` // reserved for future use
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Dist     float64                `json:"dist"`
}

type MemorySearchPayload struct {
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type DocumentSearchPayload struct {
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type DocumentSearchResult struct {
	Document *Document `json:"document"`
	Dist     float64   `json:"dist"`
}

type DocumentSearchResultPage struct {
	DocumentSearchResults []DocumentSearchResult `json:"results"`
	ResultCount           int                    `json:"result_count"`
	TotalPages            int                    `json:"total_pages"`
	CurrentPage           int                    `json:"current_page"`
}
