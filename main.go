package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	data        map[string][]string
	current_sub map[string]string
)

type RedditResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func main() {

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}

	// Get the bot token from the environment variable
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		fmt.Println("Discord bot token not found in environment variable.")
		return
	}

	// Create a new Discord session with the bot token
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	data = make(map[string][]string)
	current_sub = make(map[string]string)

	// Register message create event handler
	dg.AddHandler(messageCreate)
	dg.AddHandler(interactionCreate)

	// Open a websocket connection to Discord and begin listening
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord connection: ", err)
		return
	}

	// Wait until the bot is stopped
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	<-make(chan struct{})
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages sent by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	if (len(m.Content) >= 5) && m.Content[:5] == ".show" {
		str := strings.Split(m.Content, " ")
		if len(str) != 2 {
			s.ChannelMessageSend(m.ChannelID, "Please provide the command and subreddit name.")
		} else {
			s.ChannelMessageSend(m.ChannelID, "Please wait while I fetch the data for you.")
			if isFetched, err := getData(m.ChannelID, str[1]); !isFetched {
				s.ChannelMessageSend(m.ChannelID, err)
			} else {
				if len(data[m.ChannelID]) > 0 {
					current_sub[m.ChannelID] = str[1]
					sendPost(s, m.ChannelID, m.Author.Username)
				} else {
					s.ChannelMessageSend(m.ChannelID, "Please run `.show` command to get new data.")
				}
			}
		}
	}
	if (len(m.Content) >= 5) && m.Content[:5] == ".help" {
		s.ChannelMessageSend(m.ChannelID, "Please provide the command and subreddit name to start like `.show awww`")
	}
}

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Ignore interactions sent by the bot itself
	if i.Member.User.ID == s.State.User.ID {
		return
	}
	// Check if the interaction is a button click
	if i.Type == discordgo.InteractionMessageComponent {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		})
		// Send the next message
		if current_sub[i.ChannelID] == "" {
			s.ChannelMessageSend(i.ChannelID, "Please choose a subreddit with the `.show` command")
		} else {
			sendPost(s, i.ChannelID, i.Member.User.Username)
		}
		// }
	}
}

func sendPost(s *discordgo.Session, channelID string, user string) {
  if len(data) <= 0 {
    s.ChannelMessageSend(channelID, "Empty list. Please choose a new subreddit.")
    return 
  }
	value := data[channelID][0]
	data[channelID] = data[channelID][1:]

	button1 := discordgo.Button{
		Label:    "More",
		Style:    discordgo.PrimaryButton,
		CustomID: "more",
	}

	// Create a message send struct
	embed := discordgo.MessageEmbed{
		Title: "r/" + current_sub[channelID],
		Color: 0xFF4500,
		Image: &discordgo.MessageEmbedImage{
			URL: value,
		},
    URL: "https://www.reddit.com/r/" + current_sub[channelID],
		Footer: &discordgo.MessageEmbedFooter{
			Text: user,
		},
	}

	messageSend := &discordgo.MessageSend{
		Embed: &embed,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{button1}},
		},
	}

	// Send the embedded message
	_, err := s.ChannelMessageSendComplex(channelID, messageSend)
	if err != nil {
		fmt.Println("Cannot send message:", err)
	}
}

func getData(channelID string, subreddit string) (bool, string) {
	urls := make([]string, 0)
	// Send an HTTP GET request to the Reddit API endpoint
	url := "https://www.reddit.com/r/" + subreddit + ".json"
	response, err := http.Get(url + "?limit=100")
	if err != nil {
		return false, "Cannot access json data. Please check if the subreddit exists. If it does, please run the same command a few times till you get the data."
	}
	defer response.Body.Close()

	// Parse the JSON response
	var redditResponse RedditResponse
	err = json.NewDecoder(response.Body).Decode(&redditResponse)
	if err != nil {
		return false, "Cannot parse json data"
	}

	// Iterate over the children and print the titles and URLs
	for _, child := range redditResponse.Data.Children {
		url := child.Data.URL
		links := []string{"i.redd.it", "catbox.moe"}
		for _, link := range links {
			if strings.Index(url, link) != -1 {
				urls = append(urls, url)
			}
		}
	}
	data[channelID] = urls
	return true, ""
}
