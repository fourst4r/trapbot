package main

import "fmt"

type FAH struct {
	Earned      int    `json:"earned"`
	Contributed int    `json:"contributed"`
	TeamTotal   int64  `json:"team_total"`
	TeamName    string `json:"team_name"`
	TeamURL     string `json:"team_url"`
	TeamRank    int    `json:"team_rank"`
	TeamURLLogo string `json:"team_urllogo"`
	URL         string `json:"url"`
}

func FAHStats(user, team string) (*FAH, error) {
	var fah FAH
	url := fmt.Sprintf("https://api.foldingathome.org/user/%s/stats?team=%s", user, team)
	err := getUnmarshal(url, &fah)
	return &fah, err
}

func (f *FAH) NumTokens() int {
	var ntokens int
	if f.Contributed >= 100_000_000 {
		ntokens = 8
	} else if f.Contributed >= 50_000_000 {
		ntokens = 7
	} else if f.Contributed >= 25_000_000 {
		ntokens = 6
	} else if f.Contributed >= 10_000_000 {
		ntokens = 5
	} else if f.Contributed >= 1_000_000 {
		ntokens = 4
	} else if f.Contributed >= 1_000 {
		ntokens = 3
	} else if f.Contributed >= 500 {
		ntokens = 2
	} else if f.Contributed >= 1 {
		ntokens = 1
	}
	return ntokens
}
