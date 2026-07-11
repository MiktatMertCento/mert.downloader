package domain

const (
	cookieFile  = "cookies.txt"
	downloadDir = "downloads"
	browserUA   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	igAppID     = "936619743392459"
	alphabet    = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
)

type NetscapeCookie struct {
	Domain string
	Name   string
	Value  string
}

type ParsedURL struct {
	Shortcode   string
	IsReel      bool
	IsStory     bool
	IsProfile   bool
	IsHighlight bool
	Username    string
	StoryID     string
	HighlightID string
	Platform    string
	VideoID     string
}

type HighlightCover struct {
	Title string
	ID    string
	Item  MediaItem
}

type MediaItem struct {
	URL    string `json:"url"`
	Type   string `json:"type"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type MediaInfo struct {
	Shortcode string      `json:"shortcode"`
	MediaType string      `json:"media_type"`
	Username  string      `json:"username"`
	Caption   string      `json:"caption"`
	Items     []MediaItem `json:"items"`
}

type DownloadRequest struct {
	URL string `json:"url"`
}

type DownloadedFile struct {
	Filename string `json:"filename"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	Size     int64  `json:"size"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
}

type DownloadResponse struct {
	Success   bool             `json:"success"`
	Shortcode string           `json:"shortcode"`
	MediaType string           `json:"media_type"`
	Username  string           `json:"username"`
	Caption   string           `json:"caption,omitempty"`
	Files     []DownloadedFile `json:"files"`
	Error     string           `json:"error,omitempty"`
}
