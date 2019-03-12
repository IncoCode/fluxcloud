package exporters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/justinbarrick/fluxcloud/pkg/config"
	"github.com/justinbarrick/fluxcloud/pkg/msg"

	"github.com/nlopes/slack"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// The Slack exporter sends Flux events to a Slack channel via a webhook.
type Slack struct {
	Url        string // ToDo: delete
	Username   string
	Channels   []SlackChannel
	IconEmoji  string
	OAuthToken string
	slackApi   *slack.Client
}

// Represents a slack message sent to the API
type SlackMessage struct {
	Channel     string            `json:"channel"`
	IconEmoji   string            `json:"icon_emoji"`
	Username    string            `json:"username"`
	Attachments []SlackAttachment `json:"attachments"`
}

// Represents a section of a slack message that is sent to the API
type SlackAttachment struct {
	Color     string `json:"color"`
	Title     string `json:"title"`
	TitleLink string `json:"title_link"`
	Text      string `json:"text"`
}

// Represents a slack channel and the Kubernetes namespace linked to it
type SlackChannel struct {
	Channel   string `json:"channel"`
	Namespace string `json:"namespace"`
}

// Initialize a new Slack instance
func NewSlack(config config.Config) (*Slack, error) {
	var err error
	s := Slack{}

	s.Url, err = config.Required("slack_url")
	if err != nil {
		return nil, err
	}

	s.OAuthToken, err = config.Required("slack_oauth_token")
	if err != nil {
		return nil, err
	}

	channels, err := config.Required("slack_channel")
	if err != nil {
		return nil, err
	}
	s.parseSlackChannelConfig(channels)
	log.Println(s.Channels)

	s.Username = config.Optional("slack_username", "Flux Deployer")
	s.IconEmoji = config.Optional("slack_icon_emoji", ":star-struck:")

	return &s, nil
}

// Send a SlackMessage to Slack
func (s *Slack) Send(client *http.Client, message msg.Message) error {
	for _, slackMessage := range s.NewSlackMessage(message) {
		fmt.Println(slackMessage)
		b := new(bytes.Buffer)
		err := json.NewEncoder(b).Encode(slackMessage)
		if err != nil {
			log.Print("Could encode message to slack:", err)
			return err
		}

		log.Print(string(b.Bytes()))
		res, err := client.Post(s.Url, "application/json", b)
		if err != nil {
			log.Print("Could not post to slack:", err)
			return err
		}

		if res.StatusCode != 200 {
			log.Print("Could not post to slack, status: ", res.Status)
			return fmt.Errorf("Could not post to slack, status: %d", res.StatusCode)
		}
	}

	return nil
}

// Return the new line character for Slack messages
func (s *Slack) NewLine() string {
	return "\n"
}

// Return a formatted link for Slack.
func (s *Slack) FormatLink(link string, name string) string {
	return fmt.Sprintf("<%s|%s>", link, name)
}

// Convert a flux event into a Slack message(s)
func (s *Slack) NewSlackMessage(message msg.Message) []SlackMessage {
	var messages []SlackMessage

	for _, channel := range s.determineChannels(message) {
		slackMessage := SlackMessage{
			Channel:   channel,
			IconEmoji: s.IconEmoji,
			Username:  s.Username,
			Attachments: []SlackAttachment{
				SlackAttachment{
					Color:     "#4286f4",
					TitleLink: message.TitleLink,
					Title:     message.Title,
					Text:      message.Body,
				},
			},
		}
		messages = append(messages, slackMessage)
	}

	return messages
}

// Return the name of the exporter.
func (s *Slack) Name() string {
	return "Slack"
}

// Parse the channel configuration string in a backwards
// compatible manner.
func (s *Slack) parseSlackChannelConfig(channels string) error {
	if len(strings.Split(channels, "=")) == 1 {
		s.Channels = append(s.Channels, SlackChannel{channels, "*"})
		return nil
	}

	re := regexp.MustCompile("([#a-z0-9][a-z0-9._-]*)=([a-z0-9*][-A-Za-z0-9_.]*)")
	for _, kv := range strings.Split(channels, ",") {
		if !re.MatchString(kv) {
			return fmt.Errorf("Could not parse channel/namespace configuration: %s", kv)
		}

		cn := strings.Split(kv, "=")
		channel := strings.TrimSpace(cn[0])
		namespace := strings.TrimSpace(cn[1])
		s.Channels = append(s.Channels, SlackChannel{channel, namespace})
	}

	return nil
}

// Match namespaces from service IDs to Slack channels.
func (s *Slack) determineChannels(message msg.Message) []string {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	pods, err := clientset.CoreV1().Pods("default").List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	for i, pod := range pods.Items {
		log.Println(fmt.Sprintf("Pod #%v = %v, Annotations = %v", i, pod.Name, pod.Annotations))
	}

	var channels []string
	for _, serviceID := range message.Event.ServiceIDs {
		ns, _, name := serviceID.Components()
		log.Println("service name =" + name)

		for _, ch := range s.Channels {
			if ch.Namespace == "*" || ch.Namespace == ns {
				channels = appendIfMissing(channels, ch.Channel)
			}
		}
	}
	return channels
}

func appendIfMissing(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}
