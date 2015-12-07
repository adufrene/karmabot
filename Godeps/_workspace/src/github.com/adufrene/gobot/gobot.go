package gobot

import (
	"encoding/json"
	"fmt"
	"github.com/adufrene/karmabot/Godeps/_workspace/src/github.com/gorilla/websocket"
	"net/http"
)

// TODO: Create noop functions as default functions, so we don't need to check nil
func NewGobot(apiToken string) (gBot gobot) {
	gBot.slackApi = NewSlackApi(apiToken)
	return
}

func (g *gobot) RegisterSetupFunction(setupFunction func(SlackApi)) {
	oldSetup := g.setupFunc
	g.setupFunc = func(slackApi SlackApi) {
		setupFunction(slackApi)
		if oldSetup != nil {
			oldSetup(slackApi)
		}
	}
}

func (g *gobot) RegisterMessageFunction(messageFunc func(SlackApi, Message)) {
	g.messageFunc = messageFunc
}

func (g *gobot) RegisterAllMessageFunction(messageFunc func(SlackApi, Message)) {
	g.allMessageFunc = messageFunc
}

func (g *gobot) RegisterPresenceChangeFunction(presenceChangeFunc func(SlackApi, PresenceChange)) {
	g.presenceChangeFunc = presenceChangeFunc
}

func (g *gobot) RegisterUserTypingFunc(userTypingFunc func(SlackApi, UserTyping)) {
	g.userTypingFunc = userTypingFunc
}

func (g *gobot) Listen() (err error) {
	start, err := g.slackApi.startRTM()
	if err != nil {
		return
	}
	if !start.Okay {
		return fmt.Errorf("Real-Time Messaging failed to start, aborting")
	}

	if g.setupFunc != nil {
		g.setupFunc(g.slackApi)
	}

	conn := start.openWebSocket()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var msgType SlackMessage
		if err = json.Unmarshal(msg, &msgType); err != nil {
			return err
		}
		g.delegate(msgType.Type, msg)
	}
}

func (start slackStart) openWebSocket() *websocket.Conn {
	var emptyHeader http.Header
	var defaultDialer *websocket.Dialer
	conn, _, err := defaultDialer.Dial(start.URL, emptyHeader)
	if err != nil {
		panic(err)
	}

	return conn
}

func (g *gobot) delegate(msgType string, msg []byte) {
	switch msgType {
	case "hello":
		fmt.Println("Hello from Slack!")
	case "presence_change":
		if g.presenceChangeFunc != nil {
			var presenceChange PresenceChange
			json.Unmarshal(msg, &presenceChange)
			g.presenceChangeFunc(g.slackApi, presenceChange)
		}
	case "message":
		if g.messageFunc != nil || g.allMessageFunc != nil {
			var message Message
			json.Unmarshal(msg, &message)
			if g.allMessageFunc != nil {
				g.allMessageFunc(g.slackApi, message)
			}
			if message.User != "" { // Slack uses empty user to indicate I sent message
				g.messageFunc(g.slackApi, message)
			}
		}
	case "user_typing":
		if g.userTypingFunc != nil {
			var userTyping UserTyping
			json.Unmarshal(msg, &userTyping)
			g.userTypingFunc(g.slackApi, userTyping)
		}
	default:
		fmt.Println(string(msg))
	}
}
