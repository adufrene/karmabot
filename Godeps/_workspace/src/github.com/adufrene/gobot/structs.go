package gobot

type SlackMessage struct {
	Type string `json:"type"`
}

type Hello struct {
	SlackMessage
}

type PresenceChange struct {
	SlackMessage
	User     string `json:"user"`
	Presence string `json:"presence"`
}

type UserTyping struct {
	SlackMessage
	Channel string `json:"channel"`
	User    string `json:"user"`
}

type Message struct {
	SlackMessage
	Channel   string `json:"channel"`
	User      string `json:"user"`
	Text      string `json:"text"`
	Timestamp string `json:"ts"`
	Team      string `json:"team"`
}

type SlackApi struct {
	token string
}

type SlackUser struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Deleted  bool   `json:"deleted"`
	Color    string `json:"color"`
	RealName string `json:"real_name"`
	TimeZone string `json:"tz"`
	TZLabel  string `json:"tz_label"`
	TZOffset int32  `json:"tz_offset"`

	Profile SlackProfile `json:"profile"`

	IsAdmin           bool `json:"is_admin"`
	IsOwner           bool `json:"is_owner"`
	IsPrimaryOwner    bool `json:"is_primary_owner"`
	IsRestricted      bool `json:"is_restricted"`
	IsUltraRestricted bool `json:"is_ultra_restricted"`
	IsBot             bool `json:"is_bot"`
	HasFiles          bool `json:"has_files"`

	// status
}

type SlackProfile struct {
	BotId              string `json:"bot_id"`
	RealName           string `json:"real_name"`
	RealNameNormalized string `json:"real_name_normalized"`
	Email              string `json:"email"`
	Image24            string `json:"image_24"`
	Image32            string `json:"image_32"`
	Image48            string `json:"image_48"`
	Image72            string `json:"image_72"`
	Image192           string `json:"image_192"`
	Image512           string `json:"image_512"`
	Image1024          string `json:"image_1024"`
	ImageOriginal      string `json:"image_original"`
}

type gobot struct {
	slackApi           SlackApi
	setupFunc          func(SlackApi)
	messageFunc        func(SlackApi, Message)
	allMessageFunc     func(SlackApi, Message)
	presenceChangeFunc func(SlackApi, PresenceChange)
	userTypingFunc     func(SlackApi, UserTyping)
}

type slackResponse struct {
	Okay bool `json:"ok"`
}

type slackAuthResponse struct {
	slackResponse
	Error string `json:"error"`
}

type slackStart struct {
	slackResponse
	URL string `json:"url"`
}

type slackUser struct {
	slackResponse
	User SlackUser `json:"user"`
}

type slackUsers struct {
	slackResponse
	Users []SlackUser `json:"members"`
}
