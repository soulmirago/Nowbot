package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"

	//"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
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
	OWNER      string
	NOWBOT_ID  string
	GLOBALLIST []string

	// Global variables for Lorebot
	LOREADDFLAG          = false
	LOREADDUSER_ID       string
	LOREADDUSER_USERNAME string
	LOREADDSTARTTIME     = time.Now()
	LOREADDGLOBALLIST    []string
	LOREADDITEMNAME      string
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

func loreQuery(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild, msg string) []string {
	log.Info("Debug: loreQuery start")

	// combine string to get query (excluding the command word)
	query := strings.Join(parts[1:], " ")
	s.ChannelMessageSend(m.ChannelID, "Nowbot searching lores for "+m.Author.Username+" for item '"+query+"'")
	// hardcoded for now, change to init file
	dir := "//mnt//disks//nowbot-storage//lores"

	// create directory
	files, _ := os.ReadDir(dir)
	log.Info("Directory: " + dir)

	// iterate over all filenames in the directory
	lorecount := 0
	loremax := 10
	lorelist := []string{""}
	lines := []string{""}
	for _, file := range files {
		if file.Type().IsRegular() {
			matched, err := regexp.MatchString(query, strings.ToLower(file.Name()))
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Warning("Regexp error")
			}
			if matched {
				lorecount += 1
				lorelist = append(lorelist, strings.TrimSuffix(file.Name(), ".txt"))
				lines = append(lines, strconv.Itoa(lorecount)+" :: "+lorelist[lorecount])
				log.Info("File contains: " + query + " : " + file.Name())
			}
			if lorecount > loremax {
				lines = append(lines, "Too many results, ending search.")
				break
			}
		}
	}
	//output results using code-markdown format
	s.ChannelMessageSend(m.ChannelID, "```"+strings.Join(lines[0:], "\n")+"```")
	if lorecount == 0 {
		s.ChannelMessageSend(m.ChannelID, "No hits on "+query+".")
	} else {
		s.ChannelMessageSend(m.ChannelID, "Done searching. Enter '!lorestats [item number]' to get results for that item.")
	}

	return lorelist
}

func loreStats(s *discordgo.Session, m *discordgo.MessageCreate, g *discordgo.Guild, lorenumber int) {

	// Send acknowledgement
	log.Info("Debug: loreStats start")

	// hardcoded for now, change to init file
	dir := "//mnt//disks//nowbot-storage//lores//"
	log.Info("Lorestats directory pre-append: " + dir)
	path := dir + GLOBALLIST[lorenumber] + ".txt"
	log.Info("Lorestats directory post-append: " + path)

	file, err := os.Open(path)
	if err != nil {
		log.Info("Debug: loreStats file open problem")
		return
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	itemname := strings.TrimSuffix(GLOBALLIST[lorenumber], ".txt")
	output := "Lore #" + strconv.Itoa(lorenumber) + ":\n" + "```Name: " + itemname + "\n" + strings.Join(lines[0:], "\n") + "\n" + "```"
	s.ChannelMessageSend(m.ChannelID, output)

	return
}

// Starts the bot listening to add lores to the database
func loreAddStart(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild) {

	// Send acknowledgement
	log.Info("Debug: loreAddStart start")

	// check to see if a loreadd is already running

	if LOREADDFLAG {
		s.ChannelMessageSend(m.ChannelID, "Error on !loreadd, program already running for "+LOREADDUSER_USERNAME+", wait until !loreend.")
		return
	}

	if len(parts) < 2 {
		log.Info("Debug: User didn't enter an argument")
		s.ChannelMessageSend(m.ChannelID, "Error on !loreadd, you need to enter an item name.")
	} else {
		LOREADDFLAG = true
		LOREADDSTARTTIME = time.Now()
		LOREADDUSER_ID = m.Author.ID
		LOREADDUSER_USERNAME = m.Author.Username
		itemname := strings.Join(parts[1:], " ")
		LOREADDITEMNAME = itemname
		s.ChannelMessageSend(m.ChannelID, "Adding lore for '"+itemname+"' for "+m.Author.Username+". \n"+"Paste information and type !loreend to end.")
	}

	return
}

// Checks incoming messages to see if they need to be added to an incoming lore
func loreAddInput(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild, msg string) {

	// quit if loreadd not running or if a different user
	if !LOREADDFLAG || m.Author.ID != LOREADDUSER_ID {
		return
	}

	// ignore all lines starting with !, since we don't want to write to the lorefile user commands
	if m.Content[0] == '!' {
		return
	}

	// build the lore
	LOREADDGLOBALLIST = append(LOREADDGLOBALLIST, msg)
	log.Info("Debug: loreAddInput adding:" + msg)

	return
}

// adds lores to database
func loreAddEnd(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild) {

	log.Info("Debug: loreAddEnd start")

	if m.Author.ID != LOREADDUSER_ID {
		s.ChannelMessageSend(m.ChannelID, "Error, !loreadd not running for "+m.Author.Username+".")
		return
	}

	// build the final lore output
	// todo add error checking
	lines := strings.Join(LOREADDGLOBALLIST[0:], "\r\n")

	if len(lines) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Loreadd error: User entered a blank lore, aborting !loreadd.")
	} else {
		// output lore for debug
		//s.ChannelMessageSend(m.ChannelID, "Full lore was:" + "\n" + lines)

		// hardcoded for now, change to init file
		dir := "//mnt//disks//nowbot-storage//lores//"
		log.Info("Loreadd directory: " + dir)
		path := dir + "//" + LOREADDITEMNAME + ".txt"
		log.Info("Loreadd item filename: " + path)

		file, err := os.Create(path)
		if err != nil {
			log.Info("Debug: loreadd file open problem")
			return
		}
		defer file.Close()

		// write to file
		w := bufio.NewWriter(file)
		w.WriteString(lines + "\r\n")

		// log who added the lore
		t := time.Now()
		w.WriteString(t.Format("2006-01-02") + " by " + LOREADDUSER_USERNAME)

		w.Flush()

		s.ChannelMessageSend(m.ChannelID, "Finished inputting lore for '"+LOREADDITEMNAME+"' for "+LOREADDUSER_USERNAME+".")
	}

	// reset the input variables for next time
	LOREADDFLAG = false
	LOREADDUSER_ID = "0"
	LOREADDUSER_USERNAME = ""
	LOREADDSTARTTIME = time.Now()
	LOREADDGLOBALLIST = nil
	LOREADDITEMNAME = ""

	return
}

