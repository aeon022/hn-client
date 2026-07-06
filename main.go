package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gweiher/hn-client/internal/hnapi"
)

var (
	// Farben & Styles
	orange   = lipgloss.Color("#FF6600")
	gray     = lipgloss.Color("#828282")
	lightGray = lipgloss.Color("#C0C0C0")
	white    = lipgloss.Color("#FFFFFF")
	darkGray = lipgloss.Color("#333333")
	cyan     = lipgloss.Color("#00F0FF")
	blue     = lipgloss.Color("#85A5FF")
	purple   = lipgloss.Color("#FF85FF")
	green    = lipgloss.Color("#ADFF2F")

	commentColors = []lipgloss.Color{
		orange,
		cyan,
		green,
		purple,
	}

	titleStyle = lipgloss.NewStyle().
		Foreground(white).
		Background(orange).
		Padding(0, 1).
		Bold(true)

	headerStyle = lipgloss.NewStyle().
		Foreground(orange).
		Bold(true).
		Padding(1, 0)

	selectedTitleStyle = lipgloss.NewStyle().
		Foreground(orange).
		Bold(true)

	unselectedTitleStyle = lipgloss.NewStyle().
		Foreground(white)

	readTitleStyle = lipgloss.NewStyle().
		Foreground(gray)

	selectedBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(orange).
		PaddingLeft(2)

	unselectedBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(darkGray).
		PaddingLeft(2)

	detailStyle = lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(orange)

	commentStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		PaddingLeft(1).
		MarginLeft(1)

	footerStyle = lipgloss.NewStyle().
		Foreground(darkGray).
		Border(lipgloss.NormalBorder(), true, false, false, false)
)

type writeStep int

const (
	stepNone writeStep = iota
	stepLoginPassword
	stepStoryWriting
	stepCommentWriting
)

type config struct {
	Username      string `json:"username"`
	ShowDead      bool   `json:"show_dead"`
	UseGUIBrowser bool   `json:"use_gui_browser"`
	Cookie        string `json:"cookie"`
}

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".hn-config.json"
	}
	return home + "/.hn-config.json"
}

func saveConfig(cfg config) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(getConfigPath(), data, 0600)
}

func loadConfig() (config, error) {
	path := getConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func getLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "hn-client.log"
	}
	return home + "/.hn-client.log"
}

func logError(format string, args ...interface{}) {
	path := getLogPath()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintf(f, "[%s] %s\n", timestamp, msg)
}

// statusMsg wird gesendet, wenn die Story-Liste geladen ist.
type statusMsg []hnapi.Item

// commentsMsg wird gesendet, wenn die Kommentare für eine Story geladen sind.
type commentsMsg struct {
	storyID  int
	comments []comment
}

