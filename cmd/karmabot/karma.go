package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/adufrene/karmabot/Godeps/_workspace/src/github.com/adufrene/gobot"
	"os"
	"regexp"
	"strings"
)

const (
	API_TOKEN  = "xoxb-16117683524-AJdMORD8PMjPELCIsP3cuMOQ"
	KARMA_FILE = "karma.csv"
)

var userIdRegex *regexp.Regexp
var requestKarmaRegex *regexp.Regexp
var myUserId string
var myUserName string
var karmaCount map[string]int

func main() {
	userIdRegex = regexp.MustCompile(`^<@U[0-9A-Z]{8}>$`)

	go loadKarmaCount()
	gobot := gobot.NewGobot(API_TOKEN)
	gobot.RegisterMessageFunction(delegateFunction)
	gobot.RegisterSetupFunction(setup)
	if err := gobot.Listen(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while listening: %s\n", err.Error())
		os.Exit(1)
	}
}

func setup(slackApi gobot.SlackApi) {
	user, err := slackApi.Whoami()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching info about bot")
		os.Exit(1)
	}

	myUserId = user.Id
	myUserName = user.Name
	requestKarmaRegex = regexp.MustCompile(fmt.Sprintf("(<@%s>|%s).*karma", myUserId, myUserName))
}

func delegateFunction(slackApi gobot.SlackApi, message gobot.Message) {
	if requestKarmaRegex.MatchString(message.Text) {
		displayKarma(slackApi, message.Channel)
	} else {
		tryUpdateKarma(slackApi, message)
	}
}

func displayKarma(slackApi gobot.SlackApi, channel string) {
	users, err := slackApi.GetUsersInTeam()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching users\n")
		slackApi.PostMessage(channel, "Could not fetch users to resolve karma")
		return
	}

	userMap := make(map[string]string, len(users))
	for _, user := range users {
		userMap[user.Id] = user.Name
	}

	var message bytes.Buffer

	message.WriteString("Current Karma:```\n")
	for user, score := range karmaCount {
		message.WriteString(fmt.Sprintf("%s: %d\n", userMap[user], score))
	}
	message.WriteString("```")
	slackApi.PostMessage(channel, message.String())
}

func tryUpdateKarma(slackApi gobot.SlackApi, message gobot.Message) {
	var users []gobot.SlackUser
	var err error
	scanner := bufio.NewScanner(strings.NewReader(message.Text))
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		word := scanner.Text()
		if strings.HasSuffix(word, "++") || strings.HasSuffix(word, "--") {
			// Lazy load users
			if users == nil {
				users, err = slackApi.GetUsersInTeam()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error fetching users: %s\n", err.Error())
					return
				}
			}
			resolveUserAndTryKarma(slackApi, users, word, message)
		}
	}
}

func resolveUserAndTryKarma(slackApi gobot.SlackApi, users []gobot.SlackUser, karmaCommand string, message gobot.Message) {
	suffix := karmaCommand[len(karmaCommand)-2:]
	userName := karmaCommand[:len(karmaCommand)-2]
	userId, err := resolveUser(users, userName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving user: %s\n", err.Error())
		slackApi.PostMessage(message.Channel, fmt.Sprintf("Who the heck is %s?\n", userName))
		return
	}
	if userId != message.User {
		doKarma(userId, suffix)
		if userId == myUserId && suffix == "++" {
			slackApi.PostMessage(message.Channel, fmt.Sprintf("Thank you <@%s>!", message.User))
		} else if userId == myUserId && suffix == "--" {
			slackApi.PostMessage(message.Channel, ":angry:")
		}
	} else {
		slackApi.PostMessage(message.Channel, fmt.Sprintf("Nice try <@%s>!", userId))
	}
}

func doKarma(user, action string) {
	go writeKarmaCount(user, action)
	if action == "++" {
		karmaCount[user]++
	} else if action == "--" {
		karmaCount[user]--
	}
}

func loadKarmaCount() {
	karmaCount = make(map[string]int)

	if _, err := os.Stat(KARMA_FILE); os.IsNotExist(err) {
		return
	}

	file, err := os.OpenFile(KARMA_FILE, os.O_RDONLY, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open file %s\n", KARMA_FILE)
		os.Exit(1)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 2

	rawData, err := reader.ReadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from karma file: %s\n", err.Error())
		os.Exit(1)
	}

	for _, record := range rawData {
		action := record[1]
		increment := 0
		if action == "++" {
			increment = 1
		} else if action == "--" {
			increment = -1
		}

		karmaCount[record[0]] += increment
	}

}

func writeKarmaCount(user, action string) {
	file, err := os.OpenFile(KARMA_FILE, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open/create file %s\n", KARMA_FILE)
		os.Exit(1)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Write([]string{user, action})
	writer.Flush()
}

func resolveUser(users []gobot.SlackUser, username string) (string, error) {
	if userIdRegex.MatchString(username) {
		return username[2 : len(username)-1], nil
	}
	for _, user := range users {
		if user.Name == username {
			return user.Id, nil
		}
	}
	return "", fmt.Errorf("Username %s not found", username)
}
