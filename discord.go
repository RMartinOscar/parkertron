package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	stopDiscord = make(map[string]chan string)

	discordGlobal discord

	discordLoad = make(chan string)
)

// This function will be called (due to AddHandler) when the bot receives
// the "ready" event from Discord.
func readyDiscord(dg *discordgo.Session, event *discordgo.Ready, game string) {
	// if there is an error setting the game log and return
	if err := dg.UpdateStatus(0, game); err != nil {
		Log.Fatalf("error setting game: %s", err)
		return
	}

	Log.Debugf("set game to: %s", game)
}

// This function will be called (due to AddHandler) every time a new
// message is created on any channel that the autenticated bot has access to.
func discordMessageHandler(dg *discordgo.Session, m *discordgo.MessageCreate, serverConfig discordServer) {

}

// kick a user and log it to a channel if configured
func kickDiscordUser(dg *discordgo.Session, guild, user, username, reason, authorname string) (err error) {
	if err = dg.GuildMemberDeleteWithReason(guild, user, reason); err != nil {
		return
	}

	// embed := &discordgo.MessageEmbed{
	// 	Title: "User has been kicked",
	// 	Color: 0xf39c12,
	// 	Fields: []*discordgo.MessageEmbedField{
	// 		&discordgo.MessageEmbedField{
	// 			Name:   "User",
	// 			Value:  username,
	// 			Inline: true,
	// 		},
	// 		&discordgo.MessageEmbedField{
	// 			Name:   "By",
	// 			Value:  authorname,
	// 			Inline: true,
	// 		},
	// 		&discordgo.MessageEmbedField{
	// 			Name:   "Reason",
	// 			Value:  reason,
	// 			Inline: true,
	// 		},
	// 	},
	// }

	// TODO: Need to use new config for this
	// sendDiscordEmbed(getDiscordConfigString("embed.audit"), embed)

	Log.Info("User " + authorname + " has been kicked from " + guild + " for " + reason)

	return
}

// ban a user and log it to a channel if configured
func banDiscordUser(dg *discordgo.Session, guild, user, username, reason, authorname string, days int) (err error) {
	if err = dg.GuildBanCreateWithReason(guild, user, reason, days); err != nil {
		return
	}

	// embed := &discordgo.MessageEmbed{
	// 	Title: "User has been banned for " + strconv.Itoa(days) + " days",
	// 	Color: 0xc0392b,
	// 	Fields: []*discordgo.MessageEmbedField{
	// 		&discordgo.MessageEmbedField{
	// 			Name:   "User",
	// 			Value:  username,
	// 			Inline: true,
	// 		},
	// 		&discordgo.MessageEmbedField{
	// 			Name:   "By",
	// 			Value:  authorname,
	// 			Inline: true,
	// 		},
	// 		&discordgo.MessageEmbedField{
	// 			Name:   "Reason",
	// 			Value:  reason,
	// 			Inline: true,
	// 		},
	// 	},
	// }

	// TODO: Need to use new config for embed audit to log to a webhook
	//	sendDiscordEmbed(getDiscordConfigString("embed.audit"), embed)

	Log.Info("User " + authorname + " has been kicked from " + guild + " for " + reason)

	return
}

// clean up messages if configured to
func deleteDiscordMessage(dg *discordgo.Session, channelID, messageID, message string) (err error) {
	Log.Debugf("Removing message \n'%s'\n from %s", message, channelID)

	if err = dg.ChannelMessageDelete(channelID, messageID); err != nil {
		return
	}

	// embed := &discordgo.MessageEmbed{
	// 	Title: "Message was deleted",
	// 	Color: 0xf39c12,
	// 	Fields: []*discordgo.MessageEmbedField{
	// 		&discordgo.MessageEmbedField{
	// 			Name:   "MessageID",
	// 			Value:  messageID,
	// 			Inline: true,
	// 		},
	// 		&discordgo.MessageEmbedField{
	// 			Name:   "Message Content",
	// 			Value:  message,
	// 			Inline: true,
	// 		},
	// 	},
	// }

	// TODO: Need to use new config for embed audit to log to a webhook
	// 	sendDiscordEmbed(getDiscordConfigString("embed.audit"), embed)

	Log.Debug("message was deleted.")

	return
}

