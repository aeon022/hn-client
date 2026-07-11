package main

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
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
	Theme         string `json:"theme"`
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

var categories = []string{"top", "new", "best", "ask", "show", "mine", "bookmarks"}

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
	deleteConfirmActive    bool
	deleteConfirmStoryID   int
	commentInput           textinput.Model
	commentInputActive     bool
	page                   int
	theme                  string
	collapsedComments      map[int]bool
	detailComments         []comment
	commentCursor          int
	extractedLinks         []string
	linkSelectorActive     bool
	linkCursor             int
	userProfileActive      bool
	userProfileLoading     bool
	userProfileData        *hnapi.User
	userProfileErr         error
	newReplies             []int
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

	activeTheme := cfg.Theme
	if activeTheme == "" {
		activeTheme = "orange"
	}
	applyTheme(activeTheme)

	ci := textinput.New()
	ci.Placeholder = "Write a reply... (Enter to submit, Esc to cancel)"
	ci.CharLimit = 1000
	ci.Width = 50
	ci.TextStyle = lipgloss.NewStyle().Foreground(white)
	ci.PlaceholderStyle = lipgloss.NewStyle().Foreground(gray)

	return model{
		loading:            true,
		comments:           make(map[int][]comment),
		cursor:             0,
		state:              stateList,
		category:           "top",
		showHelp:           false,
		searchActive:       false,
		searchInput:        ti,
		usernameInput:      ui,
		passwordInput:      pi,
		username:           cfg.Username,
		showDead:           cfg.ShowDead,
		useGUIBrowser:      cfg.UseGUIBrowser,
		cookie:             cfg.Cookie,
		writeStep:          stepNone,
		history:            history,
		commentInput:       ci,
		commentInputActive: false,
		page:               0,
		theme:              activeTheme,
		collapsedComments:  make(map[int]bool),
	}
}

var linkRegex = regexp.MustCompile(`<a\s+href="([^"]+)"[^>]*>(.*?)</a>`)