type comment struct {
	item     hnapi.Item
	children []comment
	indent   int
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type state int

const (
	stateList state = iota
	stateDetail
)

var categories = []string{"top", "new", "best", "ask", "show", "mine"}

type model struct {
	stories                []hnapi.Item
	comments               map[int][]comment
	err                    error
	loading                bool
	loadingComments        bool
	width                  int
	height                 int
	cursor                 int
	state                  state
	viewport               viewport.Model
	ready                  bool
	category               string
	showHelp               bool
	searchActive           bool
	searchInput            textinput.Model
	username               string
	usernameInput          textinput.Model
	passwordInput          textinput.Model
	cookie                 string
	writeStep              writeStep
	submitting             bool
	wizardError            string
	loginActive            bool
	showDead               bool
	useGUIBrowser          bool
	lastBrowserOpen        time.Time
	history                map[int]int64 // StoryID -> Unix-Zeitstempel des letzten Besuchs
	currentStoryLastViewed int64         // Zeitstempel des letzten Besuchs der aktuell geöffneten Story
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 50
	ti.Width = 20
	ti.TextStyle = lipgloss.NewStyle().Foreground(white)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(gray)

	ui := textinput.New()
	ui.Placeholder = "HN Username..."
	ui.CharLimit = 50
	ui.Width = 20
	ui.TextStyle = lipgloss.NewStyle().Foreground(white)
	ui.PlaceholderStyle = lipgloss.NewStyle().Foreground(gray)

	pi := textinput.New()
	pi.Placeholder = "Password..."
	pi.EchoMode = textinput.EchoPassword
	pi.CharLimit = 100
	pi.Width = 30
	pi.TextStyle = lipgloss.NewStyle().Foreground(white)
	pi.PlaceholderStyle = lipgloss.NewStyle().Foreground(gray)

	history, _ := loadHistory()
	cfg, _ := loadConfig()

	return model{
		loading:       true,
		comments:      make(map[int][]comment),
		cursor:        0,
		state:         stateList,
		category:      "top",
		showHelp:      false,
		searchActive:  false,
		searchInput:   ti,
		usernameInput: ui,
		passwordInput: pi,
		username:      cfg.Username,
		showDead:      cfg.ShowDead,
		useGUIBrowser: cfg.UseGUIBrowser,
		cookie:        cfg.Cookie,
		writeStep:     stepNone,
		history:       history,
	}
}

var linkRegex = regexp.MustCompile(`<a\s+href="([^"]+)"[^>]*>(.*?)</a>`)

// cleanHTML ist eine einfache Hilfe zum Säubern von HN Texten.
func cleanHTML(text string) string {
	t := text
	t = strings.ReplaceAll(t, "<p>", "\n\n")
	t = strings.ReplaceAll(t, "&#x27;", "'")
	t = strings.ReplaceAll(t, "&quot;", "\"")
	t = strings.ReplaceAll(t, "&gt;", ">")
	t = strings.ReplaceAll(t, "&lt;", "<")
	t = strings.ReplaceAll(t, "&amp;", "&")
	t = strings.ReplaceAll(t, "<i>", "")
	t = strings.ReplaceAll(t, "</i>", "")
	t = strings.ReplaceAll(t, "<code>", "`")
	t = strings.ReplaceAll(t, "</code>", "`")
	t = strings.ReplaceAll(t, "<pre>", "\n")
	t = strings.ReplaceAll(t, "</pre>", "\n")

	// HTML-Links säubern und lesbarer formatieren
	t = linkRegex.ReplaceAllStringFunc(t, func(m string) string {
		match := linkRegex.FindStringSubmatch(m)
		if len(match) < 3 {
			return m
		}
		url := match[1]
		linkText := match[2]

		// Eventuell verschachtelte Tags im Link-Text entfernen
		linkText = regexp.MustCompile("<[^>]*>").ReplaceAllString(linkText, "")

		// Wenn der Text identisch zur URL ist, nur die URL anzeigen
		if url == linkText || strings.TrimSuffix(url, "/") == strings.TrimSuffix(linkText, "/") {
			return url
		}
		return fmt.Sprintf("%s (%s)", linkText, url)
	})

	return t
}

// truncateString kürzt einen String auf eine maximale Länge und fügt "..." an.
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

// formatTime calculates relative time (e.g. "2 hours ago") from a Unix timestamp.
func formatTime(unixTime int64) string {
	t := time.Unix(unixTime, 0)
	duration := time.Since(t)

	if duration.Seconds() < 60 {
		return "just now"
	} else if duration.Minutes() < 60 {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	} else if duration.Hours() < 24 {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

func openURL(url string, useGUI bool) tea.Cmd {
	// 1. Browser suchen
	var browser string
	if !useGUI {
		browsers := []string{
			"w3m", "/opt/homebrew/bin/w3m", "/usr/local/bin/w3m",
			"lynx", "/opt/homebrew/bin/lynx", "/usr/local/bin/lynx", "/usr/bin/lynx",
			"links", "/opt/homebrew/bin/links",
		}

		for _, b := range browsers {
			if strings.Contains(b, "/") {
				if _, err := os.Stat(b); err == nil {
					browser = b
					break
				}
			} else {
				if path, err := exec.LookPath(b); err == nil {
					browser = path
					break
				}
			}
		}
	}

	// 2. Command vorbereiten
	var c *exec.Cmd
	isTerminalBrowser := false

	if browser != "" {
		isTerminalBrowser = true
		args := []string{url}
		if strings.Contains(browser, "lynx") {
			args = []string{
				"-accept_all_cookies",
				"-useragent=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				"-display_charset=utf-8",
				url,
			}
		} else if strings.Contains(browser, "w3m") {
			args = []string{
				"-header",
				"User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				url,
			}
		}
		c = exec.Command(browser, args...)
	} else {
		// Fallback auf Standard-Open (Browser)
		switch runtime.GOOS {
		case "darwin":
			c = exec.Command("open", url)
		case "windows":
			c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			c = exec.Command("xdg-open", url)
		}
	}

	// 3. Command ausführen
	if isTerminalBrowser {
		// tea.ExecProcess ist ein Cmd, das direkt zurückgegeben werden muss!
		return tea.ExecProcess(c, func(err error) tea.Msg {
			if err != nil {
				return errMsg{fmt.Errorf("Browser Fehler: %v", err)}
			}
			return nil
		})
	} else {
		// Hintergrund-Start für 'open'
		return func() tea.Msg {
			err := c.Start()
			if err != nil {
				return errMsg{fmt.Errorf("Browser Fehler: %v", err)}
			}
			return nil
		}
	}
}

type loginResultMsg struct {
	cookie string
	err    error
}

type submitResultMsg struct {
	err error
}

func doLogin(username, password string) tea.Cmd {
	return func() tea.Msg {
		client, err := hnapi.Login(username, password)
		if err != nil {
			return loginResultMsg{err: err}
		}
		return loginResultMsg{cookie: client.Cookie}
	}
}

func doSubmitStory(cookie, title, urlStr, text string) tea.Cmd {
	return func() tea.Msg {
		client := &hnapi.Client{Cookie: cookie}
		err := client.SubmitStory(title, urlStr, text)
		return submitResultMsg{err: err}
	}
}

func doSubmitComment(cookie string, parentID int, text string) tea.Cmd {
	return func() tea.Msg {
		client := &hnapi.Client{Cookie: cookie}
		err := client.SubmitComment(parentID, text)
		return submitResultMsg{err: err}
	}
}

type externalEditorMsg struct {
	filePath string
	err      error
}

func runExternalEditor(filePath string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	if _, err := exec.LookPath(editor); err != nil && editor == "vim" {
		editor = "nano"
	}
	c := exec.Command(editor, filePath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return externalEditorMsg{filePath: filePath, err: err}
	})
}

func (m model) startStoryEditor() (model, tea.Cmd) {
	tmpFile, err := os.CreateTemp("", "hn-story-*.md")
	if err != nil {
		m.wizardError = fmt.Sprintf("Temp-Datei Fehler: %v", err)
		m.writeStep = stepNone
		return m, nil
	}
	defer tmpFile.Close()

	template := "---\nTitle: \nURL: \n---\n" +
		"=== SCHREIBE DEINEN TEXT UNTER DIESER ZEILE / WRITE YOUR TEXT BELOW THIS LINE ===\n" +
		"# HINWEIS / NOTE (Hacker News Regeln):\n" +
		"# - Wenn du oben eine 'URL' einträgst, wird dieser Textbereich unten IGNORIERT (Link-Post).\n" +
		"# - Wenn du einen reinen Text-Beitrag schreiben willst, lasse das Feld 'URL' oben LEER.\n" +
		"# - If you enter a 'URL' above, this text section below will be IGNORED (Link Post).\n" +
		"# - If you want to submit a text-only post, leave the 'URL' field above completely EMPTY.\n"

	if _, err := tmpFile.WriteString(template); err != nil {
		_ = os.Remove(tmpFile.Name())
		m.wizardError = fmt.Sprintf("Schreibfehler: %v", err)
		m.writeStep = stepNone
		return m, nil
	}

	m.writeStep = stepStoryWriting
	return m, runExternalEditor(tmpFile.Name())
}

func (m model) startCommentEditor() (model, tea.Cmd) {
	tmpFile, err := os.CreateTemp("", "hn-comment-*.md")
	if err != nil {
		m.wizardError = fmt.Sprintf("Temp-Datei Fehler: %v", err)
		m.writeStep = stepNone
		return m, nil
	}
	defer tmpFile.Close()

	template := "=== SCHREIBE DEINE ANTWORT UNTER DIESER ZEILE / WRITE YOUR REPLY BELOW THIS LINE ===\n" +
		"# (Alles unter dieser Zeile wird als Antwort übermittelt. Du kannst Markdown verwenden.)\n"

	if _, err := tmpFile.WriteString(template); err != nil {
		_ = os.Remove(tmpFile.Name())
		m.wizardError = fmt.Sprintf("Schreibfehler: %v", err)
		m.writeStep = stepNone
		return m, nil
	}

	m.writeStep = stepCommentWriting
	return m, runExternalEditor(tmpFile.Name())
}

func markdownToHN(md string) string {
	// 1. Markdown Links konvertieren: [text](url) -> text (url)
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	md = linkRegex.ReplaceAllStringFunc(md, func(match string) string {
		sub := linkRegex.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		text := sub[1]
		url := sub[2]
		if text == url || strings.TrimSuffix(text, "/") == strings.TrimSuffix(url, "/") {
			return url
		}
		return fmt.Sprintf("%s (%s)", text, url)
	})

	// 2. Bold text konvertieren: **text** -> *text*
	boldRegex := regexp.MustCompile(`\*\*([^*]+)\*\*`)
	md = boldRegex.ReplaceAllString(md, "*$1*")

	// 3. Fenced Code Blocks konvertieren: ```go ... ``` -> 2-Space Einrückung
	lines := strings.Split(md, "\n")
	var result []string
	inCodeBlock := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			result = append(result, "  "+line)
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func parseStoryFile(content string) (title, urlStr, body string, err error) {
	sep := "=== SCHREIBE DEINEN TEXT UNTER DIESER ZEILE / WRITE YOUR TEXT BELOW THIS LINE ==="
	parts := strings.SplitN(content, sep, 2)
	
	frontmatter := content
	if len(parts) == 2 {
		frontmatter = parts[0]
		body = parts[1]
	}

	// Parse frontmatter
	lines := strings.Split(frontmatter, "\n")
	inFrontmatter := false
	frontmatterDone := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFrontmatter && !frontmatterDone {
				inFrontmatter = true
			} else if inFrontmatter {
				inFrontmatter = false
				frontmatterDone = true
			}
			continue
		}

		if inFrontmatter {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.ToLower(strings.TrimSpace(parts[0]))
				val := strings.TrimSpace(parts[1])
				if key == "title" || key == "titel" {
					title = val
				} else if key == "url" || key == "link" {
					urlStr = val
				}
			}
		}
	}

	// Strip leading lines starting with # from the body (instructions)
	bodyLines := strings.Split(body, "\n")
	var cleanedBody []string
	seenContent := false
	for _, line := range bodyLines {
		trimmed := strings.TrimSpace(line)
		if !seenContent && (strings.HasPrefix(trimmed, "#") || trimmed == "") {
			continue
		}
		seenContent = true
		cleanedBody = append(cleanedBody, line)
	}

	body = strings.TrimSpace(strings.Join(cleanedBody, "\n"))
	return title, urlStr, body, nil
}

