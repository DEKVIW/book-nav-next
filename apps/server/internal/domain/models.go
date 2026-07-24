package domain

import "time"

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Avatar       string    `json:"avatar,omitempty"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) Public() map[string]any {
	if u == nil {
		return nil
	}
	return map[string]any{
		"id":         u.ID,
		"username":   u.Username,
		"email":      u.Email,
		"avatar":     u.Avatar,
		"role":       u.Role,
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
	}
}

type Session struct {
	ID        string
	UserID    int64
	CSRFToken string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type InvitationCode struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	CreatedBy *int64     `json:"created_by,omitempty"`
	UsedBy    *int64     `json:"used_by,omitempty"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
}

type Category struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description,omitempty"`
	Icon         string     `json:"icon,omitempty"`
	Color        string     `json:"color,omitempty"`
	SortOrder    int        `json:"sort_order"`
	DisplayLimit int        `json:"display_limit"`
	ParentID     *int64     `json:"parent_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	Children     []Category `json:"children,omitempty"`
	// Aggregates
	DirectCount           int       `json:"direct_count,omitempty"`
	TotalCountWithChildren int      `json:"total_count_with_children,omitempty"`
	WebsiteCount          int       `json:"website_count,omitempty"`
	Websites              []Website `json:"websites,omitempty"`
	DisplayedSubcategoryID *int64   `json:"displayed_subcategory_id,omitempty"`
}

type Website struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Description string     `json:"description,omitempty"`
	Icon        string     `json:"icon,omitempty"`
	CategoryID  *int64     `json:"category_id,omitempty"`
	CreatedBy   int64      `json:"created_by"`
	IsFeatured  bool       `json:"is_featured"`
	IsPrivate   bool       `json:"is_private"`
	SortOrder   int        `json:"sort_order"`
	Views       int64      `json:"views"`
	IsValid     bool       `json:"is_valid"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ViewerIDs   []int64    `json:"viewer_ids,omitempty"`
	CategoryName string    `json:"category_name,omitempty"`
}

type OperationLog struct {
	ID           int64     `json:"id"`
	UserID       *int64    `json:"user_id,omitempty"`
	Action       string    `json:"action"`
	WebsiteID    *int64    `json:"website_id,omitempty"`
	WebsiteTitle string    `json:"website_title"`
	WebsiteURL   string    `json:"website_url"`
	WebsiteIcon  string    `json:"website_icon"`
	CategoryID   *int64    `json:"category_id,omitempty"`
	CategoryName string    `json:"category_name"`
	DetailsJSON  string    `json:"details_json"`
	CreatedAt    time.Time `json:"created_at"`
}

type Job struct {
	ID          int64      `json:"id"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	Progress    int        `json:"progress"`
	Total       int        `json:"total"`
	Success     int        `json:"success"`
	Failed      int        `json:"failed"`
	PayloadJSON string     `json:"payload_json,omitempty"`
	ResultJSON  string     `json:"result_json,omitempty"`
	Error       string     `json:"error,omitempty"`
	CreatedBy   *int64     `json:"created_by,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type DeadlinkCheck struct {
	ID             int64     `json:"id"`
	BatchID        string    `json:"batch_id"`
	WebsiteID      int64     `json:"website_id"`
	URL            string    `json:"url"`
	IsValid        bool      `json:"is_valid"`
	StatusCode     *int      `json:"status_code,omitempty"`
	ErrorType      string    `json:"error_type,omitempty"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	ResponseTimeMs *int      `json:"response_time_ms,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
	WebsiteTitle   string    `json:"website_title,omitempty"`
}

// PublicSettings is safe to expose to portal.
type PublicSettings struct {
	SiteName              string `json:"site_name"`
	SiteSubtitle          string `json:"site_subtitle,omitempty"`
	SiteLogo              string `json:"site_logo,omitempty"`
	SiteFavicon           string `json:"site_favicon,omitempty"`
	FooterContent         string `json:"footer_content,omitempty"`
	AISearchEnabled       bool   `json:"ai_search_enabled"`
	AIAllowAnon           bool   `json:"ai_search_allow_anonymous"`
	EnableTransition      bool   `json:"enable_transition"`
	TransitionTime        int    `json:"transition_time"`
	AdminTransition       int    `json:"admin_transition_time"`
	TransitionRemember    bool   `json:"transition_remember_choice"`
	TransitionShowDesc    bool   `json:"transition_show_description"`
	TransitionTheme       string `json:"transition_theme,omitempty"`
	TransitionColor       string `json:"transition_color,omitempty"`
	TransitionAd1         string `json:"transition_ad1,omitempty"`
	TransitionAd2         string `json:"transition_ad2,omitempty"`
	AnnouncementOn        bool   `json:"announcement_enabled"`
	AnnouncementTitle     string `json:"announcement_title,omitempty"`
	AnnouncementContent   string `json:"announcement_content,omitempty"`
	AnnouncementStart     string `json:"announcement_start,omitempty"`
	AnnouncementEnd       string `json:"announcement_end,omitempty"`
	AnnouncementRemember  int    `json:"announcement_remember_days"`
}

type HomeData struct {
	Categories []Category     `json:"categories"`
	Featured   []Website      `json:"featured"`
	Settings   PublicSettings `json:"settings"`
	User       map[string]any `json:"user"`
}
