package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	//BotSession is the DiscordSession
	dg *discordgo.Session
)

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	//variables to get used.
	var attached []string
	var authed bool

	// Ignore all messages created by bots
	if m.Author.Bot {
		debug("User is a bot and being ignored.")
		return
	}

	// get channel information
	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		fatal("Channel error", err)
		return
	}

	// Respond on DM's
	// TODO: Make the response customizable
	if channel.Type == 1 {
		debug("This was a DM")
		sendDiscordMessage(channel.ID, getDiscordConfigString("direct.response"))
		return
	}

	// get guild info
	guild, err := s.Guild(channel.GuildID)
	if err != nil {
		fatal("Guild error", err)
		return
	}

	bot, err := dg.User("@me")
	if err != nil {
		fmt.Println("error obtaining account details,", err)
	}

	// quick referrence for information
	message := m.Content
	messageID := m.ID
	author := m.Author.ID
	authorname := m.Author.Username
	botID := bot.ID
	channelID := channel.ID
	attachments := m.Attachments

	// get group status. If perms are set and group name. These are note weighted yet.
	perms, group := discordPermissioncheck(author)

	// setting server owner default to admin perms
	if author == guild.OwnerID {
		perms = true
		group = "admin"
		authed = true
	}

	// debug messaging
	if perms {
		debug("author has perms and is in the group: " + group)
		if group == "admin" || group == "mod" {
			authed = true
		}
	}

	// something something kicked on mentioning a group.
	if getDiscordKOMChannel(channelID) {
		if author != guild.OwnerID || !perms {
			debug("Message is not being parsed but listened to.")
			// Check if a group is mentioned in message
			for _, ment := range m.MentionRoles {
				debug("Group " + ment + " was Mentioned")
				if strings.Contains(getDiscordKOMID(channelID+".group"), ment) {
					debug("Sending message to channel")
					sendDiscordMessage(channelID, getDiscordKOMMessage(channelID))
					debug("Sending message to user")
					sendDiscordDirectMessage(author, getDiscordKOMID(channelID+".reason"))
					kickDiscordUser(guild.ID, author, authorname, getDiscordKOMID(channelID+".reason"), botID)
				}
			}
		}
		return
	}

	// ignore blacklisted users
	if strings.Contains(getDiscordBlacklist(), author) == true {
		debug("User is blacklisted and being ignored.")
	}

	// making a string array for attached images on messages.
	for _, y := range attachments {
		debug(y.ProxyURL)
		attached = append(attached, y.ProxyURL)
		discordAttachmentHandler(attached, channelID)
	}

	// Always parse owner and group commands. Keyswords in matched channels.
	if !authed {
		// Ignore all messages created by blacklisted members, channels it's not listening on, with debug messaging.
		if !discordChannelFilter(channelID) {
			debug("This channel is being filtered out and ignored.")
			for _, ment := range m.Mentions {
				if ment.ID == dg.State.User.ID {
					debug("The bot was mentioned")
					sendDiscordMessage(channelID, getDiscordConfigString("mention.wrong_channel"))
				}
			}
			debug("Message has been ignored.")
			return
		}
	}

	// Check if the bot is mentioned
	for _, ment := range m.Mentions {
		if ment.ID == dg.State.User.ID {
			debug("The bot was mentioned")
			sendDiscordMessage(channelID, getDiscordConfigString("mention.response"))
			if strings.Replace(message, "<@"+dg.State.User.ID+">", "", -1) == "" {
				sendDiscordMessage(channelID, getDiscordConfigString("mention.empty"))
			}
		}
	}

	//
	// Message Handling
	//
	if message != "" {
		debug("Message Content: " + message)
		discordMessageHandler(message, channelID, messageID, author, perms, group)
		return
	}
}

func sendDiscordMessage(ChannelID string, response string) {
	response = strings.Replace(response, "&prefix&", getDiscordConfigString("prefix"), -1)

	if strings.Contains(response, "&react&") {
		response = strings.Replace(response, "&react&", "", -1)
		//discordReaction()
	}

	superdebug("ChannelID " + ChannelID + " \n Discord Message Sent: \n" + response)
	dg.ChannelMessageSend(ChannelID, response)
}

func deleteDiscordMessage(ChannelID string, MessageID string) {
	dg.ChannelMessageDelete(ChannelID, MessageID)

	embed := &discordgo.MessageEmbed{
		Title: "Message was deleted",
		Color: 0xf39c12,
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "MessageID",
				Value:  MessageID,
				Inline: true,
			},
		},
	}

	sendDiscordEmbed(getDiscordConfigString("embed.audit"), embed)
	superdebug("message was deleted.")
}

func sendDiscordReaction(channelID string, messageID string, emojiID string, userID string, job string) {
	if job == "add" {
		dg.MessageReactionAdd(channelID, messageID, emojiID)
	}
	if job == "remove" {
		dg.MessageReactionRemove(channelID, messageID, emojiID, userID)
	}
}

func sendDiscordDirectMessage(userID string, response string) {
	channel, err := dg.UserChannelCreate(userID)
	if err != nil {
		fatal("error creating direct message channel,", err)
		return
	}
	sendDiscordMessage(channel.ID, response)
}

func kickDiscordUser(guild string, user string, username string, reason string, authorname string) {
	dg.GuildMemberDeleteWithReason(guild, user, reason)

	embed := &discordgo.MessageEmbed{
		Title: "User has been kicked",
		Color: 0xf39c12,
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "User",
				Value:  username,
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "By",
				Value:  authorname,
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Reason",
				Value:  reason,
				Inline: true,
			},
		},
	}

	sendDiscordEmbed(getDiscordConfigString("embed.audit"), embed)
	info("User " + authorname + " has been kicked from " + guild + " for " + reason)
}

func banDiscordUser(guild string, user string, username string, reason string, days int, authorname string) {
	dg.GuildBanCreateWithReason(guild, user, reason, days)

	embed := &discordgo.MessageEmbed{
		Title: "User has been banned for " + strconv.Itoa(days) + " days",
		Color: 0xc0392b,
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "User",
				Value:  username,
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "By",
				Value:  authorname,
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Reason",
				Value:  reason,
				Inline: true,
			},
		},
	}

	sendDiscordEmbed(getDiscordConfigString("embed.audit"), embed)
	info("User " + authorname + " has been kicked from " + guild + " for " + reason)
}

func sendDiscordEmbed(channelID string, embed *discordgo.MessageEmbed) {
	_, err := dg.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		fatal("Embed send error", err)
		return
	}
}

func startDiscordConnection() {
	//Initializing Discord connection
	// Create a new Discord session using the provided bot token.
	dg, err = discordgo.New("Bot " + getDiscordConfigString("token"))

	if err != nil {
		fatal("error creating Discord session,", err)
		return
	}

	// Register messageCreate as a callback for the messageCreate events.
	dg.AddHandler(messageCreate)

	debug("Discord service connected\n")

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		fatal("error opening connection,", err)
		return
	}
	debug("Discord service started\n")

	ServStat <- "discord_online"
}
