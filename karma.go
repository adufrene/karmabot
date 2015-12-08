package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/adufrene/gobot"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

type Configuration struct {
	Token string `yaml:"apiToken"`
}

var userIdRegex *regexp.Regexp
var requestKarmaRegex *regexp.Regexp
var assignKarmaRegex *regexp.Regexp

var myUserId string
var myUserName string

var karmaCount map[string]int
var karmaFile string

func main() {
	userIdRegex = regexp.MustCompile(`^<@U[0-9A-Z]{8}>$`)
	assignKarmaRegex = regexp.MustCompile(`\b(\w+)((\+\+|--)+)`)

	apiToken, err := loadApiToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not find api token: %s\n", err.Error())
		os.Exit(1)
	}

	initKarmaFile()
	go loadKarmaCount()
	gobot := gobot.NewGobot(apiToken)
	gobot.RegisterMessageFunction(delegateFunction)
	gobot.RegisterSetupFunction(setup)
	if err := gobot.Listen(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while listening: %s\n", err.Error())
		os.Exit(1)
	}
}

func initKarmaFile() {
	if len(os.Args) > 1 && os.Args[1] != "" {
		karmaFile = os.Args[1]
	} else {
		karmaFile = "karma.csv"
	}
}

func loadApiToken() (string, error) {
	token := os.Getenv("KARMABOT_API")
	if token != "" {
		return token, nil
	}

	filename := "configuration.yaml"
	if len(os.Args) > 2 && os.Args[2] != "" {
		filename = os.Args[2]
	}

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	var conf Configuration
	if err = yaml.Unmarshal(file, &conf); err != nil {
		return "", err
	}
	if len(conf.Token) == 0 {
		return "", fmt.Errorf("Empty token in configuration file")
	}
	return conf.Token, nil
}

func setup(slackApi gobot.SlackApi) {
	user, err := slackApi.Whoami()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching info about bot")
		os.Exit(1)
	}

	myUserId = user.Id
	myUserName = user.Name
	requestKarmaRegex = regexp.MustCompile(fmt.Sprintf("(<@%s>|%s).*(karma|help)",
		strings.ToLower(myUserId), strings.ToLower(myUserName)))
}

func delegateFunction(slackApi gobot.SlackApi, message gobot.Message) {
	text := strings.ToLower(message.Text)
	if requestKarmaRegex.MatchString(text) {
		request := requestKarmaRegex.FindStringSubmatch(text)[2]
		if "karma" == request {
			displayKarma(slackApi, message.Channel)
		} else if "help" == request {
			displayHelp(slackApi, message.Channel)
		} else {
			fmt.Fprintln(os.Stderr, "Shouldn't get to this branch...")
		}
	} else {
		tryUpdateKarma(slackApi, message)
	}
}

func displayKarma(slackApi gobot.SlackApi, channel string) {
	if len(karmaCount) == 0 {
		slackApi.PostMessage(channel, "No karma yet")
		return
	}
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

func displayHelp(slackApi gobot.SlackApi, channel string) {
	message := `
 		karmabot karma - Display current karma
 		user[++|--] - give or remove karma from user
 		karmabot help - Ask karmabot for help`

	slackApi.PostMessage(channel, fmt.Sprintf("```%s```", message))
}

func tryUpdateKarma(slackApi gobot.SlackApi, message gobot.Message) {
	var users []gobot.SlackUser
	var err error
	scanner := bufio.NewScanner(strings.NewReader(message.Text))
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		word := scanner.Text()

		if assignKarmaRegex.MatchString(word) {
			matches := assignKarmaRegex.FindStringSubmatch(word)
			// Lazy load users
			if users == nil {
				users, err = slackApi.GetUsersInTeam()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error fetching users: %s\n", err.Error())
					return
				}
			}
			resolveUserAndTryKarma(slackApi, users, matches[1], parseActions(matches[2]), message)
		}
	}
}

func parseActions(allActions string) []string {
	result := make([]string, len(allActions)/2)

	for i := 0; i < len(allActions); i += 2 {
		result[i/2] = allActions[i : i+2]
	}

	return result
}

func resolveUserAndTryKarma(slackApi gobot.SlackApi, users []gobot.SlackUser, user string, actions []string, message gobot.Message) {
	userId, err := resolveUser(users, user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving user: %s\n", err.Error())
		slackApi.PostMessage(message.Channel, fmt.Sprintf("Who the heck is %s?\n", user))
		return
	}
	if userId != message.User {
		if response, shouldPost := doKarma(userId, actions, message.User); shouldPost {
			slackApi.PostMessage(message.Channel, response)
		}
	} else {
		slackApi.PostMessage(message.Channel, fmt.Sprintf("Nice try <@%s>!", userId))
	}
}

func doKarma(user string, actions []string, from string) (string, bool) {
	go writeKarmaCount(user, actions)
	total := 0
	for _, action := range actions {
		if action == "++" {
			total++
		} else if action == "--" {
			total--
		}
	}
	karmaCount[user] += total

	if user == myUserId {
		if total > 0 {
			return fmt.Sprintf("Thank you <@%s>!", from), true
		} else if total < 0 {
			return ":angry:", true
		}
	}
	return "", false
}

func loadKarmaCount() {
	karmaCount = make(map[string]int)

	if _, err := os.Stat(karmaFile); os.IsNotExist(err) {
		return
	}

	file, err := os.OpenFile(karmaFile, os.O_RDONLY, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open file %s\n", karmaFile)
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

func writeKarmaCount(user string, actions []string) {
	file, err := os.OpenFile(karmaFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open/create file %s\n", karmaFile)
		os.Exit(1)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	for _, action := range actions {
		writer.Write([]string{user, action})
	}
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