func parseCommentFile(content string) string {
	sep := "=== SCHREIBE DEINE ANTWORT UNTER DIESER ZEILE / WRITE YOUR REPLY BELOW THIS LINE ==="
	parts := strings.SplitN(content, sep, 2)
	
	body := content
	if len(parts) == 2 {
		body = parts[1]
	}

	// Strip leading lines starting with # from the body (instructions)
	bodyLines := strings.Split(body, "\n")
	var cleanedBody []string
	seenContent := false
	for _, line := range bodyLines {
		trimmed := strings.TrimSpace(line)
		if !seenContent && (strings.HasPrefix(trimmed, "#") || trimmed == "") {
			continue
		}
		seenContent = true
		cleanedBody = append(cleanedBody, line)
	}

	return strings.TrimSpace(strings.Join(cleanedBody, "\n"))
}

func fetchStories(category string, username string) tea.Cmd {
	return func() tea.Msg {
		var ids []int
		var err error
		if category == "mine" {
			if username == "" {
				return statusMsg(nil)
			}
			ids, err = hnapi.GetUserSubmissions(username)
		} else {
			ids, err = hnapi.GetStories(category)
		}
		if err != nil {
			return errMsg{err}
		}

		const limit = 30
		fetchLimit := limit
		if category == "mine" {
			fetchLimit = 50 // Fetch more IDs in case there are many comments to filter out
		}

		stories := make([]hnapi.Item, fetchLimit)
		var wg sync.WaitGroup
		var mu sync.Mutex
		for i := 0; i < fetchLimit && i < len(ids); i++ {
			wg.Add(1)
			go func(index, id int) {
				defer wg.Done()
				item, err := hnapi.GetItem(id)
				if err == nil {
					mu.Lock()
					stories[index] = item
					mu.Unlock()
				}
			}(i, ids[i])
		}
		wg.Wait()

		var finalStories []hnapi.Item
		for _, s := range stories {
			if s.ID != 0 {
				if category == "mine" && s.Type != "story" && s.Type != "poll" {
					continue
				}
				finalStories = append(finalStories, s)
			}
		}

		if len(finalStories) > limit {
			finalStories = finalStories[:limit]
		}
		return statusMsg(finalStories)
	}
}

func fetchComments(storyID int, kids []int, indent int) []comment {
	if indent > 3 {
		return nil
	}
	var res []comment
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, id := range kids {
		wg.Add(1)
		go func(cid int) {
			defer wg.Done()
			item, err := hnapi.GetItem(cid)
			if err == nil && !item.Dead && !item.Deleted {
				childComments := fetchComments(storyID, item.Kids, indent+1)
				mu.Lock()
				res = append(res, comment{item: item, children: childComments, indent: indent})
				mu.Unlock()
			}
		}(id)
	}
	wg.Wait()
	return res
}

func getHistoryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".hn-history.json"
	}
	return home + "/.hn-history.json"
}

func saveHistory(history map[int]int64) error {
	data, err := json.Marshal(history)
	if err != nil {
		return err
	}
	return os.WriteFile(getHistoryPath(), data, 0600)
}