// Handles bot operator messages
func handleBotControlMessages(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild, msg string) {
	if scontains(parts[0], "!nowbot") {
		log.Info("Debug: !nowbot trying to output")
		s.ChannelMessageSend(m.ChannelID, "Owner !nowbot, with message "+msg)
		s.ChannelMessageSend(m.ChannelID, "World list is "+strings.Join(GLOBALLIST[:], " "))
		log.Info("Debug: !nowbot done trying to output")
	}
	log.Info("Debug: handleBotControlMessages finished")
}

// Handles user messages
func handleUserCommandMessages(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild, msg string) {

	// If loreAdd is running and user enters command to end, run the end function.
	if scontains(parts[0], "!loreend") {
		loreAddEnd(s, m, parts, g)
	}
	// Search database for hits
	if scontains(parts[0], "!lore") {
		log.Info("Debug: !lore trying to output")
		locallorelist := loreQuery(s, m, parts, g, msg)
		if locallorelist == nil {
			log.Info("Debug: lorequery failed")
		} else {
			GLOBALLIST = locallorelist
			log.Info(strings.Join(GLOBALLIST[1:], " "))
		}
	}
	// return results from search on database
	if scontains(parts[0], "!lorestats") {
		log.Info("Debug: !lorestats trying to output")
		if len(parts) < 2 {
			log.Info("Debug: Lorestats user didn't enter an argument")
			s.ChannelMessageSend(m.ChannelID, "Error on !lorestats, you need to enter a lore number.")
		} else {
			lorehit, _ := strconv.Atoi(parts[1])
			if lorehit > len(GLOBALLIST)-1 {
				log.Info("Debug: !lorestats argument is bigger than globallist length")
				s.ChannelMessageSend(m.ChannelID, "Error on !lorestats, the item number you entered is too high.")
			} else if lorehit == 0 {
				log.Info("Debug: !lorestats argument is zero")
				s.ChannelMessageSend(m.ChannelID, "Error on !lorestats, you entered a zero item number.")
			} else {
				loreStats(s, m, g, lorehit)
			}
		}
	}

	// start the program to add a new lore
	if scontains(parts[0], "!loreadd") {
		loreAddStart(s, m, parts, g)
	}

	log.Info("Debug: handleUserCommandMessages finished")
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Info("Recieved READY payload")
	//Idletime and Game
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
	if m.Author.ID == NOWBOT_ID || m.Author.Bot {
		return
	}

	// exit if message is nil
	if len(m.Content) <= 0 {
		return
	}

	// clean up message
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

	// check to see if user is adding a lore
	loreAddInput(s, m, parts, guild, msg)

	// exit if message does not contain command character @ mention
	if m.Content[0] != '!' && len(m.Mentions) < 1 {
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