// cleanHTML ist eine einfache Hilfe zum Säubern von HN Texten.
func cleanHTML(text string) string {
	// ponytail: use stdlib html.UnescapeString instead of manual replacements
	t := text
	t = strings.ReplaceAll(t, "<p>", "\n\n")
	t = html.UnescapeString(t)
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

	logError("openURL aufgerufen: URL=%q, useGUI=%t, gefundener Browser=%q", url, useGUI, browser)

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
				logError("ExecProcess beendet mit Fehler: %v", err)
				return errMsg{fmt.Errorf("Browser Fehler: %v", err)}
			}
			logError("ExecProcess w3m beendet mit nil error")
			return nil
		})
	} else {
		// Hintergrund-Start für 'open'
		return func() tea.Msg {
			err := c.Start()
			if err != nil {
				logError("GUI Browser start fehlgeschlagen: %v", err)
				return errMsg{fmt.Errorf("Browser Fehler: %v", err)}
			}
			logError("GUI Browser im Hintergrund gestartet")
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

type deleteResultMsg struct {
	err error
}

func doDeleteStory(cookie string, id int) tea.Cmd {
	return func() tea.Msg {
		client := &hnapi.Client{Cookie: cookie}
		err := client.DeleteItem(id)
		return deleteResultMsg{err: err}
	}
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

func fetchStories(category string, username string, cookie string, page int) tea.Cmd {
	return func() tea.Msg {
		if category == "bookmarks" {
			bms, _ := loadBookmarks()
			var stories []hnapi.Item
			for _, bm := range bms {
				stories = append(stories, bm.Story)
			}
			sort.Slice(stories, func(i, j int) bool {
				return bms[stories[i].ID].SavedAt > bms[stories[j].ID].SavedAt
			})

			const limit = 30
			startIndex := page * limit
			if startIndex >= len(stories) {
				return statusMsg(nil)
			}
			endIndex := startIndex + limit
			if endIndex > len(stories) {
				endIndex = len(stories)
			}
			return statusMsg(stories[startIndex:endIndex])
		}

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
		startIndex := page * limit
		if startIndex >= len(ids) {
			return statusMsg(nil)
		}
		endIndex := startIndex + limit
		if endIndex > len(ids) {
			endIndex = len(ids)
		}

		fetchIds := ids[startIndex:endIndex]

		client := &hnapi.Client{Cookie: cookie}
		stories := make([]hnapi.Item, len(fetchIds))
		var wg sync.WaitGroup
		var mu sync.Mutex
		for i := 0; i < len(fetchIds); i++ {
			wg.Add(1)
			go func(index, id int) {
				defer wg.Done()
				item, err := client.GetItemWithCookie(id)
				if err == nil {
					mu.Lock()
					stories[index] = item
					mu.Unlock()
				}
			}(i, fetchIds[i])
		}
		wg.Wait()

		var finalStories []hnapi.Item
		for _, s := range stories {
			if s.ID != 0 {
				if s.Deleted {
					continue
				}
				if category == "mine" && s.Type != "story" && s.Type != "poll" {
					continue
				}
				finalStories = append(finalStories, s)
			}
		}

		return statusMsg(finalStories)
	}
}

func fetchComments(storyID int, kids []int, indent int) []comment {
	// ponytail: expanded from 3 to 7 levels to support reading deeper HN discussions
	if indent > 7 {
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
	cmds := []tea.Cmd{fetchStories(m.category, m.username, m.cookie, m.page)}
	if m.username != "" {
		cmds = append(cmds, checkRepliesCmd(m.username), checkRepliesTick())
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := 3
		footerHeight := 3
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
			Theme:         m.theme,
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
			
			// Clear cookie if session is expired or unauthorized
			errStr := msg.err.Error()
			if strings.Contains(errStr, "nicht angemeldet") || 
			   strings.Contains(errStr, "Session abgelaufen") || 
			   strings.Contains(errStr, "erneut anmelden") {
				m.cookie = ""
				_ = saveConfig(config{
					Username:      m.username,
					ShowDead:      m.showDead,
					UseGUIBrowser: m.useGUIBrowser,
					Cookie:        "",
					Theme:         m.theme,
				})
			}
			
			m.updateViewport()
			return m, nil
		}
		m.loading = true
		if m.state == stateList {
			m.cursor = 0
			m.viewport.YOffset = 0
			m.updateViewport()
			return m, fetchStories(m.category, m.username, m.cookie, m.page)
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

	case deleteResultMsg:
		m.submitting = false
		m.deleteConfirmActive = false
		if msg.err != nil {
			logError("Löschen fehlgeschlagen: %v", msg.err)
			m.wizardError = msg.err.Error()
			m.updateViewport()
			return m, nil
		}
		m.loading = true
		m.cursor = 0
		m.viewport.YOffset = 0
		m.updateViewport()
		return m, fetchStories(m.category, m.username, m.cookie, m.page)

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

			parentID := story.ID
			if m.state == stateDetail && len(m.detailComments) > 0 && m.commentCursor < len(m.detailComments) {
				parentID = m.detailComments[m.commentCursor].item.ID
			}

			m.submitting = true
			m.updateViewport()
			return m, doSubmitComment(m.cookie, parentID, body)
		}

		m.writeStep = stepNone
		m.updateViewport()
		return m, nil

	case statusMsg:
		m.stories = msg
		m.loading = false
		m.err = nil
		m.cursor = 0
		m.updateViewport()

	case commentsMsg:
		m.comments[msg.storyID] = msg.comments
		m.loadingComments = false
		m.err = nil
		m.rebuildDetailComments()
		if m.state == stateDetail {
			m.updateViewport()
		}

	case userProfileMsg:
		m.userProfileLoading = false
		if msg.err != nil {
			m.userProfileErr = msg.err
		} else {
			m.userProfileData = &msg.user
		}
		m.updateViewport()
		return m, nil

	case bookmarkSavedMsg:
		if msg.err != nil {
			m.wizardError = fmt.Sprintf("Bookmark fehlgeschlagen: %v", msg.err)
		} else {
			m.wizardError = "Story & Kommentare offline gespeichert!"
		}
		m.updateViewport()
		return m, nil

	case repliesResultMsg:
		if msg.err == nil {
			m.newReplies = msg.newReplyIDs
		}
		m.updateViewport()
		return m, nil

	case checkRepliesTickMsg:
		if m.username != "" {
			return m, tea.Batch(checkRepliesCmd(m.username), checkRepliesTick())
		}
		return m, nil

	case errMsg:
		logError("Genereller App-Fehler: %v", msg.err)
		m.err = msg.err
		m.loading = false
		m.loadingComments = false
		m.updateViewport()
		return m, nil

	case tea.MouseMsg:
		if m.showHelp || m.searchActive || m.loginActive || m.commentInputActive || m.writeStep != stepNone || m.submitting || m.linkSelectorActive || m.userProfileActive {
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

		if m.linkSelectorActive {
			switch msg.String() {
			case "j", "down":
				if m.linkCursor < len(m.extractedLinks)-1 {
					m.linkCursor++
				}
			case "k", "up":
				if m.linkCursor > 0 {
					m.linkCursor--
				}
			case "enter":
				m.linkSelectorActive = false
				url := m.extractedLinks[m.linkCursor]
				return m, openURL(url, m.useGUIBrowser)
			case "esc", "q", "u":
				m.linkSelectorActive = false
			}
			return m, nil
		}

		if m.userProfileActive {
			switch msg.String() {
			case "esc", "q", "U", "enter":
				m.userProfileActive = false
			}
			return m, nil
		}

		if m.deleteConfirmActive {
			switch msg.String() {
			case "y", "Y":
				m.deleteConfirmActive = false
				m.submitting = true
				m.updateViewport()
				return m, doDeleteStory(m.cookie, m.deleteConfirmStoryID)
			case "n", "N", "esc":
				m.deleteConfirmActive = false
				m.updateViewport()
				return m, nil
			}
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

		if m.commentInputActive {
			switch msg.String() {
			case "esc":
				m.commentInputActive = false
				m.commentInput.SetValue("")
				m.updateViewport()
				return m, nil
			case "enter":
				text := strings.TrimSpace(m.commentInput.Value())
				m.commentInput.SetValue("")
				m.commentInputActive = false
				if text == "" {
					m.updateViewport()
					return m, nil
				}
				displayStories := m.getDisplayStories()
				if len(displayStories) == 0 {
					m.updateViewport()
					return m, nil
				}
				story := displayStories[m.cursor]
				m.submitting = true
				m.updateViewport()
				return m, doSubmitComment(m.cookie, story.ID, text)
			}
			m.commentInput, cmd = m.commentInput.Update(msg)
			return m, cmd
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
				_ = saveConfig(config{Username: m.username, ShowDead: m.showDead, UseGUIBrowser: m.useGUIBrowser, Cookie: m.cookie, Theme: m.theme})
				// If they are on the "mine" category, reload it!
				if m.category == "mine" {
					m.loading = true
					m.cursor = 0
					m.viewport.YOffset = 0
					m.updateViewport()
					return m, fetchStories(m.category, m.username, m.cookie, m.page)
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
				m.page = 0
				m.viewport.YOffset = 0
				m.searchInput.SetValue("") // Clear filter on category change
				return m, fetchStories(m.category, m.username, m.cookie, m.page)
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
				m.page = 0
				m.viewport.YOffset = 0
				m.searchInput.SetValue("") // Clear filter on category change
				return m, fetchStories(m.category, m.username, m.cookie, m.page)
			}
		case "1", "2", "3", "4", "5", "6", "7":
			if m.state == stateList {
				idx := int(msg.String()[0] - '1')
				if idx >= 0 && idx < len(categories) {
					m.category = categories[idx]
					m.loading = true
					m.cursor = 0
					m.page = 0
					m.viewport.YOffset = 0
					m.searchInput.SetValue("") // Clear filter on category change
					return m, fetchStories(m.category, m.username, m.cookie, m.page)
				}
			}
		case "n":
			// ponytail: fetch the next page of stories
			if m.state == stateList && !m.searchActive && !m.loginActive {
				m.page++
				m.loading = true
				m.cursor = 0
				m.viewport.YOffset = 0
				return m, fetchStories(m.category, m.username, m.cookie, m.page)
			}
		case "p":
			// ponytail: fetch the previous page of stories
			if m.state == stateList && !m.searchActive && !m.loginActive && m.page > 0 {
				m.page--
				m.loading = true
				m.cursor = 0
				m.viewport.YOffset = 0
				return m, fetchStories(m.category, m.username, m.cookie, m.page)
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
				_ = saveConfig(config{Username: m.username, ShowDead: m.showDead, UseGUIBrowser: m.useGUIBrowser, Cookie: m.cookie, Theme: m.theme})
				m.updateViewport()
				return m, nil
			}
		case "b":
			m.useGUIBrowser = !m.useGUIBrowser
			_ = saveConfig(config{Username: m.username, ShowDead: m.showDead, UseGUIBrowser: m.useGUIBrowser, Cookie: m.cookie, Theme: m.theme})
			m.updateViewport()
			return m, nil
		case "?":
			m.showHelp = true
			return m, nil
		case "d":
			if m.state == stateList && m.category == "mine" {
				displayStories := m.getDisplayStories()
				if len(displayStories) > 0 && m.cursor >= 0 && m.cursor < len(displayStories) {
					story := displayStories[m.cursor]
					if m.cookie == "" {
						m.wizardError = "Bitte logge dich zuerst ein."
						m.updateViewport()
						return m, nil
					}
					m.deleteConfirmActive = true
					m.deleteConfirmStoryID = story.ID
					m.updateViewport()
					return m, nil
				}
			}
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
		case "e":
			// ponytail: press 'e' to selectively write comments using the external Vim editor
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
			}
		case "r":
			// ponytail: press 'r' for a quick inline reply in the TUI footer
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
				m.commentInputActive = true
				m.commentInput.Focus()
				m.commentInput.SetValue("")
				m.updateViewport()
				return m, textinput.Blink
			} else {
				m.loading = true
				m.err = nil
				m.cursor = 0
				m.page = 0
				m.viewport.YOffset = 0
				return m, fetchStories(m.category, m.username, m.cookie, m.page)
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
		case "ctrl+n":
			if len(m.newReplies) > 0 {
				seen, _ := loadSeenReplies()
				for _, id := range m.newReplies {
					seen[id] = true
				}
				_ = saveSeenReplies(seen)
				m.newReplies = nil
				m.updateViewport()
			}
			return m, nil
		case "U":
			var username string
			if m.state == stateList {
				displayStories := m.getDisplayStories()
				if len(displayStories) > 0 && m.cursor < len(displayStories) {
					username = displayStories[m.cursor].By
				}
			} else if m.state == stateDetail {
				if len(m.detailComments) > 0 && m.commentCursor < len(m.detailComments) {
					username = m.detailComments[m.commentCursor].item.By
				}
			}
			username = strings.TrimSpace(username)
			if username != "" {
				m.userProfileActive = true
				m.userProfileLoading = true
				m.userProfileData = nil
				m.userProfileErr = nil
				m.updateViewport()
				return m, fetchUserProfileCmd(username)
			}
			return m, nil
		case "s":
			if m.state == stateList || m.state == stateDetail {
				displayStories := m.getDisplayStories()
				if len(displayStories) > 0 && m.cursor < len(displayStories) {
					story := displayStories[m.cursor]
					if m.state == stateDetail {
						bms, _ := loadBookmarks()
						bms[story.ID] = bookmark{
							Story:    story,
							Comments: m.comments[story.ID],
							SavedAt:  time.Now().Unix(),
						}
						_ = saveBookmarks(bms)
						m.wizardError = "Story & Kommentare offline gespeichert!"
						m.updateViewport()
					} else {
						m.wizardError = "Lade Kommentare für Offline-Speicherung..."
						m.updateViewport()
						return m, bookmarkStoryCmd(story)
					}
				}
			}
			return m, nil
		case "u":
			if m.state == stateDetail {
				if m.linkSelectorActive {
					m.linkSelectorActive = false
				} else {
					displayStories := m.getDisplayStories()
					if len(displayStories) > 0 {
						story := displayStories[m.cursor]
						var allText string
						if story.URL != "" {
							allText += story.URL + "\n"
						}
						allText += story.Text + "\n"
						for _, c := range m.detailComments {
							allText += c.item.Text + "\n"
						}
						m.extractedLinks = extractURLs(allText)
						if len(m.extractedLinks) > 0 {
							m.linkSelectorActive = true
							m.linkCursor = 0
						} else {
							m.wizardError = "Keine Links in Kommentaren gefunden!"
							m.updateViewport()
						}
					}
				}
				return m, nil
			}
		case "c":
			if m.state == stateDetail && len(m.detailComments) > 0 && m.commentCursor < len(m.detailComments) {
				curr := m.detailComments[m.commentCursor]
				m.collapsedComments[curr.item.ID] = !m.collapsedComments[curr.item.ID]
				m.rebuildDetailComments()
				m.updateViewport()
			}
			return m, nil
		case "up", "k":
			if m.state == stateList {
				if m.cursor > 0 {
					m.cursor--
					m.updateViewport()
				}
			} else if m.state == stateDetail {
				if len(m.detailComments) > 0 && m.commentCursor > 0 {
					m.commentCursor--
					m.updateViewport()
				}
			}
		case "down", "j":
			if m.state == stateList {
				displayStories := m.getDisplayStories()
				if m.cursor < len(displayStories)-1 {
					m.cursor++
					m.updateViewport()
				}
			} else if m.state == stateDetail {
				if len(m.detailComments) > 0 && m.commentCursor < len(m.detailComments)-1 {
					m.commentCursor++
					m.updateViewport()
				}
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
				
				m.commentCursor = 0
				if _, exists := m.comments[story.ID]; !exists {
					if m.category == "bookmarks" {
						bms, _ := loadBookmarks()
						if bm, ok := bms[story.ID]; ok {
							m.comments[story.ID] = bm.Comments
							m.loadingComments = false
							m.rebuildDetailComments()
							m.updateViewport()
						} else {
							m.loadingComments = true
							m.updateViewport()
							cmds = append(cmds, func() tea.Msg {
								c := fetchComments(story.ID, story.Kids, 0)
								return commentsMsg{storyID: story.ID, comments: c}
							})
						}
					} else {
						m.loadingComments = true
						m.updateViewport()
						cmds = append(cmds, func() tea.Msg {
							c := fetchComments(story.ID, story.Kids, 0)
							return commentsMsg{storyID: story.ID, comments: c}
						})
					}
				} else {
					m.rebuildDetailComments()
					m.updateViewport()
				}
			}
		case "o":
			displayStories := m.getDisplayStories()
			if len(displayStories) > 0 {
				story := displayStories[m.cursor]
				url := story.URL
				if url == "" {
					url = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID)
				}
				// ponytail: simple time-based debounce to prevent duplicate browser launches from key repeating.
				if time.Since(m.lastBrowserOpen) < 1500*time.Millisecond {
					return m, nil
				}
				m.lastBrowserOpen = time.Now()
				if !m.useGUIBrowser && !hasTerminalBrowser() {
					m.wizardError = "Terminal browser (w3m/lynx) not found! Falling back to GUI browser."
					m.updateViewport()
				}
				return m, openURL(url, m.useGUIBrowser)
			}
		case "t":
			// ponytail: toggle color theme
			if m.state == stateList && !m.searchActive && !m.loginActive {
				nextIdx := 0
				for i, th := range themes {
					if th.Name == m.theme {
						nextIdx = (i + 1) % len(themes)
						break
					}
				}
				m.theme = themes[nextIdx].Name
				applyTheme(m.theme)
				_ = saveConfig(config{
					Username:      m.username,
					ShowDead:      m.showDead,
					UseGUIBrowser: m.useGUIBrowser,
					Cookie:        m.cookie,
					Theme:         m.theme,
				})
				m.updateViewport()
				return m, nil
			}
		case "esc", "backspace":
			if m.err != nil {
				m.err = nil
				m.state = stateList
				m.updateViewport()
				return m, nil
			}
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
		if item.Deleted {
			continue
		}
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

		itemStrings := make([]string, len(displayStories))
		for i, item := range displayStories {
			titleText := item.Title
			timeStr := formatTime(item.Time)

			hasBeenRead := m.history[item.ID] > 0
			titlePrefix := ""
			if hasBeenRead {
				titlePrefix = "✓ "
			}

			// Styled metadata items
			var pts, author, commentsCount string
			if hasBeenRead && m.cursor != i {
				// Dimmed for read, unselected
				pts = lipgloss.NewStyle().Foreground(gray).Render(fmt.Sprintf("%d pts", item.Score))
				author = lipgloss.NewStyle().Foreground(gray).Render(item.By)
				commentsCount = lipgloss.NewStyle().Foreground(gray).Render(fmt.Sprintf("%d comments", item.Descendants))
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
					title = deadLabel + selectedTitleStyle.Copy().Width(m.width - 12).Render(titlePrefix + titleText)
				} else {
					title = selectedTitleStyle.Copy().Width(m.width - 6).Render(titlePrefix + titleText)
				}
				itemStr = selectedBoxStyle.Render(title + "\n" + metaText)
			} else {
				var title string
				if item.Dead {
					deadLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true).Render("[DEAD] ")
					if hasBeenRead {
						title = deadLabel + readTitleStyle.Copy().Width(m.width - 12).Render(titlePrefix + titleText)
					} else {
						title = deadLabel + unselectedTitleStyle.Copy().Width(m.width - 12).Render(titlePrefix + titleText)
					}
				} else {
					if hasBeenRead {
						title = readTitleStyle.Copy().Width(m.width - 6).Render(titlePrefix + titleText)
					} else {
						title = unselectedTitleStyle.Copy().Width(m.width - 6).Render(titleText)
					}
				}
				itemStr = unselectedBoxStyle.Render(title + "\n" + metaText)
			}
			itemStrings[i] = itemStr
		}

		s.Reset()
		for _, itemStr := range itemStrings {
			s.WriteString(itemStr + "\n\n")
		}
		content = s.String()

		// ponytail: dynamic scrolling logic based on actual rendered heights of items to support responsive title wrapping
		if len(displayStories) > 0 {
			itemTop := 0
			for i := 0; i < m.cursor; i++ {
				itemTop += strings.Count(itemStrings[i], "\n") + 2 // +2 for separating newlines
			}
			itemHeight := strings.Count(itemStrings[m.cursor], "\n") + 2
			itemBottom := itemTop + itemHeight

			if itemTop < m.viewport.YOffset {
				m.viewport.YOffset = itemTop
			} else if itemBottom >= m.viewport.YOffset+m.viewport.Height {
				m.viewport.YOffset = itemBottom - m.viewport.Height + 1
			}
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
				selectedID := -1
				if len(m.detailComments) > 0 && m.commentCursor < len(m.detailComments) {
					selectedID = m.detailComments[m.commentCursor].item.ID
				}
				var renderPos []commentRenderPos
				currentLine := strings.Count(s.String(), "\n")

				commentsStr := m.renderComments(comments, selectedID, &renderPos, &currentLine)
				s.WriteString(commentsStr)

				var selectedPos *commentRenderPos
				for _, rp := range renderPos {
					if rp.id == selectedID {
						selectedPos = &rp
						break
					}
				}

				if selectedPos != nil {
					itemTop := selectedPos.startLine
					itemBottom := selectedPos.startLine + selectedPos.height

					if itemTop < m.viewport.YOffset {
						m.viewport.YOffset = itemTop
					} else if itemBottom >= m.viewport.YOffset+m.viewport.Height {
						m.viewport.YOffset = itemBottom - m.viewport.Height + 1
					}
				}
			}
		}
		content = s.String()
	}

	m.viewport.SetContent(content)
}

type commentRenderPos struct {
	id        int
	startLine int
	height    int
}

func (m model) renderComments(comments []comment, selectedID int, renderPos *[]commentRenderPos, currentLine *int) string {
	var s strings.Builder
	for _, c := range comments {
		startLine := *currentLine
		indentSize := c.indent * 2
		indentStr := strings.Repeat(" ", indentSize)

		colorIndex := c.indent % len(commentColors)
		commentBorderColor := commentColors[colorIndex]

		isSelected := c.item.ID == selectedID
		if isSelected {
			commentBorderColor = orange
		}

		isNew := m.currentStoryLastViewed > 0 && c.item.Time > m.currentStoryLastViewed

		var author string
		authorText := c.item.By
		if isSelected {
			authorText = "▶ " + authorText
		}
		if isNew {
			newTag := lipgloss.NewStyle().Foreground(green).Bold(true).Render(" [NEU]")
			author = lipgloss.NewStyle().Foreground(commentBorderColor).Bold(true).Render(authorText) + newTag
		} else {
			author = lipgloss.NewStyle().Foreground(commentBorderColor).Bold(true).Render(authorText)
		}

		isCollapsed := m.collapsedComments[c.item.ID]
		if isCollapsed {
			childCount := countChildren(c)
			collapsedText := fmt.Sprintf(" [%d replies hidden]", childCount)
			author += lipgloss.NewStyle().Foreground(gray).Italic(true).Render(collapsedText)
		}

		authorLine := indentStr + author + "\n"
		s.WriteString(authorLine)
		*currentLine += strings.Count(authorLine, "\n")

		if !isCollapsed {
			text := cleanHTML(c.item.Text)
			commentWidth := m.width - indentSize - 10
			if commentWidth < 20 {
				commentWidth = 20
			}

			style := commentStyle.Copy().
				BorderForeground(commentBorderColor).
				Width(commentWidth)
			if isSelected {
				style = style.Border(lipgloss.DoubleBorder(), false, false, false, true)
			}

			styledText := style.Render(text) + "\n\n"
			s.WriteString(indentStr + styledText)
			*currentLine += strings.Count(indentStr + styledText, "\n")

			*renderPos = append(*renderPos, commentRenderPos{
				id:        c.item.ID,
				startLine: startLine,
				height:    *currentLine - startLine,
			})

			s.WriteString(m.renderComments(c.children, selectedID, renderPos, currentLine))
		} else {
			s.WriteString("\n")
			*currentLine += 1

			*renderPos = append(*renderPos, commentRenderPos{
				id:        c.item.ID,
				startLine: startLine,
				height:    *currentLine - startLine,
			})
		}
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
	table.WriteString(shortcut("j / k / ↓ / ↑", "Navigate Stories") + "\n")
	table.WriteString(shortcut("Mouse Wheel", "Scroll / Move Cursor") + "\n")
	table.WriteString(shortcut("Tab / Shift+Tab", "Switch Category") + "\n")
	table.WriteString(shortcut("p / n", "Previous / Next Page of Stories") + "\n")
	table.WriteString(shortcut("1 - 7", "Direct Category Selection (7: Bookmarks)") + "\n")
	table.WriteString(shortcut("l", "Set HN Username / Login") + "\n")
	table.WriteString(shortcut("h", "Toggle Show Dead/Flagged posts") + "\n")
	table.WriteString(shortcut("b", "Toggle Browser Mode (GUI vs Terminal)") + "\n")
	table.WriteString(shortcut("Enter", "Open Details & Comments") + "\n")
	table.WriteString(shortcut("r", "Reload Feed") + "\n")
	table.WriteString(shortcut("o", "Open Original Link") + "\n")
	table.WriteString(shortcut("s", "Bookmark story + comments offline") + "\n")
	table.WriteString(shortcut("U", "View author profile card") + "\n")
	table.WriteString(shortcut("w", "Write Submission (TUI)") + "\n")
	table.WriteString(shortcut("d", "Delete Selected Story (under Mine)") + "\n")

	table.WriteString(section("Comments View") + "\n")
	table.WriteString(shortcut("j / k / ↓ / ↑", "Navigate comments") + "\n")
	table.WriteString(shortcut("Mouse Wheel", "Scroll viewport") + "\n")
	table.WriteString(shortcut("Esc / q", "Back to Story List") + "\n")
	table.WriteString(shortcut("o", "Open Original Link") + "\n")
	table.WriteString(shortcut("c", "Fold / unfold selected thread") + "\n")
	table.WriteString(shortcut("r", "Quick Reply to selection (Inline)") + "\n")
	table.WriteString(shortcut("e", "Reply to selection via Vim (External)") + "\n")
	table.WriteString(shortcut("u", "Extract and select comment links") + "\n")
	table.WriteString(shortcut("s", "Bookmark story + comments offline") + "\n")
	table.WriteString(shortcut("U", "View comment author profile") + "\n")
	table.WriteString(shortcut("b", "Toggle Browser Mode (GUI vs Terminal)") + "\n")

	table.WriteString(section("General") + "\n")
	table.WriteString(shortcut("t", "Toggle Color Theme") + "\n")
	table.WriteString(shortcut("ctrl+n", "Clear new reply alerts") + "\n")
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
		if m.page > 0 {
			headerText += lipgloss.NewStyle().Foreground(gray).Render(fmt.Sprintf("  [Page %d]", m.page+1))
		}
		
		// If a filter is active, append it to the header
		if m.searchInput.Value() != "" {
			headerText += lipgloss.NewStyle().Foreground(gray).Italic(true).Render(fmt.Sprintf("  (Filter: %q)", m.searchInput.Value()))
		}

		if len(m.newReplies) > 0 {
			headerText += " " + lipgloss.NewStyle().Foreground(green).Bold(true).Render(fmt.Sprintf("🔔 (%d new, Ctrl+N to clear)", len(m.newReplies)))
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
	} else if m.linkSelectorActive {
		content = m.renderLinkSelector()
	} else if m.userProfileActive {
		content = m.renderUserProfile()
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
	} else if m.deleteConfirmActive {
		storyTitle := "this story"
		displayStories := m.getDisplayStories()
		for _, item := range displayStories {
			if item.ID == m.deleteConfirmStoryID {
				storyTitle = truncateString(item.Title, m.width-30)
				break
			}
		}
		label := lipgloss.NewStyle().Foreground(white).Bold(true).Render(" ⚠️ Delete post? ")
		confirmPrompt := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8585")).Render(fmt.Sprintf("%q (y/n)", storyTitle))
		footer = footerStyle.Width(m.width).Render(label + confirmPrompt)
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
	} else if m.commentInputActive {
		replyLabel := lipgloss.NewStyle().Foreground(white).Bold(true).Render(" 💬 Quick Reply: ")
		footer = footerStyle.Width(m.width).Render(
			replyLabel + m.commentInput.View() + lipgloss.NewStyle().Foreground(lightGray).Render("  (Esc: Cancel / Enter: Submit)"),
		)
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
		line1Shortcuts := []string{
			formatShortcut("q", "quit"),
			formatShortcut("tab", "feed"),
			formatShortcut("p/n", "prev/next page"),
			formatShortcut("j/k", "nav"),
			formatShortcut("enter", "view"),
			formatShortcut("o", "link"),
		}
		line2Shortcuts := []string{
			formatShortcut("r", "reload"),
			formatShortcut("w", "post"),
			formatShortcut("l", "login"),
			formatShortcut("t", "theme"),
			formatShortcut("s", "bookmark"),
			formatShortcut("U", "profile"),
		}
		if m.category == "mine" {
			line2Shortcuts = append(line2Shortcuts, formatShortcut("d", "delete"))
		}
		line2Shortcuts = append(line2Shortcuts, []string{
			formatShortcut("/", "search"),
			formatShortcut("b", browserMode),
			formatShortcut("?", "help"),
		}...)
		if m.searchInput.Value() != "" {
			line2Shortcuts = append(line2Shortcuts, formatShortcut("x", "clear filter"))
		}
		
		line1 := strings.Join(line1Shortcuts, " | ")
		line2 := strings.Join(line2Shortcuts, " | ")
		footer = footerStyle.Width(m.width).Render(line1 + "\n " + line2)
	} else {
		browserMode := "terminal"
		if m.useGUIBrowser {
			browserMode = "gui"
		}
		line1Shortcuts := []string{
			formatShortcut("esc/q", "back"),
			formatShortcut("j/k", "nav comment"),
			formatShortcut("o", "link"),
			formatShortcut("c", "fold"),
		}
		line2Shortcuts := []string{
			formatShortcut("r", "reply"),
			formatShortcut("e", "vim reply"),
			formatShortcut("u", "links list"),
			formatShortcut("s", "bookmark"),
			formatShortcut("U", "profile"),
			formatShortcut("b", browserMode),
			formatShortcut("?", "help"),
		}
		
		line1 := strings.Join(line1Shortcuts, " | ")
		line2 := strings.Join(line2Shortcuts, " | ")
		footer = footerStyle.Width(m.width).Render(line1 + "\n " + line2)
	}

	return fmt.Sprintf("%s%s\n%s", header, content, footer)
}

func main() {
	if len(os.Args) > 1 {
		if os.Args[1] == "publish" || os.Args[1] == "post" {
			if len(os.Args) < 3 {
				fmt.Println("Usage: hn-client publish <file.md>")
				os.Exit(1)
			}
			filePath := os.Args[2]
			data, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Error reading file: %v\n", err)
				os.Exit(1)
			}

			cfg, err := loadConfig()
			if err != nil || cfg.Cookie == "" {
				fmt.Println("Error: Session cookie not found. Please log in using the 'hn-client' TUI first.")
				os.Exit(1)
			}

			content := string(data)
			// Try standard frontmatter parse first
			title, urlStr, body, _ := parseStoryFile(content)
			


			title = strings.TrimSpace(title)
			urlStr = strings.TrimSpace(urlStr)
			body = markdownToHN(body)

			if title == "" {
				fmt.Println("Error: Could not parse Title from markdown file.")
				os.Exit(1)
			}

			if urlStr == "" && body == "" {
				fmt.Println("Error: Could not parse URL or Text/Comment from markdown file.")
				os.Exit(1)
			}

			fmt.Printf("Submitting to Hacker News...\n")
			fmt.Printf("Title: %s\n", title)
			if urlStr != "" {
				fmt.Printf("URL:   %s\n", urlStr)
			} else {
				fmt.Printf("Text:  %d characters\n", len(body))
			}

			client := &hnapi.Client{Cookie: cfg.Cookie}
			err = client.SubmitStory(title, urlStr, body)
			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "nicht angemeldet") || 
				   strings.Contains(errStr, "Session abgelaufen") || 
				   strings.Contains(errStr, "erneut anmelden") {
					cfg.Cookie = ""
					_ = saveConfig(cfg)
					fmt.Println("Error: Hacker News session has expired. Your session cookie was cleared.")
					fmt.Println("Please run the 'hn-client' TUI first, press 'l' and log in again to refresh your session.")
				} else {
					fmt.Printf("Submission failed: %v\n", err)
				}
				os.Exit(1)
			}

			fmt.Println("Successfully posted to Hacker News!")
			os.Exit(0)
		}
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Fehler: %v", err)
		os.Exit(1)
	}
}

