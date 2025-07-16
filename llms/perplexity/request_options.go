package perplexity

import (
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

type RequestOptions struct {
	SearchMode        SearchMode `structs:"search_mode,omitempty"`
	SearchRecency     string     `structs:"search_recency_filter,omitempty"`
	BeforeDate        date       `structs:"search_before_date_filter,string,omitempty"`
	AfterDate         date       `structs:"search_after_date_filter,string,omitempty"`
	LastUpdatedBefore date       `structs:"last_updated_before_filter,string,omitempty"`
	LastUpdatedAfter  date       `structs:"last_updated_after_filter,string,omitempty"`
	WebSearch         WebSearch  `structs:"web_search_options,omitempty"`
}

type WebSearch struct {
	ContextSize  ContextSize  `structs:"search_context_size,omitempty"`
	UserLocation UserLocation `structs:"user_location,omitempty"`
}

type UserLocation struct {
	Latitude  float64 `structs:"latitude,omitempty"`
	Longitude float64 `structs:"longitude,omitempty"`
	Country   string  `structs:"country,omitempty"`
}

func (RequestOptions) ProviderRequestOptions() {}

type date struct {
	time.Time
}

func NewDate(t time.Time) date {
	return date{t}
}

func (t date) String() string {
	return t.Format("1/2/2006")
}
