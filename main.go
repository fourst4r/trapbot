package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	ltgeneralID      = "449529653871247373"
	trappersFile     = "trappers.txt"
	exclusionsATFile = "exclusionsat.txt"
	exclusionsWPFile = "exclusionswp.txt"
	freeFile         = "freealts.txt"
	prefix           = "!"
	redoEmoji        = "♻"
	successEmoji     = "☑"
	failureEmoji     = "❗"
)

var (
	trappers, freeAlts, unbeatenExclusionsAT, unbeatenExclusionsWP []string
	editorID                                                       = []string{
		"254635501074513920", // oxy
		"268870756862001153", // mag
	}
)

func saveTrappers() {
	sort.Strings(trappers)
	data := strings.Join(trappers, "\r\n")
	err := ioutil.WriteFile(trappersFile, []byte(data), 0)
	if err != nil {
		fmt.Println("error writing to trappers file:", err)
		return
	}
}

func loadCfg() {
	splitLines := func(b []byte) []string {
		var lines []string
		sc := bufio.NewScanner(bytes.NewBuffer(b))
		for sc.Scan() {
			lines = append(lines, sc.Text())
		}
		return lines
	}

	data, err := ioutil.ReadFile(trappersFile)
	if err != nil {
		panic(err)
	}
	trappers = splitLines(data)

	data, err = ioutil.ReadFile(exclusionsATFile)
	if err != nil {
		panic(err)
	}
	unbeatenExclusionsAT = splitLines(data)

	data, err = ioutil.ReadFile(exclusionsWPFile)
	if err != nil {
		panic(err)
	}
	unbeatenExclusionsWP = splitLines(data)

	data, err = ioutil.ReadFile(freeFile)
	if err != nil {
		panic(err)
	}
	freeAlts = splitLines(data)
}

func isEditor(id string) bool {
	for _, eid := range editorID {
		if id == eid {
			return true
		}
	}
	return false
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

	split := strings.SplitN(m.Content[len(prefix):], " ", 2)
	command, rest, args := split[0], "", []string{}
	if len(split) > 1 {
		rest = split[1]
		args = strings.Split(rest, " ")
	}

	fmt.Printf("command=%s args=%v\n", command, args)
	switch command {
	case "view":
		pi, err := PlayerInfo(rest)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error occurred fetching player info: `"+err.Error()+"`")
			return
		}
		if !pi.Success {
			s.MessageReactionAdd(m.ChannelID, m.ID, failureEmoji)
			return
		}
		parts := [12]string{
			pi.Hat, pi.HatColor, fmt.Sprint(pi.HatColor2),
			pi.Head, pi.HeadColor, fmt.Sprint(pi.HeadColor2),
			pi.Body, pi.BodyColor, fmt.Sprint(pi.BodyColor2),
			pi.Feet, pi.FeetColor, fmt.Sprint(pi.FeetColor2),
		}

		file, err := generatePR2Avi(parts)
		if err != nil {
			fmt.Println("Error occurred generating avi: " + err.Error())
		}
		defer file.Close()

		b, err := io.ReadAll(file)
		if err != nil {
			fmt.Println("Error occurred reading file: " + err.Error())
		}

		var group string = "???"
		switch pi.Group {
		case "0":
			group = "Guest"
		case "1":
			group = "Member"
		case "2":
			group = "Mod"
		case "3":
			group = "Admin"
		}
		joined := time.Unix(pi.RegisterDate, 0).UTC().Format(time.RFC822)
		active := time.Unix(pi.LoginDate, 0).UTC().Format(time.RFC822)
		description := fmt.Sprintf("_%s_\n"+
			"Group: %s\n"+
			"Guild: %s\n"+
			"Rank: %d\n"+
			"Hats: %d\n"+
			"Joined: %s\n"+
			"Active: %s\n", pi.Status, group, pi.GuildName, pi.Rank, pi.Hats, joined, active)

		s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: "",
			Files: []*discordgo.File{
				{
					Name:        "avi.png",
					ContentType: "image/png",
					Reader:      bytes.NewBuffer(b),
				},
			},
			Embed: &discordgo.MessageEmbed{
				Title:       "-- " + pi.Name + " --",
				Description: description,
				Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: "attachment://avi.png"},
			},
		})

	case "pick":
		trapper := randomTrapper()
		msg, err := s.ChannelMessageSend(m.ChannelID, trapper)
		if err != nil {
			fmt.Println("failed to send message in", m.ChannelID, ":", trapper)
			return
		}
		err = s.MessageReactionAdd(msg.ChannelID, msg.ID, redoEmoji)
		if err != nil {
			return
		}
		for _, alt := range freeAlts {
			if strings.EqualFold(trapper, alt) {
				s.MessageReactionAdd(msg.ChannelID, msg.ID, "🆓")
				break
			}
		}
	case "has":
		name := rest
		for _, trapper := range trappers {
			if strings.EqualFold(trapper, name) {
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
	case "at", "wp":
		if m.ChannelID != "487193614598930447" {
			s.ChannelMessageSend(m.ChannelID, "This command is spammy, use #bot")
			return
		}
		names := strings.Split(rest, ",")
		for i := range names {
			names[i] = strings.TrimSpace(names[i])
		}
		if len(names[0]) == 0 {
			names[0] = m.Member.Nick
		}
		var at = command == "at"
		var unbeaten []string
		var err error
		if at {
			unbeaten, err = findUnbeatenAT(names)
		} else {
			unbeaten, err = findUnbeatenWP(names)
		}
		if err != nil {
			fmt.Println("err finding unbeaten:", err)
			s.ChannelMessageSend(m.ChannelID, err.Error())
			return
		}
		var content string
		if len(unbeaten) == 0 {
			content = "no more maps to beat o_0"
		} else {
			content = strings.Join(unbeaten, ", ")
		}
		if len(content) > 2000 {
			// content = content[:1997] + "..."
			var buf bytes.Buffer
			buf.WriteString(strings.Join(unbeaten, "\r\n"))
			s.ChannelFileSend(m.ChannelID, "unbeaten.txt", &buf)
		} else {
			s.ChannelMessageSend(m.ChannelID, content)
		}
	case "doc":
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Description: "[AT information](https://docs.google.com/document/d/16vwNMfUOGWEGRDTV6IjM_QeeAM5Uh3QXgTVQ6vi0w5w/edit?usp=sharing) // [AT leaderboard](https://docs.google.com/spreadsheets/d/1xvR1BOLcFEL42wtplSbnTRVh-y2FuOkjp-1bDVFJZJo/edit?usp=sharing)\n" +
				"[WP information](https://docs.google.com/document/d/1KJx5Pha-rf5aNmvDBV_n_O7JUu8AUZkmaeHIRACeoQ4/edit) // [WP leaderboard](https://docs.google.com/spreadsheets/d/1DTquUKV-ayLsKU64P9w9Lcs2oddDrpiGjcOfsKsPiYk/)",
		})
	case "help":
		s.ChannelMessageSend(m.ChannelID, `!pick, !has <name>, !list, !at <name>,<name>, !wp <name>,<name>, !doc`)
	case "reload":
		loadCfg()
		s.MessageReactionAdd(m.ChannelID, m.ID, successEmoji)
	case "add":
		if !isEditor(m.Author.ID) {
			return
		}
		name := rest
		for _, trapper := range trappers {
			if strings.EqualFold(trapper, name) {
				s.MessageReactionAdd(m.ChannelID, m.ID, failureEmoji)
				return
			}
		}
		trappers = append(trappers, name)
		updateStatus(s)
		saveTrappers()
		s.MessageReactionAdd(m.ChannelID, m.ID, successEmoji)
	case "remove":
		if !isEditor(m.Author.ID) {
			return
		}
		name := rest
		for i, trapper := range trappers {
			if strings.EqualFold(trapper, name) {
				trappers = append(trappers[:i], trappers[i+1:]...)
				updateStatus(s)
				saveTrappers()
				s.MessageReactionAdd(m.ChannelID, m.ID, successEmoji)
				return
			}
		}
		s.MessageReactionAdd(m.ChannelID, m.ID, failureEmoji)
	}
}

