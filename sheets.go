package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

const (
	spreadsheetATID = "1xvR1BOLcFEL42wtplSbnTRVh-y2FuOkjp-1bDVFJZJo"
	spreadsheetWPID = "1DTquUKV-ayLsKU64P9w9Lcs2oddDrpiGjcOfsKsPiYk"
	playerRangeAT   = "Trap Trial!B2:D"
	readRangeAT     = "Trap Trial!E%d:%d"
	playerRangeWP   = "Ark1!B2:B51"
	readRangeWP     = "Ark1!E%d:%d"
)

var (
	// redRanges          = []string{"Trap Trial!D%d:LB", "Trap Trial!LF%d:LJ", "Trap Trial!LN%d:LP", "Trap Trial!LT%d:LW"}
	// blueRanges         = []string{readRange}
	redMaps  = []int{311, 312, 313, 319, 320, 321, 325, 326, 327, 331, 332, 333, 334}
	blueMaps = []int{314, 315, 316, 322, 323, 324, 328, 329, 330, 335, 336, 337, 338}
	srv      *sheets.Service
	spaceRgx = regexp.MustCompile(`\s\s+`)
)

const (
	TeamNone = iota
	TeamRed
	TeamBlue
)

type sheetPlayer struct {
	row  int
	name string
	team int
}

func extractSheetPlayers(vrange *sheets.ValueRange) []sheetPlayer {
	r := make([]sheetPlayer, len(vrange.Values))
	for i, row := range vrange.Values {
		if len(row) > 0 && len(row[0].(string)) > 0 {
			team := TeamNone
			if len(row) > 2 {
				switch row[2] {
				case "R":
					team = TeamRed
				case "B":
					team = TeamBlue
				}
			}
			sp := sheetPlayer{
				row:  i,
				name: row[0].(string),
				team: team,
			}
			r[i] = sp
		}
	}
	return r
}

func matchName(expr string, possible []sheetPlayer) (sheetPlayer, bool) {
	aliases := make([][]string, len(possible))
	for i, p := range possible {
		aliases[i] = strings.Split(p.name, "|")
	}

	// simple alias search
	for i, a := range aliases {
		for _, alias := range a {
			if strings.EqualFold(alias, expr) {
				return possible[i], true
			}
		}
	}

	// fuzzy search
	// playerNames := make([]string, len(possible))
	// for i := range possible {
	// 	playerNames[i] = aliases[i][0] // only check the first alias
	// }

	// var best float32 = math.MaxFloat32
	// var bestIdx int
	// for i, pname := range playerNames {
	// 	dist := float32(levenshtein.ComputeDistance(pname, expr))
	// 	threshold := 0.2 * float32(len(expr))
	// 	if dist <= threshold && dist < best {
	// 		best = dist
	// 		if dist == 0 {
	// 			return possible[i], true
	// 		}
	// 		bestIdx = i
	// 	}
	// }

	// if best < math.MaxFloat32 {
	// 	return possible[bestIdx], true
	// }

	// no match found :(
	return sheetPlayer{}, false
}

func findUnbeatenAT(players []string) ([]string, error) {
	playersResp, err := srv.Spreadsheets.Values.Get(spreadsheetATID, playerRangeAT).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve players from sheet: %v", err)
	}
	sheetPlayers := extractSheetPlayers(playersResp)

	wantRanges := []string{fmt.Sprintf(readRangeAT, 1, 1)}
	found := []string{}
	var hasRed, hasBlue bool

	for _, player := range players {

		// matched := strings.EqualFold(player, sheetPlayer.name)
		match, matched := matchName(player, sheetPlayers)
		fmt.Printf("matched %s with %s\n", player, match.name)

		if matched {
			found = append(found, player)
			wantRow := match.row + 2 // +2 to skip name and team columns
			wantRanges = append(wantRanges, fmt.Sprintf(readRangeAT, wantRow, wantRow))

			switch match.team {
			case TeamRed:
				hasRed = true
			case TeamBlue:
				hasBlue = true
			}
		}
	}

	searchFailed := len(found) != len(players)
	if searchFailed {
		// construct an informative error message
		notfound := []string{}
		for _, p := range players {
			missing := true
			for _, f := range found {
				if strings.EqualFold(p, f) {
					missing = false
				}
			}

			if missing {
				notfound = append(notfound, p)
			}
		}

		return nil, fmt.Errorf("the following names were not found on the spreadsheet: %s.\n<https://docs.google.com/spreadsheets/d/1xvR1BOLcFEL42wtplSbnTRVh-y2FuOkjp-1bDVFJZJo/edit?usp=sharing>", strings.Join(notfound, ", "))
	}

	resp, err := srv.Spreadsheets.Values.BatchGet(spreadsheetATID).Ranges(wantRanges...).Do()
	if err != nil {
		return nil, err
	}

	unbeaten := []string{}
	titles, ranges := resp.ValueRanges[0].Values[0], resp.ValueRanges[1:]
	// fmt.Println("teams:", ranges[0].Values[0][0])
	fmt.Println("titles:", len(titles))