// ponytail: check if a terminal browser is installed and available
func hasTerminalBrowser() bool {
	browsers := []string{"w3m", "lynx", "links"}
	for _, b := range browsers {
		if _, err := exec.LookPath(b); err == nil {
			return true
		}
	}
	commonPaths := []string{
		"/opt/homebrew/bin/w3m", "/usr/local/bin/w3m",
		"/opt/homebrew/bin/lynx", "/usr/local/bin/lynx", "/usr/bin/lynx",
		"/opt/homebrew/bin/links",
	}
	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

type theme struct {
	Name      string
	Orange    string
	Gray      string
	LightGray string
	White     string
	DarkGray  string
	Cyan      string
	Blue      string
	Purple    string
	Green     string
}

var themes = []theme{
	{
		Name:      "orange",
		Orange:    "#FF6600",
		Gray:      "#828282",
		LightGray: "#C0C0C0",
		White:     "#FFFFFF",
		DarkGray:  "#333333",
		Cyan:      "#00F0FF",
		Blue:      "#85A5FF",
		Purple:    "#FF85FF",
		Green:     "#ADFF2F",
	},
	{
		Name:      "dracula",
		Orange:    "#BD93F9",
		Gray:      "#6272A4",
		LightGray: "#44475A",
		White:     "#F8F8F2",
		DarkGray:  "#282A36",
		Cyan:      "#8BE9FD",
		Blue:      "#FF79C6",
		Purple:    "#FFB86C",
		Green:     "#50FA7B",
	},
	{
		Name:      "nord",
		Orange:    "#88C0D0",
		Gray:      "#4C566A",
		LightGray: "#434C5E",
		White:     "#ECEFF4",
		DarkGray:  "#2E3440",
		Cyan:      "#8FBCBB",
		Blue:      "#81A1C1",
		Purple:    "#B48EAD",
		Green:     "#A3BE8C",
	},
	{
		Name:      "monokai",
		Orange:    "#F92672",
		Gray:      "#75715E",
		LightGray: "#49483E",
		White:     "#F8F8F2",
		DarkGray:  "#272822",
		Cyan:      "#66D9EF",
		Blue:      "#AE81FF",
		Purple:    "#FD971F",
		Green:     "#A6E22E",
	},
}

func applyTheme(themeName string) {
	t := themes[0]
	for _, th := range themes {
		if th.Name == themeName {
			t = th
			break
		}
	}

	orange = lipgloss.Color(t.Orange)
	gray = lipgloss.Color(t.Gray)
	lightGray = lipgloss.Color(t.LightGray)
	white = lipgloss.Color(t.White)
	darkGray = lipgloss.Color(t.DarkGray)
	cyan = lipgloss.Color(t.Cyan)
	blue = lipgloss.Color(t.Blue)
	purple = lipgloss.Color(t.Purple)
	green = lipgloss.Color(t.Green)

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
}

type bookmark struct {
	Story    hnapi.Item `json:"story"`
	Comments []comment  `json:"comments"`
	SavedAt  int64      `json:"saved_at"`
}

func loadBookmarks() (map[int]bookmark, error) {
	path := getBookmarksPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[int]bookmark), nil
	}
	var bms map[int]bookmark
	if err := json.Unmarshal(data, &bms); err != nil {
		return make(map[int]bookmark), nil
	}
	return bms, nil
}

