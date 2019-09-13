package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/agnivade/levenshtein"
	"github.com/bwmarrin/discordgo"
	pq "github.com/jupp0r/go-priority-queue"
)

const (
	token        = "NjEwMzUyODUxMTMwMTIyMjYx.XVEDGw.AeQ2tTnpKJmF3ghWK8-OoHAbEtg"
	editorID     = "254635501074513920"
	trappersFile = "trappers.txt"
	prefix       = "!"
	redoEmoji    = "♻"
	successEmoji = "☑"
	failureEmoji = "❗"
)

var trappers []string

func saveTrappers() {
	sort.Strings(trappers)
	data := strings.Join(trappers, "\r\n")
	err := ioutil.WriteFile(trappersFile, []byte(data), 0)
	if err != nil {
		panic(err)
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

func messageReactionAdd(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	// fmt.Printf("id=(%s) name=(%s) apiname=(%s)\n", m.Emoji.ID, m.Emoji.Name, m.Emoji.APIName())
	if m.UserID == s.State.User.ID || m.Emoji.Name != redoEmoji {
		return
	}
	s.ChannelMessageEdit(m.ChannelID, m.MessageID, randomTrapper())
	// s.MessageReactionRemove(m.ChannelID, m.MessageID, m.Emoji.Name, m.UserID)
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, prefix) {
		return
	}

	split := strings.Split(m.Content[len(prefix):], " ")
	command := split[0]
	var args []string
	if len(split) > 1 {
		args = split[1:]
	}

	fmt.Printf("command=%s args=%v\n", command, args)
	switch command {
	case "pick":
		msg, err := s.ChannelMessageSend(m.ChannelID, randomTrapper())
		if err != nil {
			panic(err)
		}
		err = s.MessageReactionAdd(msg.ChannelID, msg.ID, redoEmoji)
		if err != nil {
			panic(err)
		}
	case "add":
		if m.Author.ID != editorID {
			return
		}
		name := strings.Join(args, " ")
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
		name := strings.Join(args, " ")
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
		name := strings.Join(args, " ")
		for _, trapper := range trappers {
			if strings.ToLower(trapper) == strings.ToLower(name) {
				s.MessageReactionAdd(m.ChannelID, m.ID, successEmoji)
				return
			}
		}
		s.MessageReactionAdd(m.ChannelID, m.ID, failureEmoji)
	case "search":
		name := strings.Join(args, " ")
		if len(name) > 20 {
			s.ChannelMessageSend(m.ChannelID, failureEmoji+"search too long")
			return
		}
		if len(name) == 0 {
			s.ChannelMessageSend(m.ChannelID, failureEmoji+"search too short")
			return
		}
		const maxMatches = 5
		var matches []string
		q := pq.New()
		for _, trapper := range trappers {
			if true || strings.ToLower(trapper) == strings.ToLower(name) {
				dist := levenshtein.ComputeDistance(strings.ToLower(trapper), strings.ToLower(name))
				q.Insert(trapper, float64(dist))

				// matches = append(matches, trapper)
				// if len(matches) == maxMatches {
				// 	break
				// }
			}
		}
		if false && len(matches) == 0 {
			s.ChannelMessageSend(m.ChannelID, "no matches found")
		} else {
			for i := 0; i < 5; i++ {
				match, err := q.Pop()
				if err != nil {
					break
				}
				matches = append(matches, match.(string))
			}
			s.ChannelMessageSend(m.ChannelID, strings.Join(matches, ", "))
		}
	case "list":
		file, err := os.Open(trappersFile)
		if err != nil {
			s.MessageReactionAdd(m.ChannelID, m.ID, failureEmoji)
			return
		}
		s.ChannelFileSend(m.ChannelID, trappersFile, file)
	case "help":
		s.ChannelMessageSend(m.ChannelID, `!pick, !has <name>, !list`)
	}
}

func main() {
	rand.Seed(time.Now().Unix())
	loadTrappers()

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(messageCreate)
	dg.AddHandler(messageReactionAdd)

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	dg.UpdateStatus(0, fmt.Sprintf("with %d searches", len(trappers)))

	fmt.Println("Bot is now running.  Press CTRL+C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}