func onReady(s *discordgo.Session, r *discordgo.Ready) {
	// https://discordapp.com/developers/docs/topics/gateway#resuming
	updateStatus(s)
}

func generatePR2Avi(args [12]string) (*os.File, error) {
	const bindir = "./pr2avi/Export/windows/bin/"
	argstr := strings.Join(args[:], "_")
	hash := md5.Sum([]byte(argstr))
	avipath := bindir + hex.EncodeToString(hash[:]) + ".png"

	// generate a new one if there's not a cached one
	if _, err := os.Stat(avipath); os.IsNotExist(err) {
		fmt.Println("generating pr2 avi", argstr)
		cmd := exec.Command("./pr2avi.exe", argstr)
		cmd.Dir = bindir
		err := cmd.Run()
		if err != nil {
			return nil, err
		}
	}

	fmt.Println(avipath)
	f, err := os.Open(avipath)
	return f, err
}

func main() {
	rand.Seed(time.Now().Unix())
	loadCfg()
	initSheets()

	tokenBytes, err := ioutil.ReadFile("token.txt")
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

connect:
	err = dg.Open()
	if err != nil {
		fmt.Println("dg.Open() failed: retrying in 60s")
		<-time.After(time.Minute)
		goto connect
	}

	go restartAlorter(dg)

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}

func restartAlorter(dg *discordgo.Session) {
	for t := range time.Tick(time.Minute) {
		if dg == nil {
			continue
		}
		if t.UTC().Weekday() == time.Monday {
			h, m, _ := t.UTC().Clock()
			switch h {
			case 4:
				if m == 10 {
					dg.ChannelMessageSend(ltgeneralID, "🥲 ALORT! PR2 servers will restart in 2 hours!")
				}
			case 5:
				if m == 10 {
					dg.ChannelMessageSend(ltgeneralID, "<:PepeSad:496929351179436032> ALORT! PR2 servers will restart in 1 hour!")
				}
			case 6:
				if m == 0 {
					dg.ChannelMessageSend(ltgeneralID, "<:PepeKMS:748576081967317063> ALORT! PR2 servers will restart in 10 minutes!")
				}
			}

		}
	}
}