func saveBookmarks(bms map[int]bookmark) error {
	data, err := json.Marshal(bms)
	if err != nil {
		return err
	}
	return os.WriteFile(getBookmarksPath(), data, 0600)
}

func getBookmarksPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".hn-bookmarks.json"
	}
	return home + "/.hn-bookmarks.json"
}

func loadSeenReplies() (map[int]bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return make(map[int]bool), nil
	}
	path := home + "/.hn-seen-replies.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[int]bool), nil
	}
	var seen []int
	if err := json.Unmarshal(data, &seen); err != nil {
		return make(map[int]bool), nil
	}
	m := make(map[int]bool)
	for _, id := range seen {
		m[id] = true
	}
	return m, nil
}

func saveSeenReplies(seen map[int]bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := home + "/.hn-seen-replies.json"
	var list []int
	for id := range seen {
		list = append(list, id)
	}
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

type userProfileMsg struct {
	user hnapi.User
	err  error
}

func fetchUserProfileCmd(username string) tea.Cmd {
	return func() tea.Msg {
		user, err := hnapi.GetUser(username)
		return userProfileMsg{user: user, err: err}
	}
}

type repliesResultMsg struct {
	newReplyIDs []int
	err         error
}

type checkRepliesTickMsg struct{}

func checkRepliesTick() tea.Cmd {
	return tea.Tick(60*time.Second, func(t time.Time) tea.Msg {
		return checkRepliesTickMsg{}
	})
}

func checkRepliesCmd(username string) tea.Cmd {
	return func() tea.Msg {
		if username == "" {
			return nil
		}
		subs, err := hnapi.GetUserSubmissions(username)
		if err != nil {
			return repliesResultMsg{err: err}
		}
		if len(subs) == 0 {
			return repliesResultMsg{}
		}

		limit := 10
		if len(subs) < limit {
			limit = len(subs)
		}
		recent := subs[:limit]

		type itemResult struct {
			item hnapi.Item
			err  error
		}
		ch := make(chan itemResult, limit)
		for _, id := range recent {
			go func(itemID int) {
				item, err := hnapi.GetItem(itemID)
				ch <- itemResult{item: item, err: err}
			}(id)
		}

		var items []hnapi.Item
		for i := 0; i < limit; i++ {
			res := <-ch
			if res.err == nil {
				items = append(items, res.item)
			}
		}

		seen, _ := loadSeenReplies()
		var uncheckedKids []int
		for _, item := range items {
			for _, kidID := range item.Kids {
				if !seen[kidID] {
					uncheckedKids = append(uncheckedKids, kidID)
				}
			}
		}

		var newReplies []int
		if len(uncheckedKids) > 0 {
			if len(uncheckedKids) > 15 {
				uncheckedKids = uncheckedKids[:15]
			}
			chKids := make(chan itemResult, len(uncheckedKids))
			for _, id := range uncheckedKids {
				go func(kidID int) {
					kid, err := hnapi.GetItem(kidID)
					chKids <- itemResult{item: kid, err: err}
				}(id)
			}

			for i := 0; i < len(uncheckedKids); i++ {
				res := <-chKids
				if res.err == nil && res.item.By != username {
					newReplies = append(newReplies, res.item.ID)
				}
			}
		}

		return repliesResultMsg{newReplyIDs: newReplies}
	}
}

type bookmarkSavedMsg struct {
	storyID int
	err     error
}

func bookmarkStoryCmd(story hnapi.Item) tea.Cmd {
	return func() tea.Msg {
		c := fetchComments(story.ID, story.Kids, 0)
		bms, _ := loadBookmarks()
		bms[story.ID] = bookmark{
			Story:    story,
			Comments: c,
			SavedAt:  time.Now().Unix(),
		}
		err := saveBookmarks(bms)
		return bookmarkSavedMsg{storyID: story.ID, err: err}
	}
}

func countChildren(c comment) int {
	count := len(c.children)
	for _, child := range c.children {
		count += countChildren(child)
	}
	return count
}

func flattenComments(comments []comment, collapsed map[int]bool) []comment {
	var flat []comment
	var traverse func([]comment)
	traverse = func(list []comment) {
		for _, c := range list {
			flat = append(flat, c)
			if !collapsed[c.item.ID] {
				traverse(c.children)
			}
		}
	}
	traverse(comments)
	return flat
}

func (m *model) rebuildDetailComments() {
	storyID := 0
	displayStories := m.getDisplayStories()
	if len(displayStories) > 0 && m.cursor < len(displayStories) {
		storyID = displayStories[m.cursor].ID
	}
	if storyID > 0 {
		m.detailComments = flattenComments(m.comments[storyID], m.collapsedComments)
		if m.commentCursor >= len(m.detailComments) {
			if len(m.detailComments) > 0 {
				m.commentCursor = len(m.detailComments) - 1
			} else {
				m.commentCursor = 0
			}
		}
	} else {
		m.detailComments = nil
		m.commentCursor = 0
	}
}

func extractURLs(text string) []string {
	re := regexp.MustCompile(`https?://[^\s"<>]+`)
	matches := re.FindAllString(text, -1)

	seen := make(map[string]bool)
	var list []string
	for _, m := range matches {
		m = strings.TrimRight(m, ".,;()[]{}!?")
		if !seen[m] {
			seen[m] = true
			list = append(list, m)
		}
	}
	return list
}

func (m model) renderLinkSelector() string {
	contentWidth := 64
	title := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Foreground(white).
		Background(orange).
		Bold(true).
		Padding(0, 2).
		Render("EXTRACTED LINKS")

	var s strings.Builder
	s.WriteString("\n")
	for i, link := range m.extractedLinks {
		marker := "  "
		style := lipgloss.NewStyle().Foreground(white)
		if i == m.linkCursor {
			marker = "▶ "
			style = lipgloss.NewStyle().Foreground(orange).Bold(true)
		}

		displayLink := link
		if len(displayLink) > contentWidth-6 {
			displayLink = displayLink[:contentWidth-9] + "..."
		}
		s.WriteString("  " + marker + style.Render(displayLink) + "\n")
	}
	s.WriteString("\n" + lipgloss.NewStyle().Foreground(gray).Render("  Use j/k to navigate, Enter to open, Esc to close") + "\n")

	modalContent := lipgloss.JoinVertical(lipgloss.Left,
		title,
		s.String(),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(orange).
		Padding(1, 2).
		Render(modalContent)
}

func (m model) renderUserProfile() string {
	contentWidth := 60
	title := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Foreground(white).
		Background(orange).
		Bold(true).
		Padding(0, 2).
		Render("USER PROFILE")

	var s strings.Builder
	s.WriteString("\n")
	if m.userProfileLoading {
		s.WriteString("  ⌛ Loading profile... Please wait.\n\n")
	} else if m.userProfileErr != nil {
		s.WriteString("  ❌ Error loading profile:\n")
		s.WriteString("  " + m.userProfileErr.Error() + "\n\n")
	} else if m.userProfileData != nil {
		u := m.userProfileData
		createdTime := time.Unix(u.Created, 0).Format("2006-01-02")

		s.WriteString("  " + lipgloss.NewStyle().Foreground(orange).Bold(true).Render("Username:") + " " + u.ID + "\n")
		s.WriteString("  " + lipgloss.NewStyle().Foreground(orange).Bold(true).Render("Created:") + "  " + createdTime + "\n")
		s.WriteString("  " + lipgloss.NewStyle().Foreground(orange).Bold(true).Render("Karma:") + "    " + fmt.Sprintf("%d", u.Karma) + "\n")
		s.WriteString("\n")

		if u.About != "" {
			s.WriteString("  " + lipgloss.NewStyle().Foreground(orange).Bold(true).Render("About:") + "\n")
			aboutText := cleanHTML(u.About)
			wrappedAbout := lipgloss.NewStyle().Width(contentWidth - 6).Render(aboutText)
			s.WriteString(wrappedAbout + "\n\n")
		} else {
			s.WriteString("  No description available.\n\n")
		}
	}
	s.WriteString(lipgloss.NewStyle().Foreground(gray).Render("  Press Esc / Enter / Q to close") + "\n")

	modalContent := lipgloss.JoinVertical(lipgloss.Left,
		title,
		s.String(),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(orange).
		Padding(1, 2).
		Render(modalContent)
}
