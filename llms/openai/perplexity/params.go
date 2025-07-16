package perplexity

import (
	"encoding/json"
	"time"
)

type (
	SearchMode    string
	RecencyFilter string
	ContextSize   string
)

const (
	SearchModeWeb      SearchMode = "web"
	SearchModeAcademic SearchMode = "academic"
)

const (
	ContextSizeLow    ContextSize = "low"
	ContextSizeMedium ContextSize = "medium"
	ContextSizeHigh   ContextSize = "high"
)

type Params struct {
	SearchMode        SearchMode `json:"search_mode,omitempty"`
	SearchRecency     string     `json:"search_recency_filter,omitempty"`
	BeforeDate        date       `json:"search_before_date_filter,omitzero"`
	AfterDate         date       `json:"search_after_date_filter,omitzero"`
	LastUpdatedBefore date       `json:"last_updated_before_filter,omitzero"`
	LastUpdatedAfter  date       `json:"last_updated_after_filter,omitzero"`
	WebSearch         *WebSearch `json:"web_search_options,omitempty"`
}

type WebSearch struct {
	ContextSize  ContextSize   `json:"search_context_size,omitzero"`
	UserLocation *UserLocation `json:"user_location,omitempty"`
}

type UserLocation struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Country   string  `json:"country,omitempty"`
}

func (Params) Extras() {}

type date struct {
	time.Time
}

func NewDate(t time.Time) date {
	return date{t}
}

func (t date) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Format("1/2/2006"))
}