func loadHistory() (map[int]int64, error) {
	path := getHistoryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		// Migration: Prüfen, ob lokales History-File existiert
		if localData, localErr := os.ReadFile(".hn-history.json"); localErr == nil {
			var history map[int]int64
			if err := json.Unmarshal(localData, &history); err == nil {
				_ = os.WriteFile(path, localData, 0600)
				_ = os.Remove(".hn-history.json") // Lokales File aufräumen
				return history, nil
			}
		}
		if os.IsNotExist(err) {
			return make(map[int]int64), nil
		}
		return nil, err
	}
	var history map[int]int64
	if err := json.Unmarshal(data, &history); err != nil {
		return make(map[int]int64), nil
	}
	return history, nil
}

func (m model) Init() tea.Cmd {
	return fetchStories(m.category, m.username)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := 2
		footerHeight := 2
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.MouseWheelEnabled = true
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

		m.width = msg.Width
		m.height = msg.Height

	case loginResultMsg:
		m.submitting = false
		if msg.err != nil {
			logError("Login fehlgeschlagen (Benutzer: %q): %v", m.username, msg.err)
			m.wizardError = msg.err.Error()
			m.passwordInput.SetValue("")
			m.passwordInput.Focus()
			m.updateViewport()
			return m, textinput.Blink
		}
		m.cookie = msg.cookie
		_ = saveConfig(config{
			Username:      m.username,
			ShowDead:      m.showDead,
			UseGUIBrowser: m.useGUIBrowser,
			Cookie:        m.cookie,
		})
		
		if m.state == stateList {
			m, cmd = m.startStoryEditor()
			return m, cmd
		} else {
			m, cmd = m.startCommentEditor()
			return m, cmd
		}

	case submitResultMsg:
		m.submitting = false
		m.writeStep = stepNone
		if msg.err != nil {
			logError("Submission fehlgeschlagen: %v", msg.err)
			m.wizardError = msg.err.Error()
			m.updateViewport()
			return m, nil
		}
		m.loading = true
		if m.state == stateList {
			m.cursor = 0
			m.viewport.YOffset = 0
			m.updateViewport()
			return m, fetchStories(m.category, m.username)
		} else {
			displayStories := m.getDisplayStories()
			if len(displayStories) > 0 {
				story := displayStories[m.cursor]
				m.loadingComments = true
				delete(m.comments, story.ID)
				m.updateViewport()
				return m, func() tea.Msg {
					c := fetchComments(story.ID, story.Kids, 0)
					return commentsMsg{storyID: story.ID, comments: c}
				}
			}
			m.updateViewport()
			return m, nil
		}

	case externalEditorMsg:
		m.submitting = false
		if msg.err != nil {
			logError("Vim/Editor fehlgeschlagen: %v", msg.err)
			m.wizardError = fmt.Sprintf("Editor Fehler: %v", msg.err)
			m.writeStep = stepNone
			m.updateViewport()
			return m, nil
		}

		data, err := os.ReadFile(msg.filePath)
		_ = os.Remove(msg.filePath)
		if err != nil {
			logError("Temp-Datei Lesefehler (%q): %v", msg.filePath, err)
			m.wizardError = fmt.Sprintf("Datei-Lesefehler: %v", err)
			m.writeStep = stepNone
			m.updateViewport()
			return m, nil
		}

		content := string(data)

		if m.writeStep == stepStoryWriting {
			title, urlStr, body, err := parseStoryFile(content)
			if err != nil {
				logError("Story-Template-Parsing fehlgeschlagen: %v", err)
				m.wizardError = err.Error()
				m.writeStep = stepNone
				m.updateViewport()
				return m, nil
			}

			title = strings.TrimSpace(title)
			urlStr = strings.TrimSpace(urlStr)
			body = markdownToHN(body)

			if title == "" {
				m.wizardError = "Story Title darf nicht leer sein!"
				m.writeStep = stepNone
				m.updateViewport()
				return m, nil
			}

			if urlStr == "" && body == "" {
				m.wizardError = "Entweder URL oder Text angeben!"
				m.writeStep = stepNone
				m.updateViewport()
				return m, nil
			}

			m.submitting = true
			m.updateViewport()
			return m, doSubmitStory(m.cookie, title, urlStr, body)

		} else if m.writeStep == stepCommentWriting {
			body := parseCommentFile(content)
			body = markdownToHN(body)

			if body == "" {
				m.writeStep = stepNone
				m.updateViewport()
				return m, nil
			}

			displayStories := m.getDisplayStories()
			if len(displayStories) == 0 {
				m.writeStep = stepNone
				m.updateViewport()
				return m, nil
			}
			story := displayStories[m.cursor]

			m.submitting = true
			m.updateViewport()
			return m, doSubmitComment(m.cookie, story.ID, body)
		}

		m.writeStep = stepNone
		m.updateViewport()
		return m, nil

	case statusMsg:
		m.stories = msg
		m.loading = false
		m.cursor = 0
		m.updateViewport()

	case commentsMsg:
		m.comments[msg.storyID] = msg.comments
		m.loadingComments = false
		if m.state == stateDetail {
			m.updateViewport()
		}

	case errMsg:
		logError("Genereller App-Fehler: %v", msg.err)
		m.err = msg.err
		m.loading = false
		m.loadingComments = false
		m.updateViewport()
		return m, nil

	case tea.MouseMsg:
		if m.showHelp || m.searchActive || m.loginActive || m.writeStep != stepNone || m.submitting {
			return m, nil
		}
		if m.state == stateList {
			switch msg.Button {
			case tea.MouseButtonLeft:
				if msg.Y == 0 && msg.X >= m.width-25 {
					m.loginActive = true
					m.usernameInput.Focus()
					m.usernameInput.SetValue(m.username)
					m.updateViewport()
					return m, textinput.Blink
				}
			case tea.MouseButtonWheelUp:
				if m.cursor > 0 {
					m.cursor--
					m.updateViewport()
				}
			case tea.MouseButtonWheelDown:
				displayStories := m.getDisplayStories()
				if m.cursor < len(displayStories)-1 {
					m.cursor++
					m.updateViewport()
				}
			}
			return m, nil
		}

	case tea.KeyMsg:
		if m.wizardError != "" {
			m.wizardError = ""
			m.updateViewport()
			return m, nil
		}

		if m.showHelp {
			switch msg.String() {
			case "?", "esc", "q", "enter", "space":
				m.showHelp = false
			}
			return m, nil
		}



		if m.submitting {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}

		if m.writeStep != stepNone {
			m.wizardError = ""
			switch m.writeStep {
			case stepLoginPassword:
				switch msg.String() {
				case "esc":
					m.writeStep = stepNone
					m.passwordInput.SetValue("")
					m.updateViewport()
					return m, nil
				case "enter":
					password := strings.TrimSpace(m.passwordInput.Value())
					m.passwordInput.SetValue("")
					if password == "" {
						m.writeStep = stepNone
						m.updateViewport()
						return m, nil
					}
					m.submitting = true
					m.updateViewport()
					return m, doLogin(m.username, password)
				}
				m.passwordInput, cmd = m.passwordInput.Update(msg)
				return m, cmd
			}
		}

		if m.loginActive {
			switch msg.String() {
			case "esc":
				m.loginActive = false
				m.usernameInput.SetValue("")
				m.updateViewport()
				return m, nil
			case "enter":
				m.loginActive = false
				newUsername := strings.TrimSpace(m.usernameInput.Value())
				if newUsername != m.username {
					m.username = newUsername
					m.cookie = "" // Clear cookie if username changes
				}
				_ = saveConfig(config{Username: m.username, ShowDead: m.showDead, UseGUIBrowser: m.useGUIBrowser, Cookie: m.cookie})
				// If they are on the "mine" category, reload it!
				if m.category == "mine" {
					m.loading = true
					m.cursor = 0
					m.viewport.YOffset = 0
					m.updateViewport()
					return m, fetchStories(m.category, m.username)
				}
				m.updateViewport()
				return m, nil
			}
			m.usernameInput, cmd = m.usernameInput.Update(msg)
			return m, cmd
		}

		if m.searchActive {
			switch msg.String() {
			case "esc":
				m.searchActive = false
				m.searchInput.SetValue("")
				m.updateViewport()
				return m, nil
			case "enter":
				m.searchActive = false
				m.updateViewport()
				return m, nil
			}
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.cursor = 0 // Sucheingabe verändert -> Cursor auf erstes Element zurücksetzen
			m.updateViewport()
			return m, cmd
		}

		switch msg.String() {
		case "/":
			if m.state == stateList {
				m.searchActive = true
				m.searchInput.Focus()
				m.searchInput.SetValue("")
				m.updateViewport()
				return m, textinput.Blink
			}
		case "x":
			if m.state == stateList && m.searchInput.Value() != "" {
				m.searchInput.SetValue("")
				m.cursor = 0
				m.updateViewport()
				return m, nil
			}
		case "tab":
			if m.state == stateList {
				idx := 0
				for i, cat := range categories {
					if cat == m.category {
						idx = i
						break
					}
				}
				idx = (idx + 1) % len(categories)
				m.category = categories[idx]
				m.loading = true
				m.cursor = 0
				m.viewport.YOffset = 0
				m.searchInput.SetValue("") // Clear filter on category change
				return m, fetchStories(m.category, m.username)
			}
		case "shift+tab":
			if m.state == stateList {
				idx := 0
				for i, cat := range categories {
					if cat == m.category {
						idx = i
						break
					}
				}
				idx = (idx - 1 + len(categories)) % len(categories)
				m.category = categories[idx]
				m.loading = true
				m.cursor = 0
				m.viewport.YOffset = 0
				m.searchInput.SetValue("") // Clear filter on category change
				return m, fetchStories(m.category, m.username)
			}
		case "1", "2", "3", "4", "5", "6":
			if m.state == stateList {
				idx := int(msg.String()[0] - '1')
				if idx >= 0 && idx < len(categories) {
					m.category = categories[idx]
					m.loading = true
					m.cursor = 0
					m.viewport.YOffset = 0
					m.searchInput.SetValue("") // Clear filter on category change
					return m, fetchStories(m.category, m.username)
				}
			}
		case "l":
			if m.state == stateList {
				m.loginActive = true
				m.usernameInput.Focus()
				m.usernameInput.SetValue(m.username)
				m.updateViewport()
				return m, textinput.Blink
			}
		case "h":
			if m.state == stateList {
				m.showDead = !m.showDead
				_ = saveConfig(config{Username: m.username, ShowDead: m.showDead, UseGUIBrowser: m.useGUIBrowser, Cookie: m.cookie})
				m.updateViewport()
				return m, nil
			}
		case "b":
			m.useGUIBrowser = !m.useGUIBrowser
			_ = saveConfig(config{Username: m.username, ShowDead: m.showDead, UseGUIBrowser: m.useGUIBrowser, Cookie: m.cookie})
			m.updateViewport()
			return m, nil
		case "?":
			m.showHelp = true
			return m, nil
		case "w":
			if m.state == stateList {
				if m.username == "" {
					m.loginActive = true
					m.usernameInput.Focus()
					m.usernameInput.SetValue("")
					m.updateViewport()
					return m, textinput.Blink
				}
				if m.cookie == "" {
					m.writeStep = stepLoginPassword
					m.passwordInput.Focus()
					m.passwordInput.SetValue("")
					m.updateViewport()
					return m, textinput.Blink
				}
				m, cmd = m.startStoryEditor()
				return m, cmd
			}
		case "r":
			if m.state == stateDetail {
				displayStories := m.getDisplayStories()
				if len(displayStories) == 0 {
					return m, nil
				}
				if m.username == "" {
					m.loginActive = true
					m.usernameInput.Focus()
					m.usernameInput.SetValue("")
					m.updateViewport()
					return m, textinput.Blink
				}
				if m.cookie == "" {
					m.writeStep = stepLoginPassword
					m.passwordInput.Focus()
					m.passwordInput.SetValue("")
					m.updateViewport()
					return m, textinput.Blink
				}
				m, cmd = m.startCommentEditor()
				return m, cmd
			} else {
				m.loading = true
				m.cursor = 0
				m.viewport.YOffset = 0
				return m, fetchStories(m.category, m.username)
			}
		case "q", "ctrl+c":
			if m.state == stateDetail {
				m.state = stateList
				m.viewport.SetContent("")
				m.viewport.YOffset = 0
				m.updateViewport()
				return m, nil
			}
			return m, tea.Quit
		case "up", "k":
			if m.state == stateList {
				if m.cursor > 0 {
					m.cursor--
					m.updateViewport()
				}
			} else {
				m.viewport.LineUp(1)
			}
		case "down", "j":
			if m.state == stateList {
				displayStories := m.getDisplayStories()
				if m.cursor < len(displayStories)-1 {
					m.cursor++
					m.updateViewport()
				}
			} else {
				m.viewport.LineDown(1)
			}
		case "enter":
			displayStories := m.getDisplayStories()
			if m.state == stateList && len(displayStories) > 0 {
				m.state = stateDetail
				story := displayStories[m.cursor]
				
				// 1. Letzten Besuchszeitpunkt merken
				if lastTime, exists := m.history[story.ID]; exists {
					m.currentStoryLastViewed = lastTime
				} else {
					m.currentStoryLastViewed = 0
				}
				
				// 2. Aktuellen Zeitpunkt eintragen und speichern
				m.history[story.ID] = time.Now().Unix()
				saveHistory(m.history)
				
				if _, exists := m.comments[story.ID]; !exists {
					m.loadingComments = true
					m.updateViewport()
					cmds = append(cmds, func() tea.Msg {
						c := fetchComments(story.ID, story.Kids, 0)
						return commentsMsg{storyID: story.ID, comments: c}
					})
				} else {
					m.updateViewport()
				}
			}
		case "o":
			displayStories := m.getDisplayStories()
			if len(displayStories) > 0 {
				story := displayStories[m.cursor]
				if story.URL != "" {
					// ponytail: simple time-based debounce to prevent duplicate browser launches from key repeating.
					if time.Since(m.lastBrowserOpen) < 1500*time.Millisecond {
						return m, nil
					}
					m.lastBrowserOpen = time.Now()
					return m, openURL(story.URL, m.useGUIBrowser)
				}
			}
		case "esc", "backspace":
			if m.state == stateDetail {
				m.state = stateList
				m.viewport.SetContent("")
				m.viewport.YOffset = 0
				m.updateViewport()
			}
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) getDisplayStories() []hnapi.Item {
	var stories []hnapi.Item
	for _, item := range m.stories {
		if !m.showDead && item.Dead {
			continue
		}
		stories = append(stories, item)
	}

	if m.searchInput.Value() == "" {
		return stories
	}
	var filtered []hnapi.Item
	query := strings.ToLower(m.searchInput.Value())
	for _, item := range stories {
		if strings.Contains(strings.ToLower(item.Title), query) ||
			strings.Contains(strings.ToLower(item.By), query) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (m *model) updateViewport() {
	if !m.ready {
		return
	}

	var content string
	if m.state == stateList {
		var s strings.Builder
		if m.category == "mine" && m.username == "" {
			s.WriteString("\n\n  " + lipgloss.NewStyle().Foreground(orange).Bold(true).Render("No HN Username configured!") + "\n\n")
			s.WriteString("  Press " + lipgloss.NewStyle().Foreground(white).Bold(true).Render("L") + " (or click Login top-right) to enter your\n")
			s.WriteString("  username and see your posts.\n")
			m.viewport.SetContent(s.String())
			return
		}

		displayStories := m.getDisplayStories()
		
		// Sicherstellen, dass der Cursor im Bereich der gefilterten Stories liegt
		if len(displayStories) == 0 {
			m.cursor = 0
		} else if m.cursor >= len(displayStories) {
			m.cursor = len(displayStories) - 1
		}

		for i, item := range displayStories {
			titleText := item.Title
			timeStr := formatTime(item.Time)

			hasBeenRead := m.history[item.ID] > 0
			titlePrefix := ""
			if hasBeenRead {
				titlePrefix = "✓ "
			}

			// Kürzen des Titels, um Zeilenumbruch und Scroll-Drift zu verhindern
			maxTitleLen := m.width - 6
			if hasBeenRead {
				maxTitleLen -= 2
			}
			if maxTitleLen > 10 {
				titleText = truncateString(titleText, maxTitleLen)
			}

			// Styled metadata items
			var pts, author, commentsCount string
			if hasBeenRead && m.cursor != i {
				// Dimmed for read, unselected
				pts = lipgloss.NewStyle().Foreground(darkGray).Render(fmt.Sprintf("%d pts", item.Score))
				author = lipgloss.NewStyle().Foreground(darkGray).Render(item.By)
				commentsCount = lipgloss.NewStyle().Foreground(darkGray).Render(fmt.Sprintf("%d comments", item.Descendants))
			} else {
				pts = lipgloss.NewStyle().Foreground(cyan).Render(fmt.Sprintf("%d pts", item.Score))
				author = lipgloss.NewStyle().Foreground(orange).Render(item.By)
				commentsCount = lipgloss.NewStyle().Foreground(blue).Render(fmt.Sprintf("%d comments", item.Descendants))
			}
			metaText := fmt.Sprintf("%s · by %s · %s · %s", pts, author, timeStr, commentsCount)

			var itemStr string
			if m.cursor == i {
				var title string
				if item.Dead {
					deadLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true).Render("[DEAD] ")
					title = deadLabel + selectedTitleStyle.Render(titlePrefix + titleText)
				} else {
					title = selectedTitleStyle.Render(titlePrefix + titleText)
				}
				itemStr = selectedBoxStyle.Render(title + "\n" + metaText)
			} else {
				var title string
				if item.Dead {
					deadLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true).Render("[DEAD] ")
					if hasBeenRead {
						title = deadLabel + readTitleStyle.Render(titlePrefix + titleText)
					} else {
						title = deadLabel + unselectedTitleStyle.Render(titlePrefix + titleText)
					}
				} else {
					if hasBeenRead {
						title = readTitleStyle.Render(titlePrefix + titleText)
					} else {
						title = unselectedTitleStyle.Render(titleText)
					}
				}
				itemStr = unselectedBoxStyle.Render(title + "\n" + metaText)
			}
			s.WriteString(itemStr + "\n\n")
		}
		content = s.String()

		// Scrolling-Logik für die Story-Liste:
		// Jedes Item belegt genau 4 Zeilen im Viewport (Title, Meta, newline, newline).
		// Wir passen YOffset an, damit das aktuell ausgewählte Element immer sichtbar bleibt.
		itemTop := m.cursor * 4
		itemBottom := m.cursor * 4 + 2
		if itemTop < m.viewport.YOffset {
			m.viewport.YOffset = itemTop
		} else if itemBottom >= m.viewport.YOffset+m.viewport.Height {
			m.viewport.YOffset = itemBottom - m.viewport.Height + 1
		}
	} else {
		displayStories := m.getDisplayStories()
		if len(displayStories) == 0 {
			return
		}
		if m.cursor >= len(displayStories) {
			m.cursor = 0
		}
		curr := displayStories[m.cursor]
		var s strings.Builder
		
		titleWidth := m.width - 4
		if titleWidth < 20 {
			titleWidth = 20
		}
		title := selectedTitleStyle.Width(titleWidth).Render(curr.Title)
		if curr.Dead {
			deadLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true).Render("[DEAD] ")
			title = deadLabel + title
		}
		pts := lipgloss.NewStyle().Foreground(cyan).Render(fmt.Sprintf("%d pts", curr.Score))
		author := lipgloss.NewStyle().Foreground(orange).Render(curr.By)
		timeStr := formatTime(curr.Time)
		commentsCount := lipgloss.NewStyle().Foreground(blue).Render(fmt.Sprintf("%d comments", curr.Descendants))
		meta := fmt.Sprintf("%s · by %s · %s · %s", pts, author, timeStr, commentsCount)
		
		s.WriteString(title + "\n")
		s.WriteString(meta + "\n")
		if curr.Dead {
			warningStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000")).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(lipgloss.Color("#FF0000")).
				PaddingLeft(2).
				MarginTop(1).
				MarginBottom(1)
			s.WriteString(warningStyle.Render("⚠️ WARNING: This post has been marked dead or flagged by Hacker News filters.") + "\n")
		}
		if curr.URL != "" {
			s.WriteString(lipgloss.NewStyle().Foreground(gray).Render("Link: "+curr.URL) + "\n")
		}
		s.WriteString("\n")

		if curr.Text != "" {
			text := cleanHTML(curr.Text)
			detailWidth := m.width - 8
			if detailWidth < 20 {
				detailWidth = 20
			}
			s.WriteString(detailStyle.Width(detailWidth).Render(text) + "\n\n")
		}

		s.WriteString(headerStyle.Render("── Comments ──") + "\n\n")

		if m.loadingComments {
			s.WriteString(lipgloss.NewStyle().Foreground(orange).Render("⌛ Loading comments..."))
		} else {
			comments := m.comments[curr.ID]
			if len(comments) == 0 {
				s.WriteString(lipgloss.NewStyle().Foreground(gray).Italic(true).Render("No comments available."))
			} else {
				s.WriteString(m.renderComments(comments))
			}
		}
		content = s.String()
	}

	m.viewport.SetContent(content)
}

