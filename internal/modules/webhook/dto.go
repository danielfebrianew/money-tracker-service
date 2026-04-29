package webhook

type FonntePayload struct {
	Device    string `json:"device"`
	Sender    string `json:"sender"`
	Message   string `json:"message"`
	Text      string `json:"text"`
	Member    string `json:"member"`
	Name      string `json:"name"`
	Location  string `json:"location"`
	URL       string `json:"url"`
	Filename  string `json:"filename"`
	Extension string `json:"extension"`
	PollName  string `json:"pollname"`
	Choices   string `json:"choices"`
	InboxID   string `json:"inboxid"`
	Timestamp string `json:"timestamp"`
	Token     string `json:"token"`
}
