package domain

import "time"

const (
	SearchShareModeKeyword = "keyword"
	SearchShareModeAI      = "ai"

	SearchShareStatusActive  = "active"
	SearchShareStatusRevoked = "revoked"
	SearchShareStatusDeleted = "deleted"
)

type SearchShare struct {
	ID             int64      `json:"id"`
	Token          string     `json:"token,omitempty"`
	CreatedBy      *int64     `json:"created_by,omitempty"`
	CreatorName    string     `json:"creator_name,omitempty"`
	CreatorType    string     `json:"creator_type"`
	CreatedIPHash  string     `json:"-"`
	Title          string     `json:"title"`
	Query          string     `json:"query"`
	Mode           string     `json:"mode"`
	Summary        string     `json:"summary,omitempty"`
	Status         string     `json:"status"`
	IncludePrivate bool       `json:"include_private"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	ViewCount      int64      `json:"view_count"`
	ItemCount      int        `json:"item_count"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type SearchShareItem struct {
	ShareID   int64 `json:"share_id"`
	WebsiteID int64 `json:"website_id"`
	SortOrder int   `json:"sort_order"`
}