func (m model) renderComments(comments []comment) string {
	var s strings.Builder
	for _, c := range comments {
		indentSize := c.indent * 2
		indentStr := strings.Repeat(" ", indentSize)
		
		colorIndex := c.indent % len(commentColors)
		commentBorderColor := commentColors[colorIndex]
		
		isNew := m.currentStoryLastViewed > 0 && c.item.Time > m.currentStoryLastViewed
		
		var author string
		if isNew {
			newTag := lipgloss.NewStyle().Foreground(green).Bold(true).Render(" [NEU]")
			author = lipgloss.NewStyle().Foreground(commentBorderColor).Bold(true).Render(c.item.By) + newTag
		} else {
			author = lipgloss.NewStyle().Foreground(commentBorderColor).Bold(true).Render(c.item.By)
		}
		
		text := cleanHTML(c.item.Text)
		
		commentWidth := m.width - indentSize - 10
		if commentWidth < 20 {
			commentWidth = 20
		}
		styledText := commentStyle.Copy().
			BorderForeground(commentBorderColor).
			Width(commentWidth).
			Render(text)

		s.WriteString(indentStr + author + "\n")
		s.WriteString(indentStr + styledText + "\n\n")
		s.WriteString(m.renderComments(c.children))
	}
	return s.String()
}

