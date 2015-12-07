package gobot_test

import (
	"flag"
	"fmt"
	. "github.com/adufrene/karmabot/Godeps/_workspace/src/github.com/adufrene/gobot"
	"github.com/adufrene/karmabot/Godeps/_workspace/src/gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"testing"
)

var slackApi SlackApi

func TestMain(m *testing.M) {
	flag.Parse()
	file, err := ioutil.ReadFile("configuration.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read configuration file configuration.yaml: %s\n", err.Error())
		os.Exit(1)
	}
	var conf Configuration
	if err = yaml.Unmarshal(file, &conf); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse configuration file as yaml: %s", err.Error())
	}

	slackApi = NewSlackApi(conf.ApiToken)

	os.Exit(m.Run())
}

func TestTestAuth(t *testing.T) {
	if err := slackApi.TestAuth(); err != nil {
		t.Errorf("Failed testing auth: %v", err.Error())
	}
}