// send message handling
func sendDiscordMessage(dg *discordgo.Session, channelID, authorID, prefix string, responseArray []string) (err error) {
	response := strings.Join(responseArray, "\n")
	response = strings.Replace(response, "&user&", authorID, -1)
	response = strings.Replace(response, "&prefix&", prefix, -1)
	response = strings.Replace(response, "&react&", "", -1)

	// if there is an error return the error
	if _, err = dg.ChannelMessageSend(channelID, response); err != nil {
		return
	}

	return
}

// send a reaction to a message
func sendDiscordReaction(dg *discordgo.Session, channelID string, messageID string, reactionArray []string) (err error) {
	for _, reaction := range reactionArray {
		Log.Debugf("sending \"%s\" as a reaction to message: %s", reaction, messageID)
		// if there is an error sending a message return it
		if err = dg.MessageReactionAdd(channelID, messageID, reaction); err != nil {
			return
		}
	}
	return
}

// send a message with an embed
func sendDiscordEmbed(dg *discordgo.Session, channelID string, embed *discordgo.MessageEmbed) error {
	// if there is an error sending the embed message
	if _, err := dg.ChannelMessageSendEmbed(channelID, embed); err != nil {
		Log.Fatal("Embed send error")
		return err
	}

	return nil
}

// service handling
// start all the bots
func startDiscordsBots() {
	Log.Infof("Starting IRC server connections\n")
	// range over the bots available to start
	for _, bot := range discordGlobal.Bots {
		Log.Infof("Connecting to %s\n", bot.BotName)

		// spin up a channel to tell the bot to shutdown later
		stopDiscord[bot.BotName] = make(chan string)

		// start the bot
		go startDiscordBotConnection(bot)
		// wait on bot being able to start.
		<-discordLoad
	}

	Log.Debug("Discord service started\n")
	servStart <- "discord_online"
}

// when a shutdown is sent close out services properly
func stopDiscordBots() {
	Log.Infof("stopping discord connections")
	// loop through bots and send shutdowns
	for _, bot := range discordGlobal.Bots {
		Log.Infof("stopping %s", bot.BotName)
		stopDiscord[bot.BotName] <- ""

		<-stopDiscord[bot.BotName]
		Log.Infof("stopped %s", bot.BotName)
	}
	Log.Infof("discord connections stopped")
	// return shutdown signal on channel
	servStopped <- "discord_stopped"
}

// start connections to discord
func startDiscordBotConnection(discordConfig discordBot) {
	// Initializing Discord connection
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + discordConfig.Config.Token)
	if err != nil {
		Log.Errorf("error creating Discord session for %s: %v", discordConfig.BotName, err)
		return
	}

	// Register ready as a callback for the ready events
	// dg.AddHandler(readyDiscord)
	// Thank Stroom on the discordgopher discord for getting me this
	dg.AddHandler(func(dg *discordgo.Session, event *discordgo.Ready) {
		readyDiscord(dg, event, discordConfig.Config.Game)
	})

	// Register messageCreate as a callback for the messageCreate events.
	// dg.AddHandler(discordMessageHandler)
	// Thank Stroom on the discordgopher discord for getting me this
	for _, server := range discordConfig.Servers {
		dg.AddHandler(func(dg *discordgo.Session, event *discordgo.MessageCreate) {
			discordMessageHandler(dg, event, server)
		})
	}

	Log.Debug("Discord service connected\n")

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		Log.Fatal("error opening connection,", err)
		return
	}

	bot, err := dg.User("@me")
	if err != nil {
		fmt.Println("error obtaining account details,", err)
	}

	Log.Debug("Invite the bot to your server with https://discordapp.com/oauth2/authorize?client_id=" + bot.ID + "&scope=bot")

	discordLoad <- ""

	<-stopDiscord[discordConfig.BotName]

	// properly send a shutdown to the discord server so the bot goes offline.
	dg.Close()

	// return the shutdown signal
	stopDiscord[discordConfig.BotName] <- ""
}
