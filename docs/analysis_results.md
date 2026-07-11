# App Audit & Implementation Report

This report documents the findings from the complete code audit, details the over-engineering cleanups applied, and tracks the implementation of TUI improvements for `hn-client`.

---

## 1. Over-Engineering Cleanup (Implemented)

We audited the codebase for over-engineering and speculative features. The following reductions were successfully made:

### A. Removed Speculative Markdown Parsers
*   **What was cut:** The functions `parseArticleFile` and `parseMDSearchFile` along with their fallback execution blocks in both the TUI and CLI.
*   **Why:** The app only documents and generates standard YAML frontmatter templates (via the TUI write feature). Having three separate Markdown/YAML parsers was speculative, undocumented, and dead code.
*   **Saved:** **99 lines of Go code.**

### B. Cleaned Local History Migration Logic
*   **What was cut:** The old file migration code in `loadHistory` that looked for a local `.hn-history.json` and migrated it to the user's home directory.
*   **Saved:** **9 lines of Go code.**

### C. Replaced Manual HTML Entity Replacements with Stdlib
*   **What was cut:** Manual replacements of HTML entity strings (`&#x27;`, `&quot;`, etc.) inside `cleanHTML`.
*   **Replacement:** Replaced with standard library `html.UnescapeString(t)`.
*   **Saved:** **4 lines of Go code.**

### D. Shrank Unused Fields in API Structs
*   **What was cut:** Unused fields `ID`, `Created`, `Karma`, and `About` in the `User` struct inside `internal/hnapi/api.go`.
*   **Saved:** **4 lines of Go code.**

---

## 2. Core UI/UX Improvements (Implemented)

The following usability improvements were integrated into the primary viewport layouts:

*   **Smart Comments Depth:** Bounded fetch recursion depth to **8 levels deep** (up from 3) for fuller Hacker News threads.
*   **Inline Quick-Reply vs. Vim Editor:** Configured footer text input `r` for inline replies, alongside external editor spawning `e` for complex messages.
*   **Responsive Title Wrapping & Dynamic Scrolling:** Swapped hardcoded line-truncations for Lip Gloss fluid wrapping, with height-aware list viewport offsets to prevent visual scroll drift.
*   **Feed Pagination:** Implemented `n`/`p` pagination (30 stories/page) showing `[Page X]` in the list header.
*   **Curated Multi-Themes:** Integrated an interactive theme toggle `t` for Dracula, Nord, Monokai, and Classic Orange themes.
*   **Fallback CLI Browser Warn:** Checks w3m/lynx installation before spawning a shell, logging warning headers instead of failing silently.

---

## 3. Advanced TUI Enhancements (Fully Implemented)

We have successfully implemented all five selected advanced features requested by the user:

### 1. Comment Link Extractor (`u`)
*   **How it works:** Pressing `u` in comments view extracts all links (`http://` / `https://`) from the story and loaded comments.
*   **UI:** Shows a clean list modal overlay. Users navigate links via `j`/`k` and press `Enter` to open, or `Esc` to close.

### 2. Thread Folding/Collapsing (`c`)
*   **How it works:** Pressing `c` on the selected comment folds it, hiding its children.
*   **UI:** Hides comments below it and appends a muted indicator: `[+ X replies hidden]`. Selection skips folded items.

### 3. Bookmarks / Später lesen (`s` / Category 7)
*   **How it works:** Pressing `s` on any story or detail view saves the story and comments locally to `~/.hn-bookmarks.json`.
*   **Offline Mode:** Select Category tab `Bookmarks` (Key `7`) to view bookmarked stories and read comments completely offline.

### 4. User Profile Popup (`U` / Shift+U)
*   **How it works:** Pressing `U` fetches the author's Hacker News profile data (Karma, Created date, About bio) from the API.
*   **UI:** Renders a gorgeous word-wrapped profile card popup overlay.

### 5. Reply Notification alerts (🔔)
*   **How it works:** Runs periodic API calls to check for replies on user's recent submissions.
*   **UI:** Displays `🔔 (X new, Ctrl+N to clear)` in the top header if new replies are found. Pressing `ctrl+n` clears notifications.