colLoop:
	for col := 0; col < len(titles); col++ {

		// -4 because ABCD columns are not levels
		if hasBlue && !hasRed {
			for _, s := range redMaps {
				if s-4 == col {
					continue colLoop
				}
			}
		} else if hasRed && !hasBlue {
			for _, s := range blueMaps {
				if s-4 == col {
					continue colLoop
				}
			}
		} else {
			for _, s := range append(redMaps, blueMaps...) {
				if s-4 == col {
					continue colLoop
				}
			}
		}

		beat := false
		for _, valueRange := range ranges {
			// fmt.Println("checking",i,"for ")
			// if col < len(valueRange.Values[0]) {
			// v := valueRange.Values[0][col]
			// fmt.Printf("(%d) lvl: %s\n", i, v)
			// }

			if len(valueRange.Values) > 0 && col < len(valueRange.Values[0]) && valueRange.Values[0][col].(string) != "" {
				beat = true
				break
			}
		}
		if !beat {
			title := titles[col].(string)
			// spaces are used in between words to force them on the next line so fix that
			title = spaceRgx.ReplaceAllString(title, " ")
			for _, exclusion := range unbeatenExclusionsAT {
				if title == exclusion {
					continue colLoop
				}
			}
			unbeaten = append(unbeaten, title)
		}
	}
	return unbeaten, nil
}

func findUnbeatenWP(players []string) ([]string, error) {
	playersResp, err := srv.Spreadsheets.Values.Get(spreadsheetWPID, playerRangeWP).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve players from sheet: %v", err)
	}
	sheetPlayers := extractSheetPlayers(playersResp)

	wantRanges := []string{fmt.Sprintf(readRangeWP, 1, 1)}
	found := []string{}

	for _, player := range players {
		match, matched := matchName(player, sheetPlayers)
		fmt.Printf("matched %s with %s\n", player, match.name)

		if matched {
			found = append(found, player)
			wantRow := match.row + 2 // +2 to skip name and team columns
			wantRanges = append(wantRanges, fmt.Sprintf(readRangeWP, wantRow, wantRow))
		}
	}

	if len(found) != len(players) {
		notfound := []string{}
		for _, p := range players {
			missing := true
			for _, f := range found {
				if strings.EqualFold(p, f) {
					missing = false
				}
			}
			if missing {
				notfound = append(notfound, p)
			}
		}
		return nil, fmt.Errorf("the following names were not found on the spreadsheet: %s.\n<https://docs.google.com/spreadsheets/d/1DTquUKV-ayLsKU64P9w9Lcs2oddDrpiGjcOfsKsPiYk/edit#gid=0>", strings.Join(notfound, ", "))
	}

	resp, err := srv.Spreadsheets.Values.BatchGet(spreadsheetWPID).Ranges(wantRanges...).Do()
	if err != nil {
		return nil, err
	}

	unbeaten := []string{}
	titles, ranges := resp.ValueRanges[0].Values[0], resp.ValueRanges[1:]
	// fmt.Println("teams:", ranges[0].Values[0][0])
	fmt.Println("titles:", len(titles))

colLoop:
	for col := 0; col < len(titles); col++ {
		beat := false
		for _, valueRange := range ranges {
			// fmt.Println("checking",i,"for ")
			// if col < len(valueRange.Values[0]) {
			// v := valueRange.Values[0][col]
			// fmt.Printf("(%d) lvl: %s\n", i, v)
			// }

			if len(valueRange.Values) > 0 && col < len(valueRange.Values[0]) && valueRange.Values[0][col].(string) != "" {
				beat = true
				break
			}
		}
		if !beat {
			title := titles[col].(string)
			// spaces are used in between words to force them on the next line so fix that
			title = spaceRgx.ReplaceAllString(title, " ")
			for _, exclusion := range unbeatenExclusionsWP {
				if title == exclusion {
					continue colLoop
				}
			}
			unbeaten = append(unbeaten, title)
		}
	}
	return unbeaten, nil
}

func initSheets() {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err = sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
