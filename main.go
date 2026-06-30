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

var categories = []string{"top", "new", "best", "ask", "show"}

type state int

const (
	stateList state = iota
	stateDetail
)

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

	history, _ := loadHistory()

	return model{
		loading:      true,
		comments:     make(map[int][]comment),
		cursor:       0,
		state:        stateList,
		category:     "top",
		showHelp:     false,
		searchActive: false,
		searchInput:  ti,
		history:      history,
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

func openURL(url string) tea.Cmd {
	// 1. Browser suchen
	browsers := []string{
		"w3m", "/opt/homebrew/bin/w3m", "/usr/local/bin/w3m",
		"lynx", "/opt/homebrew/bin/lynx", "/usr/local/bin/lynx", "/usr/bin/lynx",
		"links", "/opt/homebrew/bin/links",
	}

	var browser string
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

func fetchStories(category string) tea.Cmd {
	return func() tea.Msg {
		ids, err := hnapi.GetStories(category)
		if err != nil {
			return errMsg{err}
		}
		const limit = 20
		stories := make([]hnapi.Item, limit)
		var wg sync.WaitGroup
		var mu sync.Mutex
		for i := 0; i < limit && i < len(ids); i++ {
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
				finalStories = append(finalStories, s)
			}
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
	return fetchStories(m.category)
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
		m.err = msg.err
		m.loading = false
		m.loadingComments = false
		m.updateViewport()
		return m, nil

	case tea.MouseMsg:
		if m.showHelp || m.searchActive {
			return m, nil
		}
		if m.state == stateList {
			switch msg.Button {
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
		if m.showHelp {
			switch msg.String() {
			case "?", "esc", "q", "enter", "space":
				m.showHelp = false
			}
			return m, nil
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
				return m, fetchStories(m.category)
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
				return m, fetchStories(m.category)
			}
		case "1", "2", "3", "4", "5":
			if m.state == stateList {
				idx := int(msg.String()[0] - '1')
				if idx >= 0 && idx < len(categories) {
					m.category = categories[idx]
					m.loading = true
					m.cursor = 0
					m.viewport.YOffset = 0
					m.searchInput.SetValue("") // Clear filter on category change
					return m, fetchStories(m.category)
				}
			}
		case "?":
			m.showHelp = true
			return m, nil
		case "w":
			if m.state == stateList {
				return m, openURL("https://news.ycombinator.com/login?goto=submit")
			}
		case "r":
			if m.state == stateDetail {
				displayStories := m.getDisplayStories()
				if len(displayStories) == 0 {
					return m, nil
				}
				curr := displayStories[m.cursor]
				return m, openURL(fmt.Sprintf("https://news.ycombinator.com/login?goto=item%%3Fid%%3D%d", curr.ID))
			} else {
				m.loading = true
				m.cursor = 0
				m.viewport.YOffset = 0
				return m, fetchStories(m.category)
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
					return m, openURL(story.URL)
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
	if m.searchInput.Value() == "" {
		return m.stories
	}
	var filtered []hnapi.Item
	query := strings.ToLower(m.searchInput.Value())
	for _, item := range m.stories {
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
				title := selectedTitleStyle.Render(titlePrefix + titleText)
				itemStr = selectedBoxStyle.Render(title + "\n" + metaText)
			} else {
				var title string
				if hasBeenRead {
					title = readTitleStyle.Render(titlePrefix + titleText)
				} else {
					title = unselectedTitleStyle.Render(titleText)
				}
				itemStr = unselectedBoxStyle.Render(title + "\n" + metaText)
			}
			s.WriteString(itemStr + "\n\n")
		}
		content = s.String()

		// Scrolling-Logik für die Story-Liste:
		// Jedes Item belegt genau 3 Zeilen im Viewport. Wir passen YOffset an,
		// damit das aktuell ausgewählte Element immer sichtbar bleibt.
		itemTop := m.cursor * 3
		itemBottom := m.cursor * 3 + 2
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
		pts := lipgloss.NewStyle().Foreground(cyan).Render(fmt.Sprintf("%d pts", curr.Score))
		author := lipgloss.NewStyle().Foreground(orange).Render(curr.By)
		timeStr := formatTime(curr.Time)
		commentsCount := lipgloss.NewStyle().Foreground(blue).Render(fmt.Sprintf("%d comments", curr.Descendants))
		meta := fmt.Sprintf("%s · by %s · %s · %s", pts, author, timeStr, commentsCount)
		
		s.WriteString(title + "\n")
		s.WriteString(meta + "\n")
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
	table.WriteString(shortcut("1 - 5", "Direct Feed Selection") + "\n")
	table.WriteString(shortcut("Enter", "Open Details & Comments") + "\n")
	table.WriteString(shortcut("r", "Reload Feed") + "\n")
	table.WriteString(shortcut("o", "Open Original Link") + "\n")
	table.WriteString(shortcut("w", "Write Submission (Browser)") + "\n")

	table.WriteString(section("Comments View") + "\n")
	table.WriteString(shortcut("j / k / ↓ / ↑", "Scroll") + "\n")
	table.WriteString(shortcut("Mouse Wheel", "Scroll") + "\n")
	table.WriteString(shortcut("Esc / q", "Back to Story List") + "\n")
	table.WriteString(shortcut("o", "Open Original Link") + "\n")
	table.WriteString(shortcut("r", "Reply to Thread (Browser)") + "\n")

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
	} else {
		// Header in reader mode
		headerText = titleStyle.Render(" HACKER NEWS ") + "  │  " + lipgloss.NewStyle().Foreground(orange).Bold(true).Render("READER MODE (Comments)")
	}

	// Divider across the entire terminal width
	divider := lipgloss.NewStyle().Foreground(darkGray).Render(strings.Repeat("─", m.width))
	header := fmt.Sprintf("%s\n%s\n", headerText, divider)

	if m.loading {
		return headerText + "\n" + divider + "\n\n  " + lipgloss.NewStyle().Foreground(orange).Render("⌛ Loading...")
	}

	var content string
	if m.showHelp {
		content = m.renderHelp()
	} else {
		content = m.viewport.View()
	}

	var footer string
	if m.searchActive {
		searchLabel := lipgloss.NewStyle().Foreground(white).Bold(true).Render(" 🔍 Search: ")
		footer = footerStyle.Width(m.width).Render(
			searchLabel + m.searchInput.View() + lipgloss.NewStyle().Foreground(lightGray).Render("  (Esc: Cancel / Enter: Apply)"),
		)
	} else if m.state == stateList {
		shortcuts := []string{
			formatShortcut("q", "quit"),
			formatShortcut("tab", "feed"),
			formatShortcut("r", "reload"),
			formatShortcut("j/k", "nav"),
			formatShortcut("enter", "view"),
			formatShortcut("o", "link"),
			formatShortcut("w", "post"),
			formatShortcut("/", "search"),
			formatShortcut("?", "help"),
		}
		if m.searchInput.Value() != "" {
			shortcuts = append(shortcuts, formatShortcut("x", "clear filter"))
		}
		footer = footerStyle.Width(m.width).Render(strings.Join(shortcuts, " | "))
	} else {
		shortcuts := []string{
			formatShortcut("esc/q", "back"),
			formatShortcut("j/k", "scroll"),
			formatShortcut("o", "link"),
			formatShortcut("r", "reply"),
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
