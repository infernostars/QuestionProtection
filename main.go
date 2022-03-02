package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Bot parameters
var (
	GuildID        = flag.String("guild", "", "Test guild ID. If not passed - bot registers commands globally")
	BotToken       = flag.String("token", "", "Bot access token")
	RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
	AppID          = flag.String("app", "", "Application ID")
	TestChannel    = flag.String("results", "", "Channel where send survey results to")
)

func init() {
	flag.Parse()
}

// its command initialization time!!!
var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "verify",
			Description: "Verify your account to get access to this server!",
		},
		//		{
		//			Name: "setrole",
		//			Description: "Change the role people get when verifying with QuestionProtection.",
		//		},
	}
	//ok, cool, but now we need to handle them
	commandsHandlers = map[string]func(disgo *discordgo.Session, i *discordgo.InteractionCreate){
		"verify": func(disgo *discordgo.Session, i *discordgo.InteractionCreate) {
			disgo.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &discordgo.InteractionResponseData{
					CustomID: "verify_user_" + i.Interaction.Member.User.ID,
					Title:    "Verify your account",
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "verifyquestion",
							Label:       "Example question?",
							Style:       discordgo.TextInputShort,
							Placeholder: "Answer to the question goes here.",
							Required:    true,
							MaxLength:   100,
							MinLength:   1,
						},
					},
				},
			})
		},
	}
	// ok fuck setrole we arent doing it rn jeezus thats a lot
)

func main() {
	disgo, err := discordgo.New("Bot " + *BotToken)

	if err != nil { // if we have an error:
		log.Fatalln("error: creating discord session failed.", err)
	}
	disgo.AddHandler(func(disgo *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type { //what kind of interaction?
		case discordgo.InteractionApplicationCommand:
			if h, ok := commandsHandlers[i.ApplicationCommandData().Name]; ok {
				h(disgo, i)
			} // command!!!
		case discordgo.InteractionModalSubmit:
			err := disgo.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Thank you! We are now verifying your account.",
					Flags:   1 << 6,
				},
			})
			if err != nil {
				panic(err)
			}
			data := i.ModalSubmitData()

			if !strings.HasPrefix(data.CustomID, "modals_survey") {
				return
			}

			userid := strings.Split(data.CustomID, "_")[2]
			_, err = disgo.ChannelMessageSend(*TestChannel, fmt.Sprintf(
				"Testing - <@%s>\n\n**Test:**:\n%s\n\n",
				userid,
				data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value,
			))
			if err != nil {
				panic(err)
			}
			cmdIDs := make(map[string]string, len(commands))

			for _, cmd := range commands {
				rcmd, err := disgo.ApplicationCommandCreate(*AppID, *GuildID, cmd)
				if err != nil {
					log.Fatalf("Cannot create slash command %q: %v", cmd.Name, err)
				}

				cmdIDs[rcmd.ID] = rcmd.Name

				err = disgo.Open()
				if err != nil {
					log.Fatalf("Cannot open the session: %v", err)
				}
				defer disgo.Close()

				stop := make(chan os.Signal, 1)
				signal.Notify(stop, os.Interrupt)
				<-stop
				log.Println("Gracefully shutting down.")

				if !*RemoveCommands {
					return
				}
			}
		}
	})
}
