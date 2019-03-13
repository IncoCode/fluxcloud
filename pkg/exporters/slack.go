package exporters

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/justinbarrick/fluxcloud/pkg/config"
	"github.com/justinbarrick/fluxcloud/pkg/msg"

	"github.com/nlopes/slack"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// The Slack exporter sends Flux events to a Slack channel via a webhook.
type Slack struct {
	OAuthToken       string
	slackApi         *slack.Client
	defaultChannelID string
	kubernetesClient *kubernetes.Clientset
}

// Represents a slack message sent to the API
type SlackMessage struct {
	ChannelID   string
	Attachments []slack.Attachment
}

// Represents a section of a slack message that is sent to the API
type SlackAttachment struct { // ToDo: delete
	Color     string `json:"color"`
	Title     string `json:"title"`
	TitleLink string `json:"title_link"`
	Text      string `json:"text"`
}

// Represents a slack channel and the Kubernetes namespace linked to it
type SlackChannel struct { // ToDo: delete
	Channel   string `json:"channel"`
	Namespace string `json:"namespace"`
}

const slackChannelAnnotationName = "slackChannelId"

// Initialize a new Slack instance
func NewSlack(config config.Config) (*Slack, error) {
	var err error
	s := Slack{}

	s.OAuthToken, err = config.Required("slack_oauth_token")
	if err != nil {
		return nil, err
	}

	s.defaultChannelID, err = config.Required("slack_default_channel_id")
	if err != nil {
		return nil, err
	}

	s.slackApi = slack.New(s.OAuthToken)

	// creates the in-cluster config
	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// creates the clientset
	s.kubernetesClient, err = kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// Send a SlackMessage to Slack
func (s *Slack) Send(client *http.Client, message msg.Message) error {
	for _, slackMessage := range s.NewSlackMessage(message) {
		fmt.Println("Slack message", slackMessage)

		if _, _, err := s.slackApi.PostMessage(slackMessage.ChannelID, slack.MsgOptionAttachments(slackMessage.Attachments...)); err != nil {
			return fmt.Errorf("Unable to post message to slack, error = %v", err)
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

	for _, channelID := range s.determineChannels(message) {
		slackMessage := SlackMessage{
			ChannelID: channelID,
			Attachments: []slack.Attachment{
				slack.Attachment{
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

func (s *Slack) getChannelIDByService(namespace string, serviceName string) string {
	var pod *v1.Pod

	pods, err := s.kubernetesClient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		fmt.Println(fmt.Sprintf("Error: unable to get pods with in namespace = %v, serviceName = %v, error = %v", namespace, serviceName, err))
		return s.defaultChannelID
	}

	for i, p := range pods.Items {
		if strings.Contains(p.Name, serviceName) {
			pod = &pods.Items[i]
		}
	}

	if pod == nil {
		fmt.Println("Unable to find pod with service name = ", serviceName)
		return s.defaultChannelID
	}

	if pod.Annotations[slackChannelAnnotationName] != "" {
		return pod.Annotations[slackChannelAnnotationName]
	}

	return s.defaultChannelID
}

// Match namespaces from service IDs to Slack channels.
func (s *Slack) determineChannels(message msg.Message) []string {
	var channels []string
	for _, serviceID := range message.Event.ServiceIDs {
		ns, _, name := serviceID.Components()
		channels = appendIfMissing(channels, s.getChannelIDByService(ns, name))
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