func (m model) renderHelp() string {
	contentWidth := 58

	title := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Foreground(white).
		Background(orange).
		Bold(true).
		Padding(0, 2).
		Render("HELP & KEYBINDINGS")

	var table strings.Builder

	section := func(name string) string {
		titleText := "── " + name + " ──"
		return lipgloss.NewStyle().
			Width(contentWidth).
			Align(lipgloss.Center).
			Foreground(orange).
			Bold(true).
			MarginTop(1).
			MarginBottom(1).
			Render(titleText)
	}

	shortcut := func(key, desc string) string {
		k := lipgloss.NewStyle().Foreground(cyan).Bold(true).Width(18).Render(key)
		d := lipgloss.NewStyle().Foreground(white).Render(desc)
		return "  " + k + "  " + d
	}

	table.WriteString(section("Navigation & Story List") + "\n")
	table.WriteString(shortcut("j / k / ↓ / ↑", "Navigate") + "\n")
	table.WriteString(shortcut("Mouse Wheel", "Scroll / Move Cursor") + "\n")
	table.WriteString(shortcut("Tab / Shift+Tab", "Switch Category") + "\n")
	table.WriteString(shortcut("1 - 6", "Direct Category Selection (6: Mine)") + "\n")
	table.WriteString(shortcut("l", "Set HN Username / Login") + "\n")
	table.WriteString(shortcut("h", "Toggle Show Dead/Flagged posts") + "\n")
	table.WriteString(shortcut("b", "Toggle Browser Mode (GUI vs Terminal)") + "\n")
	table.WriteString(shortcut("Enter", "Open Details & Comments") + "\n")
	table.WriteString(shortcut("r", "Reload Feed") + "\n")
	table.WriteString(shortcut("o", "Open Original Link") + "\n")
	table.WriteString(shortcut("w", "Write Submission (TUI)") + "\n")

	table.WriteString(section("Comments View") + "\n")
	table.WriteString(shortcut("j / k / ↓ / ↑", "Scroll") + "\n")
	table.WriteString(shortcut("Mouse Wheel", "Scroll") + "\n")
	table.WriteString(shortcut("Esc / q", "Back to Story List") + "\n")
	table.WriteString(shortcut("o", "Open Original Link") + "\n")
	table.WriteString(shortcut("r", "Reply to Thread (TUI)") + "\n")

	table.WriteString(section("General") + "\n")
	table.WriteString(shortcut("?", "Close Help Menu") + "\n")
	table.WriteString(shortcut("ctrl+c", "Quit Application") + "\n")

	modalContent := lipgloss.JoinVertical(lipgloss.Left,
		title,
		table.String(),
	)

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(orange).
		Padding(1, 2).
		Render(modalContent)

	height := m.height - 4
	if height < 1 {
		height = 1
	}
	return lipgloss.Place(m.width, height, lipgloss.Center, lipgloss.Center, modalBox)
}



