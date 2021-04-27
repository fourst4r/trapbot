package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	editorID     = "254635501074513920"
	trappersFile = "trappers.txt"
	prefix       = "!"
	redoEmoji    = "♻"
	successEmoji = "☑"
	failureEmoji = "❗"
)

var freeAlts = []string{
	"Abstraction",
	"Shattered Remains",
	"Free",
	"FreeRemains",
	"TrapperBob",
	"SeasonWork",
	"Popeye",
	"WaffleWork",
	"Lord of the Traps",
	"Recovered",
}

// cache of trappersFile
var trappers []string

func saveTrappers() {
	sort.Strings(trappers)
	data := strings.Join(trappers, "\r\n")
	err := ioutil.WriteFile(trappersFile, []byte(data), 0)
	if err != nil {
		fmt.Println("error writing to trappers file:", err)
		return
	}
}

func loadTrappers() {
	file, err := os.Open(trappersFile)
	if err != nil {
		panic(err)
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	trappers = strings.Split(string(data), "\r\n")
}

func randomTrapper() string {
	return escapeFormatting(trappers[rand.Intn(len(trappers))])
}

func escapeFormatting(s string) string {
	s = strings.ReplaceAll(s, "*", "\\*")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

func updateStatus(s *discordgo.Session) {
	s.UpdateStatus(0, fmt.Sprintf("with %d searches", len(trappers)))
}

func onMessageReactionAdd(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	// fmt.Printf("id=(%s) name=(%s) apiname=(%s)\n", m.Emoji.ID, m.Emoji.Name, m.Emoji.APIName())
	if m.UserID == s.State.User.ID || m.Emoji.Name != redoEmoji {
		return
	}
	s.ChannelMessageEdit(m.ChannelID, m.MessageID, randomTrapper())
	// s.MessageReactionRemove(m.ChannelID, m.MessageID, m.Emoji.Name, m.UserID)
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, prefix) {
		return
	}

	// split := strings.Split(m.Content[len(prefix):], " ")
	// command := split[0]
	// var args []string
	// if len(split) > 1 {
	// args = split[1:]
	// }
	split := strings.SplitN(m.Content[len(prefix):], " ", 2)
	command, rest, args := split[0], "", []string{}
	if len(split) > 1 {
		rest = split[1]
		args = strings.Split(rest, " ")
	}

	fmt.Printf("command=%s args=%v\n", command, args)
	switch command {
	case "pick":
		trapper := randomTrapper()
		msg, err := s.ChannelMessageSend(m.ChannelID, trapper)
		if err != nil {
			panic(err)
		}
		err = s.MessageReactionAdd(msg.ChannelID, msg.ID, redoEmoji)
		if err != nil {
			panic(err)
		}
		for _, alt := range freeAlts {
			if strings.EqualFold(trapper, alt) {
				s.MessageReactionAdd(msg.ChannelID, msg.ID, "🆓")
				break
			}
		}
	case "add":
		if m.Author.ID != editorID {
			return
		}
		name := rest
		for _, trapper := range trappers {
			if strings.ToLower(trapper) == strings.ToLower(name) {
				s.MessageReactionAdd(m.ChannelID, m.ID, failureEmoji)
				return
			}
		}
		trappers = append(trappers, name)
		updateStatus(s)
		saveTrappers()
		s.MessageReactionAdd(m.ChannelID, m.ID, successEmoji)
	case "remove":
		if m.Author.ID != editorID {
			return
		}
		name := rest
		for i, trapper := range trappers {
			if strings.ToLower(trapper) == strings.ToLower(name) {
				trappers = append(trappers[:i], trappers[i+1:]...)
				updateStatus(s)
				saveTrappers()
				s.MessageReactionAdd(m.ChannelID, m.ID, successEmoji)
				return
			}
		}
		s.MessageReactionAdd(m.ChannelID, m.ID, failureEmoji)
	case "has":
		name := rest
		for _, trapper := range trappers {
			if strings.ToLower(trapper) == strings.ToLower(name) {
				s.MessageReactionAdd(m.ChannelID, m.ID, successEmoji)
				return
			}
		}
		s.MessageReactionAdd(m.ChannelID, m.ID, failureEmoji)
	case "list":
		file, err := os.Open(trappersFile)
		if err != nil {
			fmt.Println("error opening trappers file:", err)
			s.MessageReactionAdd(m.ChannelID, m.ID, failureEmoji)
			return
		}
		s.ChannelFileSend(m.ChannelID, trappersFile, file)
	case "at":
		if m.ChannelID != "487193614598930447" {
			s.ChannelMessageSend(m.ChannelID, "This command is spammy, use #bot")
			return
		}
		names := strings.Split(rest, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		unbeaten, err := findUnbeaten(names)
		if err != nil {
			fmt.Println("err finding unbeaten:", err)
			s.ChannelMessageSend(m.ChannelID, err.Error())
			return
		}
		content := strings.Join(unbeaten, ", ")
		if len(content) > 2000 {
			// content = content[:1997] + "..."
			var buf bytes.Buffer
			buf.WriteString(strings.Join(unbeaten, "\r\n"))
			s.ChannelFileSend(m.ChannelID, "unbeaten.txt", &buf)
		} else {
			s.ChannelMessageSend(m.ChannelID, content)
		}
	case "help":
		s.ChannelMessageSend(m.ChannelID, `!pick, !has <name>, !list, !at <name>,<name>`)
	}
}

func onReady(s *discordgo.Session, r *discordgo.Ready) {
	// https://discordapp.com/developers/docs/topics/gateway#resuming
	updateStatus(s)
}

func main() {

	rand.Seed(time.Now().Unix())
	loadTrappers()
	initSheets()

	file, err := os.Open("token.txt")
	if err != nil {
		panic(err)
	}
	tokenBytes, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	dg, err := discordgo.New("Bot " + string(tokenBytes))
	if err != nil {
		panic(err)
	}

	dg.AddHandler(onReady)
	dg.AddHandler(onMessageCreate)
	dg.AddHandler(onMessageReactionAdd)

	err = dg.Open()
	if err != nil {
		panic(err)
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}
