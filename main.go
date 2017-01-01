package main

import (
	"flag"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/b4b4r07/retest-bot/travis"
	"github.com/nlopes/slack"
)

var (
	repo = flag.String("repo", "", "Specify github.com repository name")
	user = flag.String("user", "", "Specify github.com user name")
)

var pattern *regexp.Regexp = regexp.MustCompile(`New comment by .*/pulls?/(\d+)`)

func main() {
	flag.Parse()
	api := slack.New(os.Getenv("SLACK_TOKEN"))
	os.Exit(run(api))
}

func run(api *slack.Client) int {
	if *user == "" || *repo == "" {
		log.Print("user/repo: invalid format")
		return 1
	}

	connected := travis.AuthenticateWithTravis(os.Getenv("TRAVIS_CI_TOKEN"))
	if !connected {
		log.Print("can't connect travis api")
		return 1
	}

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
				log.Print("Connected!")

			case *slack.MessageEvent:
				if ev.SubType != "bot_message" {
					break
				}
				for _, attachment := range ev.Attachments {
					if !strings.Contains(attachment.Text, "retest please") {
						continue
					}
					pat := pattern.FindStringSubmatch(attachment.Pretext)
					if len(pat) < 2 {
						break
					}
					n, _ := strconv.Atoi(pat[1])
					err := travis.RestartBuildFromPR(*user+"/"+*repo, n)
					if err != nil {
						log.Print(err)
						return 1
					}
				}

			case *slack.InvalidAuthEvent:
				log.Print("Invalid credentials")
				return 1
			}
		}
	}
}
