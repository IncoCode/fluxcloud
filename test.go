package main

import (
	"fmt"
	"os"

	"github.com/nlopes/slack"
)

func main() {
	slackApi := slack.New("")
	channels, err := slackApi.GetChannels(false)
	if err != nil {
		fmt.Println("Error", err)
		os.Exit(1)
	} else {
		fmt.Println("Channels", channels)
	}

	attachment := slack.Attachment{
		TitleLink: "Title link",
		Color:     "#4286f4",
		Title:     "Title",
		Text:      "Text",
	}
	if s1, s2, err := slackApi.PostMessage("CGU5GP093", slack.MsgOptionAttachments(attachment)); err != nil {
		fmt.Println("Send Message error", err)
	} else {
		fmt.Println("s1", s1, "s2", s2)
	}
}
