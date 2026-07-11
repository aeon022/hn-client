package hnapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const BaseURL = "https://hacker-news.firebaseio.com/v0"

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// Item repräsentiert eine Story, einen Kommentar, etc.
// Wir nutzen "tags" (json:"..."), um Go zu sagen, welches JSON-Feld in welches Struct-Feld gehört.
type Item struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	By          string `json:"by"`
	Time        int64  `json:"time"`
	Text        string `json:"text"`
	URL         string `json:"url"`
	Score       int    `json:"score"`
	Title       string `json:"title"`
	Descendants int    `json:"descendants"` // Anzahl der Kommentare
	Kids        []int  `json:"kids"`        // IDs der direkten Antworten/Kinder
	Dead        bool   `json:"dead"`
	Deleted     bool   `json:"deleted"`
}

// GetStories holt die IDs der Stories einer bestimmten Kategorie (top, new, best, ask, show).
func GetStories(category string) ([]int, error) {
	resp, err := httpClient.Get(fmt.Sprintf("%s/%sstories.json", BaseURL, category))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ids []int
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return nil, err
	}

	return ids, nil
}

// GetItem holt die Details für eine bestimmte ID.
func GetItem(id int) (Item, error) {
	resp, err := httpClient.Get(fmt.Sprintf("%s/item/%d.json", BaseURL, id))
	if err != nil {
		return Item{}, err
	}
	defer resp.Body.Close()

	var item Item
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return Item{}, err
	}

	return item, nil
}

// User repräsentiert die Profildaten eines Benutzers auf Hacker News.
type User struct {
	ID        string `json:"id"`
	Created   int64  `json:"created"`
	Karma     int    `json:"karma"`
	About     string `json:"about"`
	Submitted []int  `json:"submitted"`
}

// GetUser holt die Profildaten eines Benutzers.
func GetUser(username string) (User, error) {
	resp, err := httpClient.Get(fmt.Sprintf("%s/user/%s.json", BaseURL, username))
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close()

	var u User
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return User{}, err
	}

	return u, nil
}

// GetUserSubmissions holt die IDs aller Einreichungen (Stories, Kommentare, etc.) eines Benutzers.
func GetUserSubmissions(username string) ([]int, error) {
	u, err := GetUser(username)
	if err != nil {
		return nil, err
	}
	return u.Submitted, nil
}
