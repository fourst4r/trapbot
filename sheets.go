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
	spreadsheetID = "1xvR1BOLcFEL42wtplSbnTRVh-y2FuOkjp-1bDVFJZJo"
	playerRange   = "Trap Trial!B2:B"
	readRange     = "Trap Trial!D%d:%d"
)

var (
	srv                *sheets.Service
	spaceRgx           = regexp.MustCompile(`\s\s+`)
	unbeatenExclusions = []string{
		"Obstacle: Trapper Quiz",
		"Obstacle: Tib's Quiz",
		"Lost Island [!]",
		"E-GIRLS",
	}
)

func findUnbeaten(players []string) ([]string, error) {
	// Prints the names and majors of students in a sample spreadsheet:
	// https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit
	// spreadsheetId := "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms"
	// readRange := "Class Data!A2:E"
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, playerRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	if len(resp.Values) == 0 {
		fmt.Println("No data found.")
	} else {
		wantRanges := []string{fmt.Sprintf(readRange, 1, 1)}
		for i, row := range resp.Values {
			if len(row) > 0 {
				for _, player := range players {
					if strings.ToLower(player) == strings.ToLower(row[0].(string)) {
						// fmt.Printf("%d: %s\n", i+2, row[0])
						wantRanges = append(wantRanges, fmt.Sprintf(readRange, i+2, i+2))
					}
				}
			}
		}
		// fmt.Println("rangesnym:", len(wantRanges))
		resp, err := srv.Spreadsheets.Values.BatchGet(spreadsheetID).Ranges(wantRanges...).Do()
		if err != nil {
			return nil, err
		}

		unbeaten := []string{}
		titles, ranges := resp.ValueRanges[0].Values[0], resp.ValueRanges[1:]
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
				if col < len(valueRange.Values[0]) && valueRange.Values[0][col].(string) != "" {
					beat = true
					break
				}
			}
			if !beat {
				title := titles[col].(string)
				// spaces are used in between words to force them on the next line so fix that
				title = spaceRgx.ReplaceAllString(title, " ")
				for _, exclusion := range unbeatenExclusions {
					if title == exclusion {
						continue colLoop
					}
				}
				unbeaten = append(unbeaten, title)
			}
		}
		return unbeaten, nil
	}

	return []string{}, nil
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
