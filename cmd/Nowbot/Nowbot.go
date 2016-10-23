package main

import (
	//"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
	
	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
)

var (
	// discordgo session
	discord *discordgo.Session

	// Redis client connection (used for stats)
	//rcli *redis.Client

	// Map of Guild id's to *Play channels, used for queuing and rate-limiting guilds
	//queues map[string]chan *Play = make(map[string]chan *Play)

	// Sound encoding settings
	BITRATE        = 128
	MAX_QUEUE_SIZE = 6

	// Owner
	OWNER string
	NOWBOT_ID string
)

func scontains(key string, options ...string) bool {
	for _, item := range options {
		if item == key {
			return true
		}
	}
	return false
}

func utilGetMentioned(s *discordgo.Session, m *discordgo.MessageCreate) *discordgo.User {
	for _, mention := range m.Mentions {
		if mention.ID != s.State.Ready.User.ID {
			return mention
		}
	}
	return nil
}

func loreQuery(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild, msg string) {
	log.Info("Debug: loreQuery start")
	pwd, _ := os.Getwd()
	dir := pwd + "/lores"
	files, _ := ioutil.ReadDir(dir)
	fmt.Println("Directory: ", dir, "\n")       
	for _, f := range files {
		fmt.Println(f.Name())
	}
		
	/*
	var %query $2-
	var %dir $mircdir $+ Item_Lores\
	var %string * $+ %query $+ * $+ .txt
	/set %l.count 0
	/set %queryer $nick
	var %ffile = $findfile(%dir,%string,0, /l.query $1- )
	//msg $nick ||=====================
	if ( %ffile != 0 ) {
		//echo Hits: %ffile
	}
	else {
		//msg $nick || No hits for %query
	}
		
	var %file $1-
   	var %overmax 9
	var %maxcount 8
	/set %l.count 1 + %l.count   

	if ( %l.count == 1 ) {
		//echo 1 || Possible hits:
		//msg $nick || Possible hits:
	}
	if ( %l.count <= %maxcount ) {
		//set $+(%,[,%l.count,],[,$site,],Item,) $nopath(%file)
		//echo 1 || %l.count :: $nopath( %file )
	//msg $nick || %l.count :: $nopath( %file )
	}
	elseif ( %l.count == %overmax ) {
		/echo 1 Too many results!
		//msg $nick || Error: Too many results, refine your search.
	}*/
}

// Handles bot operator messages
func handleBotControlMessages(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild, msg string) {
	if scontains(parts[0], "!nowbot") {
		log.Info("Debug: !nowbot trying to output")
		s.ChannelMessageSend(m.ChannelID, "Owner !nowbot, with message " + msg)
		log.Info("Debug: !nowbot done trying to output")
 	}
	log.Info("Debug: handleBotControlMessages finished")
}

// Handles user messages
func handleUserCommandMessages(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild, msg string) {
	if scontains(parts[0], "!lore") {
		log.Info("Debug: !lore trying to output")
		s.ChannelMessageSend(m.ChannelID, "!lore with message: " + msg)
		loreQuery(s, m, parts, g, msg)
	}
	log.Info("Debug: handleUserCommandMessages finished")
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Info("Recieved READY payload")
	//Idletime and Game
	s.UpdateStatus(0, "ArcticMUD")
}

/*func onGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable != nil {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			s.ChannelMessageSend(channel.ID, "**Nowbot ready.**")
			return
		}
	}
}*/

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	
	// print everything to terminal for debugging
	fmt.Printf("%20s %20s %20s > %s\n", m.ChannelID, time.Now().Format(time.Stamp), m.Author.Username, m.Content)
	
	// exit if it's Nowbot or another bot talking
	if (m.Author.ID == NOWBOT_ID || m.Author.Bot) {
		return
	}
	
	// exit if message is nil, or if does not contain command character @ mention
	if len(m.Content) <= 0 || (m.Content[0] != '!' && len(m.Mentions) < 1) {
		return
	}

	msg := strings.Replace(m.ContentWithMentionsReplaced(), s.State.Ready.User.Username, "username", 1)
	parts := strings.Split(strings.ToLower(msg), " ")
	channel, _ := discord.State.Channel(m.ChannelID)
	if channel == nil {
		log.WithFields(log.Fields{
			"channel": m.ChannelID,
			"message": m.ID,
		}).Warning("Failed to grab channel")
		return
	}

	guild, _ := discord.State.Guild(channel.GuildID)
	if guild == nil {
		log.WithFields(log.Fields{
			"guild":   channel.GuildID,
			"channel": channel,
			"message": m.ID,
		}).Warning("Failed to grab guild")
		return
	}

	// If this is a mention, it should come from the owner (otherwise we don't care)
	if len(m.Mentions) > 0 && m.Author.ID == OWNER && len(parts) > 0 {
		mentioned := false
		for _, mention := range m.Mentions {
			mentioned = (mention.ID == s.State.Ready.User.ID)
			if mentioned {
				break
			}
		}

		if mentioned {
			handleBotControlMessages(s, m, parts, guild, msg)
		}
		return
	}
	
	// do all other commands
	handleUserCommandMessages(s, m, parts, guild, msg)
	
	log.Info("Debug: onMessageCreate finished...")
}

func main() {
	var (
		Token      = flag.String("t", "", "Discord Authentication Token")
		Shard      = flag.String("s", "", "Shard ID")
		ShardCount = flag.String("c", "", "Number of shards")
		Owner      = flag.String("o", "", "Owner ID")
		err        error
	)
	flag.Parse()

	if *Owner != "" {
		OWNER = *Owner
		log.Info("Debug: Setting Owner...")
		log.Info("Debug: Owner is " + OWNER)
	}
	
	NOWBOT_ID = "239462226392514561"
	
	// Preload
	//log.Info("Preloading sounds...")
	//for _, coll := range COLLECTIONS {
	//	coll.Load()
	//}
	
	// Create a discord session
	log.Info("Starting discord session...")
	log.Info("Token is " + *Token)
	discord, err = discordgo.New(*Token)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to create discord session")
		return
	}

	// Set sharding info
	discord.ShardID, _ = strconv.Atoi(*Shard)
	discord.ShardCount, _ = strconv.Atoi(*ShardCount)

	if discord.ShardCount <= 0 {
		discord.ShardCount = 1
	}

	discord.AddHandler(onReady)
//	discord.AddHandler(onGuildCreate)
	discord.AddHandler(onMessageCreate)

	err = discord.Open()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to create discord websocket connection")
		return
	}
	
	// We're running!
	log.Info("Nowbot ready.")
	
	// Wait for a signal to quit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c
}
