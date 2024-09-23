package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"net/http"
	"net/url"
)

type Player struct {
	Success      bool        `json:"success"`
	Error        string      `json:"error,omitempty"`
	Rank         int         `json:"rank,omitempty"`
	Hats         int         `json:"hats,omitempty"`
	Group        interface{} `json:"group,omitempty"`
	Friend       int         `json:"friend,omitempty"`
	Ignored      int         `json:"ignored,omitempty"`
	Following    int         `json:"following,omitempty"`
	Status       string      `json:"status,omitempty"`
	LoginDate    int64       `json:"loginDate,omitempty"`
	RegisterDate int64       `json:"registerDate,omitempty"`
	Hat          interface{} `json:"hat,omitempty"`
	Head         interface{} `json:"head,omitempty"`
	Body         interface{} `json:"body,omitempty"`
	Feet         interface{} `json:"feet,omitempty"`
	HatColor     interface{} `json:"hatColor,omitempty"`
	HeadColor    interface{} `json:"headColor,omitempty"`
	BodyColor    interface{} `json:"bodyColor,omitempty"`
	FeetColor    interface{} `json:"feetColor,omitempty"`
	GuildID      interface{} `json:"guildId,omitempty"`
	GuildName    string      `json:"guildName,omitempty"`
	Name         string      `json:"name,omitempty"`
	UserID       interface{} `json:"userId,omitempty"`
	HatColor2    interface{} `json:"hatColor2,omitempty"`
	HeadColor2   interface{} `json:"headColor2,omitempty"`
	BodyColor2   interface{} `json:"bodyColor2,omitempty"`
	FeetColor2   interface{} `json:"feetColor2,omitempty"`
	ExpPoints    float64     `json:"exp_points,omitempty"`
	ExpToRank    int         `json:"exp_to_rank,omitempty"`
}

func (p *Player) GroupName() string {
	var group string = "???"
	switch fmt.Sprint(p.Group) {
	case "0":
		group = "Guest"
	case "1":
		group = "Member"
	case "2":
		group = "Mod"
	case "3":
		group = "Admin"
	}
	return group
}

type ArtifactHintModel struct {
	Hint        string `json:"hint"`
	FinderName  string `json:"finder_name"`
	UpdatedTime int    `json:"updated_time"`
}

type Server struct {
	ID           int32  `json:"server_id,string"`
	Name         string `json:"server_name"`
	Status       string `json:"status"`
	Address      string `json:"address"`
	Port         int16  `json:"port,string"`
	Population   int32  `json:"population,string"`
	GuildID      int32  `json:"guild_id,string"`
	IsTournament byte   `json:"tournament,string"`
	IsHappyHour  byte   `json:"happy_hour,string"`
}

type LoginRespModel struct {
	Status     string `json:"status"`
	Token      string `json:"token"`
	Email      bool   `json:"email"`
	Ant        bool   `json:"ant"`
	Time       int    `json:"time"`
	LastRead   string `json:"lastRead"`
	LastRecv   string `json:"lastRecv"`
	Guild      string `json:"guild"`
	GuildOwner int    `json:"guildOwner"`
	GuildName  string `json:"guildName"`
	Emblem     string `json:"emblem"`
	UserID     int    `json:"userId"`
}

type VersionInfo struct {
	Version string `json:"version"`
	Time    int    `json:"time"`
	URL     string `json:"url"`
}

func Login(i, version string) (*LoginRespModel, error) {
	form := url.Values{}
	form.Set("i", i)
	form.Set("version", version)

	body, err := postRef("https://pr2hub.com/login.php", form)
	if err != nil {
		return nil, err
	}

	// Unmarshal to the model.
	var model LoginRespModel
	json.Unmarshal(body, &model)

	return &model, nil
}

func CheckLogin() (string, error) {
	return getString("https://pr2hub.com/check_login.php")
}

func Level(id string, version string) (raw string, err error) {
	str, err := getString(fmt.Sprintf("https://pr2hub.com/levels/%s.txt?version=%s", id, version))
	return str, err
}

func ArtifactHint() (*ArtifactHintModel, error) {
	model := &ArtifactHintModel{}
	err := getUnmarshal("https://pr2hub.com/files/artifact_hint.txt", model)
	if err != nil {
		return nil, err
	}
	return model, nil
}

func PlayerInfo(name string) (Player, error) {
	v := Player{}
	err := getUnmarshal("https://pr2hub.com/get_player_info.php?name="+url.QueryEscape(name), &v)
	if err != nil {
		return Player{}, err
	}
	return v, nil
}

func SearchLevels(search, page, mode, order, dir string) (url.Values, error) {
	form := url.Values{}
	form.Set("search_str", search)
	form.Set("page", page)
	form.Set("mode", mode)
	form.Set("order", order)
	form.Set("dir", dir)
	resp, err := postRef("https://pr2hub.com/search_levels.php", form)
	if err != nil {
		return nil, err
	}
	values, err := url.ParseQuery(string(resp))
	if err != nil {
		return nil, err
	}
	return values, nil
}

func ServerStatus() ([]Server, error) {
	var svrs = []Server{}
	jsonStr, err := getString("https://pr2hub.com/files/server_status_2.txt")
	if err != nil {
		return nil, err
	}
	jsonStr = strings.TrimPrefix(jsonStr, "{\"servers\":")
	jsonStr = strings.TrimSuffix(jsonStr, "}")

	err = json.Unmarshal([]byte(jsonStr), &svrs)
	if err != nil {
		return nil, err
	}

	// err := getUnmarshal("https://pr2hub.com/files/server_status_2.txt", &svrs)
	// if err != nil {
	// 	return nil, err
	// }
	return svrs, nil
}

func Version() (*VersionInfo, error) {
	var version VersionInfo

	jsonStr, err := getString("https://pr2hub.com/version.txt")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(jsonStr), &version)
	if err != nil {
		return nil, err
	}

	return &version, nil
}

func getString(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func getUnmarshal(url string, v interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if strings.HasPrefix(string(body), "{\"error") {
		return errors.New(string(body))
	}

	err = json.Unmarshal(body, &v)
	if err != nil {
		return err
	}

	return nil
}

// Post with https://pr2hub.com/ referer.
func postRef(url string, form url.Values) (respBody []byte, err error) {
	postData := strings.NewReader(form.Encode())
	req, err := http.NewRequest("POST", url, postData)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://pr2hub.com/")

	// Send request.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	// Read and return response body.
	defer resp.Body.Close()
	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	return
}

func postForm(url string, form url.Values) (map[string]interface{}, error) {
	resp, err := http.PostForm(url, form)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var j map[string]interface{}
	err = json.Unmarshal(body, &j)
	if err != nil {
		return nil, err
	}
	if msg, ok := j["error"]; ok {
		return nil, errors.New(page(url) + ": " + msg.(string))
	}

	return j, nil
}

func page(url string) string {
	split := strings.Split(url, "/")
	page := split[len(split)-1]
	return page
}