func formatShortcut(key, desc string) string {
	k := lipgloss.NewStyle().Foreground(orange).Bold(true).Render(key)
	d := lipgloss.NewStyle().Foreground(gray).Render(desc)
	return fmt.Sprintf("%s %s", k, d)
}

func (m model) View() string {
	if !m.ready {
		return "  Initializing..."
	}

	if m.err != nil {
		return fmt.Sprintf("  %s\n\n  %s", 
			titleStyle.Render(" ERROR "),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(m.err.Error()))
	}

	var headerText string

	if m.state == stateList {
		var tabs []string
		for _, cat := range categories {
			label := " " + strings.ToUpper(cat) + " "
			if m.category == cat {
				tabs = append(tabs, lipgloss.NewStyle().
					Foreground(white).
					Background(orange).
					Bold(true).
					Render(label))
			} else {
				tabs = append(tabs, lipgloss.NewStyle().
					Foreground(gray).
					Render(label))
			}
		}
		tabsRow := strings.Join(tabs, " ")
		headerText = titleStyle.Render(" HACKER NEWS ") + "  " + tabsRow
		
		// If a filter is active, append it to the header
		if m.searchInput.Value() != "" {
			headerText += lipgloss.NewStyle().Foreground(gray).Italic(true).Render(fmt.Sprintf("  (Filter: %q)", m.searchInput.Value()))
		}

		// Append the Right-aligned Login Button!
		loginText := " [L] Login "
		if m.username != "" {
			loginText = " [L] User: " + m.username + " "
		}
		spaceCount := m.width - lipgloss.Width(headerText) - lipgloss.Width(loginText)
		if spaceCount > 0 {
			headerText += strings.Repeat(" ", spaceCount) + lipgloss.NewStyle().Foreground(orange).Bold(true).Render(loginText)
		} else {
			headerText += "  " + lipgloss.NewStyle().Foreground(orange).Bold(true).Render(loginText)
		}
	} else {
		// Header in reader mode
		headerText = titleStyle.Render(" HACKER NEWS ") + "  │  " + lipgloss.NewStyle().Foreground(orange).Bold(true).Render("READER MODE (Comments)")
	}

	// Divider across the entire terminal width
	divider := lipgloss.NewStyle().Foreground(darkGray).Render(strings.Repeat("─", m.width))
	header := fmt.Sprintf("%s\n%s\n", headerText, divider)

	var content string
	if m.showHelp {
		content = m.renderHelp()
	} else if m.loading {
		content = "\n\n  " + lipgloss.NewStyle().Foreground(orange).Render("⌛ Loading...")
	} else {
		content = m.viewport.View()
	}

	var footer string
	if m.submitting {
		footer = footerStyle.Width(m.width).Render(
			lipgloss.NewStyle().Foreground(orange).Bold(true).Render(" ⌛ Submitting... Please wait."),
		)
	} else if m.writeStep == stepLoginPassword {
		promptLabel := " 🔑 HN Password: "
		viewStr := m.passwordInput.View()
		var helpView string
		if m.wizardError != "" {
			helpView = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("  ❌ Error: " + m.wizardError)
		} else {
			helpView = lipgloss.NewStyle().Foreground(lightGray).Render("  (Esc: Cancel / Enter: Log in)")
		}
		label := lipgloss.NewStyle().Foreground(white).Bold(true).Render(promptLabel)
		footer = footerStyle.Width(m.width).Render(label + viewStr + helpView)
	} else if m.wizardError != "" {
		footer = footerStyle.Width(m.width).Render(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true).Render(" ❌ Error: " + m.wizardError + " (Press any key to clear)"),
		)
	} else if m.loginActive {
		loginLabel := lipgloss.NewStyle().Foreground(white).Bold(true).Render(" 👤 HN Username: ")
		footer = footerStyle.Width(m.width).Render(
			loginLabel + m.usernameInput.View() + lipgloss.NewStyle().Foreground(lightGray).Render("  (Esc: Cancel / Enter: Save)"),
		)
	} else if m.searchActive {
		searchLabel := lipgloss.NewStyle().Foreground(white).Bold(true).Render(" 🔍 Search: ")
		footer = footerStyle.Width(m.width).Render(
			searchLabel + m.searchInput.View() + lipgloss.NewStyle().Foreground(lightGray).Render("  (Esc: Cancel / Enter: Apply)"),
		)
	} else if m.state == stateList {
		browserMode := "terminal"
		if m.useGUIBrowser {
			browserMode = "gui"
		}
		shortcuts := []string{
			formatShortcut("q", "quit"),
			formatShortcut("tab", "feed"),
			formatShortcut("r", "reload"),
			formatShortcut("l", "login"),
			formatShortcut("j/k", "nav"),
			formatShortcut("enter", "view"),
			formatShortcut("o", "link"),
			formatShortcut("w", "post"),
			formatShortcut("/", "search"),
			formatShortcut("b", browserMode),
			formatShortcut("?", "help"),
		}
		if m.searchInput.Value() != "" {
			shortcuts = append(shortcuts, formatShortcut("x", "clear filter"))
		}
		footer = footerStyle.Width(m.width).Render(strings.Join(shortcuts, " | "))
	} else {
		browserMode := "terminal"
		if m.useGUIBrowser {
			browserMode = "gui"
		}
		shortcuts := []string{
			formatShortcut("esc/q", "back"),
			formatShortcut("j/k", "scroll"),
			formatShortcut("o", "link"),
			formatShortcut("r", "reply"),
			formatShortcut("b", browserMode),
			formatShortcut("?", "help"),
		}
		footer = footerStyle.Width(m.width).Render(strings.Join(shortcuts, " | "))
	}

	return fmt.Sprintf("%s%s\n%s", header, content, footer)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Fehler: %v", err)
		os.Exit(1)
	}
}
