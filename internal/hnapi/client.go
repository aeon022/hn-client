package hnapi

import (
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Client struct {
	Cookie string // Wert des "user" Session-Cookies
}

// Login versucht sich bei Hacker News anzumelden und liefert bei Erfolg einen Client.
func Login(username, password string) (*Client, error) {
	formData := url.Values{}
	formData.Set("acct", username)
	formData.Set("pw", password)
	formData.Set("goto", "news")

	req, err := http.NewRequest("POST", "https://news.ycombinator.com/login", strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// Redirects unterbinden, um den Set-Cookie Header der 302-Antwort lesen zu können.
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userCookie string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "user" {
			userCookie = cookie.Value
			break
		}
	}

	if userCookie == "" {
		return nil, errors.New("Login fehlgeschlagen. Bitte Benutzernamen und Passwort überprüfen.")
	}

	return &Client{Cookie: userCookie}, nil
}

// SubmitStory veröffentlicht einen neuen Beitrag auf Hacker News (entweder Link oder Text).
func (c *Client) SubmitStory(title, urlStr, text string) error {
	// 1. GET auf Submit-Seite ausführen, um das CSRF-Token (fnid) zu holen
	req, err := http.NewRequest("GET", "https://news.ycombinator.com/submit", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", fmt.Sprintf("user=%s", c.Cookie))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("nicht angemeldet oder Session abgelaufen")
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := string(bodyBytes)

	// Regex zum Extrahieren von <input type="hidden" name="fnid" value="TOKEN_VALUE">
	fnidRegex := regexp.MustCompile(`name="fnid"\s+value="([^"]+)"`)
	matches := fnidRegex.FindStringSubmatch(body)
	if len(matches) < 2 {
		return errors.New("CSRF-Token (fnid) konnte nicht geladen werden. Bitte melde dich erneut an.")
	}
	fnid := matches[1]

	// 2. POST senden zum Einreichen
	formData := url.Values{}
	formData.Set("fnid", fnid)
	formData.Set("title", title)
	formData.Set("url", urlStr)
	formData.Set("text", text)

	postReq, err := http.NewRequest("POST", "https://news.ycombinator.com/r", strings.NewReader(formData.Encode()))
	if err != nil {
		return err
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("Cookie", fmt.Sprintf("user=%s", c.Cookie))
	postReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	postResp, err := client.Do(postReq)
	if err != nil {
		return err
	}
	defer postResp.Body.Close()

	if postResp.StatusCode == http.StatusFound {
		return nil
	}

	if postResp.StatusCode == http.StatusOK {
		bodyBytes, _ := io.ReadAll(postResp.Body)
		bodyText := string(bodyBytes)

		if strings.Contains(bodyText, "already been submitted") {
			return errors.New("Diese URL wurde bereits eingereicht.")
		}
		if strings.Contains(bodyText, "too fast") {
			return errors.New("Du postest zu schnell. Bitte warte einige Minuten.")
		}

		cleanRegex := regexp.MustCompile("<[^>]*>")
		plainText := cleanRegex.ReplaceAllString(bodyText, " ")
		plainText = regexp.MustCompile(`\s+`).ReplaceAllString(plainText, " ")
		plainText = strings.TrimSpace(plainText)

		if len(plainText) > 0 {
			runes := []rune(plainText)
			if len(runes) > 150 {
				plainText = string(runes[:150]) + "..."
			}
			return fmt.Errorf("HN: %s", plainText)
		}

		return errors.New("Beitrag wurde von Hacker News abgelehnt.")
	}

	return fmt.Errorf("Fehler beim Veröffentlichen: HTTP Status %d", postResp.StatusCode)
}

// SubmitComment antwortet auf eine Story oder einen Kommentar.
func (c *Client) SubmitComment(parentID int, text string) error {
	// 1. GET auf Reply-Seite ausführen, um hmac und fnid CSRF-Token zu holen
	replyURL := fmt.Sprintf("https://news.ycombinator.com/reply?id=%d", parentID)
	req, err := http.NewRequest("GET", replyURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", fmt.Sprintf("user=%s", c.Cookie))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := string(bodyBytes)

	// fnid extrahieren
	fnidRegex := regexp.MustCompile(`name="fnid"\s+value="([^"]+)"`)
	fnidMatches := fnidRegex.FindStringSubmatch(body)
	if len(fnidMatches) < 2 {
		return errors.New("CSRF-Token (fnid) konnte nicht geladen werden")
	}
	fnid := fnidMatches[1]

	// hmac extrahieren
	hmacRegex := regexp.MustCompile(`name="hmac"\s+value="([^"]+)"`)
	hmacMatches := hmacRegex.FindStringSubmatch(body)
	if len(hmacMatches) < 2 {
		return errors.New("CSRF-Token (hmac) konnte nicht geladen werden")
	}
	hmac := hmacMatches[1]

	// 2. POST senden zum Absenden des Kommentars
	formData := url.Values{}
	formData.Set("parent", fmt.Sprintf("%d", parentID))
	formData.Set("goto", fmt.Sprintf("item?id=%d", parentID))
	formData.Set("hmac", hmac)
	formData.Set("fnid", fnid)
	formData.Set("text", text)

	postReq, err := http.NewRequest("POST", "https://news.ycombinator.com/comment", strings.NewReader(formData.Encode()))
	if err != nil {
		return err
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("Cookie", fmt.Sprintf("user=%s", c.Cookie))
	postReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	postResp, err := client.Do(postReq)
	if err != nil {
		return err
	}
	defer postResp.Body.Close()

	if postResp.StatusCode == http.StatusFound {
		return nil
	}

	if postResp.StatusCode == http.StatusOK {
		bodyBytes, _ := io.ReadAll(postResp.Body)
		bodyText := string(bodyBytes)

		if strings.Contains(bodyText, "too fast") {
			return errors.New("Du postest zu schnell. Bitte warte einige Minuten.")
		}

		cleanRegex := regexp.MustCompile("<[^>]*>")
		plainText := cleanRegex.ReplaceAllString(bodyText, " ")
		plainText = regexp.MustCompile(`\s+`).ReplaceAllString(plainText, " ")
		plainText = strings.TrimSpace(plainText)

		if len(plainText) > 0 {
			runes := []rune(plainText)
			if len(runes) > 150 {
				plainText = string(runes[:150]) + "..."
			}
			return fmt.Errorf("HN: %s", plainText)
		}

		return errors.New("Kommentar wurde von Hacker News abgelehnt.")
	}

	return fmt.Errorf("Fehler beim Kommentieren: HTTP Status %d", postResp.StatusCode)
}

func (c *Client) DeleteItem(id int) error {
	reqURL := fmt.Sprintf("https://news.ycombinator.com/delete-confirm?id=%d", id)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", fmt.Sprintf("user=%s", c.Cookie))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("nicht angemeldet oder Session abgelaufen")
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := string(bodyBytes)

	if strings.Contains(body, "You can't delete that.") || strings.Contains(body, "Kann das nicht löschen") {
		return errors.New("Hacker News erlaubt das Löschen dieses Beitrags nicht mehr (Zeitfenster abgelaufen oder Antworten vorhanden).")
	}

	formRegex := regexp.MustCompile(`<form\s+[^>]*action="([^"]+)"`)
	formMatches := formRegex.FindStringSubmatch(body)
	if len(formMatches) < 2 {
		return errors.New("Löschen-Formular konnte nicht geladen werden (eventuell Zeitfenster abgelaufen).")
	}
	action := formMatches[1]

	inputRegex := regexp.MustCompile(`<input\s+type="hidden"\s+name="([^"]+)"\s+value="([^"]+)"`)
	inputMatches := inputRegex.FindAllStringSubmatch(body, -1)
	
	formData := url.Values{}
	for _, match := range inputMatches {
		if len(match) == 3 {
			formData.Set(match[1], match[2])
		}
	}

	formData.Set("id", fmt.Sprintf("%d", id))

	postURL := fmt.Sprintf("https://news.ycombinator.com/%s", action)
	postReq, err := http.NewRequest("POST", postURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return err
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("Cookie", fmt.Sprintf("user=%s", c.Cookie))
	postReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	postResp, err := client.Do(postReq)
	if err != nil {
		return err
	}
	defer postResp.Body.Close()

	if postResp.StatusCode == http.StatusFound {
		return nil
	}

	return fmt.Errorf("Fehler beim Löschen: HTTP Status %d", postResp.StatusCode)
}

func (c *Client) GetItemWithCookie(id int) (Item, error) {
	item, err := GetItem(id)
	if err != nil {
		return Item{}, err
	}
	if item.Title == "" && c.Cookie != "" {
		_ = c.ScrapeItemHTML(id, &item)
	}
	return item, nil
}

func (c *Client) ScrapeItemHTML(id int, item *Item) error {
	reqURL := fmt.Sprintf("https://news.ycombinator.com/item?id=%d", id)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", fmt.Sprintf("user=%s", c.Cookie))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// Use client with CheckRedirect to not follow redirects if it tries to redirect to login
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch item HTML: status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := string(bodyBytes)

	if strings.Contains(body, "Sorry.") {
		return errors.New("unauthorized or not found")
	}

	// 1. Title and URL parsing from titleline
	titleLineRegex := regexp.MustCompile(`<span class="titleline">(?:[^<]*<font[^>]*>[^<]*</font>)?\s*(?:\[flagged\]\s*)?(?:\[dead\]\s*)?<a[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	matches := titleLineRegex.FindStringSubmatch(body)
	if len(matches) >= 3 {
		href := matches[1]
		title := matches[2]
		
		cleanRegex := regexp.MustCompile("<[^>]*>")
		title = cleanRegex.ReplaceAllString(title, "")
		title = html.UnescapeString(title)
		
		item.Title = title
		if !strings.HasPrefix(href, "item?id=") {
			item.URL = href
		}
	} else {
		titleTagRegex := regexp.MustCompile(`<title>(.*?) \| Hacker News</title>`)
		tMatches := titleTagRegex.FindStringSubmatch(body)
		if len(tMatches) >= 2 {
			title := tMatches[1]
			title = regexp.MustCompile(`^(?:\[flagged\]\s*)?(?:\[dead\]\s*)?`).ReplaceAllString(title, "")
			item.Title = html.UnescapeString(title)
		}
	}

	// 2. Author (By) parsing
	authorRegex := regexp.MustCompile(`by <a href="user\?id=([^"]+)"`)
	authorMatches := authorRegex.FindStringSubmatch(body)
	if len(authorMatches) >= 2 {
		item.By = authorMatches[1]
	}

	// 3. Text (toptext) parsing
	textRegex := regexp.MustCompile(`<div class="toptext"[^>]*>(.*?)</div>`)
	textMatches := textRegex.FindStringSubmatch(body)
	if len(textMatches) >= 2 {
		item.Text = textMatches[1]
	}

	return nil
}
