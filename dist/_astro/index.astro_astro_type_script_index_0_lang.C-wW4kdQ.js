const k={top:[{title:"Show HN: hn-client — Hacker News directly in the terminal",pts:256,author:"gweiher",time:"2h ago",comments:32,read:!0,commentsList:[{author:"linus_t",text:"This is exactly what I've been waiting for. No web bloat, just pure Vim keybindings. The w3m integration for comments/replies works beautifully.",depth:0,time:"1h ago"},{author:"gweiher",text:"Thanks! I built it precisely to scratch my own itch. Let me know if you run into any issues.",depth:1,time:"45m ago"},{author:"go_dev",text:"Very nice codebase structure. Bubble Tea is perfect for this.",depth:1,time:"30m ago"},{author:"hacker99",text:"Are there any plans for a Homebrew formula?",depth:2,time:"15m ago"}]},{title:"DuckDB Internals: Why Is DuckDB Fast? (Part 1)",pts:187,author:"greybeam",time:"11h ago",comments:88,read:!1,commentsList:[{author:"db_junkie",text:"Vectorized execution and lean memory management. The architectural breakdown is top tier.",depth:0,time:"9h ago"},{author:"greybeam",text:"Glad you liked it! Part 2 will cover the storage engine details.",depth:1,time:"8h ago"}]},{title:"Project Valhalla, Explained: How a Decade of Work Arrives in JDK 28",pts:116,author:"skogstokig",time:"17h ago",comments:83,read:!1,commentsList:[{author:"java_wizard",text:"Value types and primitive class types are going to revolutionize memory efficiency in Java.",depth:0,time:"15h ago"}]}],new:[{title:"Show HN: WebGL canvas particles layout",pts:12,author:"creative_coder",time:"5m ago",comments:3,read:!1,commentsList:[{author:"canvas_fan",text:"Nice animations! FPS is very stable even on my older machine.",depth:0,time:"3m ago"}]},{title:"My experience writing Go code without pointers",pts:45,author:"go_pioneer",time:"48m ago",comments:19,read:!1,commentsList:[{author:"pointer_guru",text:"Passing values by copy can have overhead, but it eliminates escape analysis issues. Interesting take.",depth:0,time:"30m ago"}]},{title:"Why I still use w3m in 2026",pts:98,author:"term_guru",time:"1h ago",comments:41,read:!0,commentsList:[{author:"vi_keys",text:"Vim-like keys and text-only rendering makes reading documentation so fast.",depth:0,time:"45m ago"}]}],best:[{title:"Why Is Apple's Silicon So Fast? (2020)",pts:894,author:"tech_mind",time:"2d ago",comments:312,read:!1,commentsList:[{author:"m1_max",text:"The unified memory architecture and wide decoder design are the key secrets.",depth:0,time:"1d ago"}]},{title:"Show HN: Are You in the Weights?",pts:402,author:"turtlesoup",time:"18h ago",comments:226,read:!1,commentsList:[{author:"llm_researcher",text:"Fascinating visualization of neural net weights. Highly recommend viewing.",depth:0,time:"12h ago"}]},{title:"What I learned from reading the UNIX v6 source code",pts:512,author:"sys_eng",time:"1d ago",comments:92,read:!0,commentsList:[{author:"retro_code",text:"The simplicity of the scheduler code in v6 is a masterclass in software engineering.",depth:0,time:"20h ago"}]}],ask:[{title:"Ask HN: What is your favorite terminal tool?",pts:143,author:"shell_lover",time:"5h ago",comments:98,read:!1,commentsList:[{author:"fzf_enjoyer",text:"fzf is definitely life-changing. Combined with fd and ripgrep, it makes file searching instant.",depth:0,time:"4h ago"}]},{title:"Ask HN: Why did you switch from Python to Go?",pts:210,author:"py_convert",time:"12h ago",comments:154,read:!1,commentsList:[{author:"speedy_go",text:"Static binaries, compilation speed, and stellar concurrency primitives did it for me.",depth:0,time:"11h ago"}]},{title:"Ask HN: Best resources to learn Bubble Tea framework?",pts:87,author:"tui_enthusiast",time:"1d ago",comments:22,read:!0,commentsList:[{author:"bubble_wrap",text:"The examples folder in the charmbracelet/bubbletea repo is by far the best documentation.",depth:0,time:"18h ago"}]}],show:[{title:"Show HN: hn-client — Hacker News directly in the terminal",pts:256,author:"gweiher",time:"2h ago",comments:32,read:!0,commentsList:[{author:"linus_t",text:"This is exactly what I've been waiting for. No web bloat, just pure Vim keybindings. The w3m integration for comments/replies works beautifully.",depth:0,time:"1h ago"}]},{title:"Show HN: Talos – Open-source WASM interpreter for Lean",pts:69,author:"mfornet",time:"13h ago",comments:15,read:!1,commentsList:[{author:"lean_user",text:"Very exciting! Running Lean interactively in WASM has lots of potential for theorem proving on the web.",depth:0,time:"10h ago"}]},{title:"Show HN: Interactive terminal simulator on a landing page",pts:145,author:"astro_dev",time:"1d ago",comments:9,read:!1,commentsList:[{author:"tui_crafter",text:"I love that the simulator actually reacts to keyboard inputs. Really sets the demo apart.",depth:0,time:"18h ago"}]}]};let v="top",r=0,b=!1;const d=document.getElementById("stories-list"),G=document.querySelectorAll(".tui-tab"),a=document.getElementById("tui-window"),g=document.querySelector("#tui-status .status-text");function B(e){if(!d)return;d.innerHTML="",k[e].forEach((t,o)=>{const n=document.createElement("div");n.className=`tui-item ${o===r?"selected":""}`;const i=t.read?"✓ ":"",m=t.read?'style="color: var(--gray);"':"";let c="";t.read&&o!==r?c=`<span class="meta-pts" style="color: var(--gray);">${t.pts} pts</span> · <span style="color: var(--gray);">by ${t.author}</span> · <span class="meta-time">${t.time}</span> · <span class="meta-comments" style="color: var(--gray);">${t.comments} comments</span>`:c=`<span class="meta-pts">${t.pts} pts</span> · by <span class="meta-author">${t.author}</span> · <span class="meta-time">${t.time}</span> · <span class="meta-comments">${t.comments} comments</span>`,n.innerHTML=`
				<div class="tui-item-body">
					<div class="tui-item-title" ${m}>${i}${t.title}</div>
					<div class="tui-item-meta">${c}</div>
				</div>
			`,n.addEventListener("click",()=>{r===o?K(t):(r=o,T())}),d.appendChild(n)})}function T(){if(!d)return;d.querySelectorAll(".tui-item").forEach((o,n)=>{n===r?(o.classList.add("selected"),o.scrollIntoView({block:"nearest",behavior:"smooth"})):o.classList.remove("selected")});const s=k[v];d.querySelectorAll(".tui-item-meta").forEach((o,n)=>{const i=s[n];i.read&&n!==r?o.innerHTML=`<span class="meta-pts" style="color: var(--gray);">${i.pts} pts</span> · <span style="color: var(--gray);">by ${i.author}</span> · <span class="meta-time">${i.time}</span> · <span class="meta-comments" style="color: var(--gray);">${i.comments} comments</span>`:o.innerHTML=`<span class="meta-pts">${i.pts} pts</span> · by <span class="meta-author">${i.author}</span> · <span class="meta-time">${i.time}</span> · <span class="meta-comments">${i.comments} comments</span>`})}function p(e){b&&C(),v=e,r=0,G.forEach(s=>{s.getAttribute("data-cat")===e?s.classList.add("active"):s.classList.remove("active")}),B(e)}function K(e){b=!0,e.read=!0;const s=document.getElementById("tui-footer-bar");if(s&&(s.innerHTML=`
				<span class="footer-key">esc/q</span> <span class="footer-desc">back</span> | 
				<span class="footer-key">j/k</span> <span class="footer-desc">scroll</span> | 
				<span class="footer-key">o</span> <span class="footer-desc">link</span> | 
				<span class="footer-key">r</span> <span class="footer-desc">reply</span> | 
				<span class="footer-key">?</span> <span class="footer-desc">help</span>
			`),d){d.innerHTML="";const t=document.createElement("div");t.className="tui-comments-container";const o=document.createElement("div");o.className="tui-comments-header";const n=document.createElement("div");n.className="tui-back-button",n.innerHTML="❮ Back to list (esc)",n.addEventListener("click",C);const i=document.createElement("div");i.className="tui-comment-story-title",i.innerText=e.title,o.appendChild(n),o.appendChild(i),t.appendChild(o);const m=document.createElement("div");if(m.className="tui-comments-list",e.commentsList&&e.commentsList.length>0){const c=["var(--tui-primary, #ff6600)","var(--tui-cyan, #00f0ff)","var(--tui-blue, #85a5ff)","var(--tui-green, #adff2f)","#ff00ff"];e.commentsList.forEach(h=>{const f=document.createElement("div");f.className="tui-comment-item",f.style.marginLeft=`${h.depth*16}px`,h.depth>0&&(f.style.borderLeft=`1px solid ${c[(h.depth-1)%c.length]}`,f.style.paddingLeft="10px"),f.innerHTML=`
						<div class="tui-comment-meta">
							by <span class="meta-author">${h.author}</span> · <span class="meta-time">${h.time}</span>
						</div>
						<div class="tui-comment-body">${h.text}</div>
					`,m.appendChild(f)})}else{const c=document.createElement("div");c.style.color="var(--gray)",c.style.fontSize="13px",c.style.padding="20px 0",c.innerText="No comments yet.",m.appendChild(c)}t.appendChild(m),d.appendChild(t)}}function C(){b=!1;const e=document.getElementById("tui-footer-bar");e&&(e.innerHTML=`
				<span class="footer-key">q</span> <span class="footer-desc">quit</span> | 
				<span class="footer-key">tab</span> <span class="footer-desc">feed</span> | 
				<span class="footer-key">j/k</span> <span class="footer-desc">nav</span> | 
				<span class="footer-key">enter</span> <span class="footer-desc">view</span> | 
				<span class="footer-key">o</span> <span class="footer-desc">link</span> | 
				<span class="footer-key">w</span> <span class="footer-desc">post</span> | 
				<span class="footer-key">/</span> <span class="footer-desc">search</span> | 
				<span class="footer-key">?</span> <span class="footer-desc">help</span>
			`),B(v)}G.forEach(e=>{e.addEventListener("click",()=>{const s=e.getAttribute("data-cat");p(s)})});a&&(a.addEventListener("focus",()=>{g&&(g.innerText="KEYBOARD ACTIVE (Use vim keys)",g.style.color="var(--green)")}),a.addEventListener("blur",()=>{g&&(g.innerText="Interactive TUI (Click to focus)",g.style.color="")}),a.addEventListener("keydown",e=>{if(["ArrowUp","ArrowDown","j","k","Enter","Escape","q","1","2","3","4","5","Tab","o","w","r","?"].includes(e.key)&&(e.preventDefault(),e.stopPropagation()),b){if(e.key==="Escape"||e.key==="q")C();else if(e.key==="o"){const n=k[v][r];n&&window.open("https://hn.algolia.com/?q="+encodeURIComponent(n.title),"_blank")}else e.key==="r"?window.open("https://news.ycombinator.com/login?goto=submit","_blank"):e.key==="?"&&alert(`hn-client simulator (Comments View):

• esc / q - Back to story list
• j / k - Scroll comments
• o - Open original link
• r - Reply to thread`);const t=document.querySelector(".tui-comments-container");t&&(e.key==="j"||e.key==="ArrowDown"?t.scrollTop+=24:(e.key==="k"||e.key==="ArrowUp")&&(t.scrollTop-=24))}else{const t=k[v];if(e.key==="j"||e.key==="ArrowDown")r<t.length-1&&(r++,T());else if(e.key==="k"||e.key==="ArrowUp")r>0&&(r--,T());else if(e.key==="Enter")K(t[r]);else if(e.key==="o"){const o=t[r];o&&window.open("https://hn.algolia.com/?q="+encodeURIComponent(o.title),"_blank")}else if(e.key==="w")window.open("https://news.ycombinator.com/login?goto=submit","_blank");else if(e.key==="1")p("top");else if(e.key==="2")p("new");else if(e.key==="3")p("best");else if(e.key==="4")p("ask");else if(e.key==="5")p("show");else if(e.key==="Tab"){const o=["top","new","best","ask","show"],n=(o.indexOf(v)+1)%o.length;p(o[n])}else e.key==="?"&&alert(`hn-client simulator (Stories View):

• q - Quit (simulator focus loss)
• tab - Cycle feed categories
• 1..5 - Switch feeds directly
• j / k - Navigate items
• enter - View comments
• o - Open link
• w - Submit post
• / - Filter active list`)}}));B("top");window.addEventListener("DOMContentLoaded",()=>{const e=document.getElementById("system-preloader");e&&setTimeout(()=>{e.classList.add("fade-out")},1300)});const y=document.getElementById("copy-btn"),H=document.getElementById("install-command"),L=document.getElementById("copy-btn-text");y&&H&&L&&y.addEventListener("click",()=>{const e=H.innerText;navigator.clipboard.writeText(e).then(()=>{L.innerText="Copied!",y.style.borderColor="var(--green)",y.style.color="var(--green)",setTimeout(()=>{L.innerText="Copy",y.style.borderColor="",y.style.color=""},2e3)}).catch(s=>{console.error("Copy failed: ",s)})});const Q=document.querySelectorAll(".feature-card");Q.forEach(e=>{e.addEventListener("mousemove",s=>{const t=e.getBoundingClientRect(),o=s.clientX-t.left,n=s.clientY-t.top;e.style.setProperty("--mouse-x",`${o}px`),e.style.setProperty("--mouse-y",`${n}px`)})});const J=document.querySelectorAll(".reveal"),X={root:null,rootMargin:"0px",threshold:.15},z=new IntersectionObserver((e,s)=>{e.forEach(t=>{t.isIntersecting&&(t.target.classList.add("visible"),s.unobserve(t.target))})},X);J.forEach(e=>{z.observe(e)});const $=document.querySelectorAll(".key-tab"),Z=document.querySelectorAll(".key-item");$.forEach(e=>{e.addEventListener("click",()=>{$.forEach(t=>t.classList.remove("active")),e.classList.add("active");const s=e.getAttribute("data-group");Z.forEach(t=>{s==="all"||t.getAttribute("data-group")===s?t.classList.remove("hidden"):t.classList.add("hidden")})})});const ee=document.querySelectorAll(".faq-trigger");ee.forEach(e=>{e.addEventListener("click",()=>{const s=e.closest(".faq-item");if(!s)return;const t=s.querySelector(".faq-content");if(!t)return;const o=s.classList.contains("active");document.querySelectorAll(".faq-item").forEach(n=>{if(n!==s){n.classList.remove("active");const i=n.querySelector(".faq-content");i&&(i.style.maxHeight="")}}),o?(s.classList.remove("active"),t.style.maxHeight=""):(s.classList.add("active"),t.style.maxHeight=t.scrollHeight+"px")})});const I=document.getElementById("reboot-btn"),E=document.getElementById("app-container"),x=document.getElementById("system-preloader"),N=document.querySelector(".preloader-lines"),_=document.querySelector(".preloader-progress-fill");window.addEventListener("scroll",()=>{window.scrollY>400?I?.classList.add("visible"):I?.classList.remove("visible")});I?.addEventListener("click",()=>{if(!(!E||!x)){if(E.classList.add("deconstruct"),N&&(N.innerHTML=`
				<div class="preloader-line line-1">❯ reboot sequence triggered...</div>
				<div class="preloader-line line-2">❯ flushing virtual memory cache...</div>
				<div class="preloader-line line-3">❯ compiling DOM layout trees...</div>
				<div class="preloader-line line-4">❯ system online (rebuilt)</div>
			`),_){const e=_;e.style.animation="none",e.offsetHeight,e.style.animation=""}x.classList.remove("fade-out"),setTimeout(()=>{window.scrollTo({top:0}),E.classList.remove("deconstruct"),document.querySelectorAll(".reveal").forEach(s=>{s.classList.remove("visible")}),document.querySelectorAll(".reveal").forEach(s=>{z.observe(s)})},600),setTimeout(()=>{x.classList.add("fade-out")},1800)}});document.querySelectorAll('a[href^="#"]').forEach(e=>{e.addEventListener("click",function(s){const t=this.getAttribute("href");if(t==="#")return;const o=document.querySelector(t);if(o){s.preventDefault();const m=o.getBoundingClientRect().top+window.pageYOffset-90;window.scrollTo({top:m,behavior:"smooth"})}})});const te=document.querySelectorAll("section[id]"),se=document.querySelectorAll('.nav-link[href^="#"]');window.addEventListener("scroll",()=>{let e="";const s=window.scrollY+180;te.forEach(t=>{const o=t.offsetTop,n=t.clientHeight;s>=o&&s<o+n&&(e=t.getAttribute("id")||"")}),se.forEach(t=>{t.classList.remove("active"),t.getAttribute("href")===`#${e}`&&t.classList.add("active")})});const oe={1:{title:"Level 1: Booting the TUI Kernel",desc:`<p>In Go, executable programs are compiled from a source entry point located inside <strong><code>package main</code></strong>. The execution runtime automatically invokes the <strong><code>func main()</code></strong> function when the binary starts.</p>
<p>To build our terminal UI, we import Charm's <strong>Bubble Tea</strong> library. We initialize our application instance using <code>tea.NewProgram()</code>. By passing the <strong><code>tea.WithAltScreen()</code></strong> option, Go instructs the terminal emulator to open a separate screen buffer. This suspends your shell session, hides previous commands, and gives the TUI control over the entire window dimensions. Upon quitting, it cleanly restores the shell state.</p>`,checklist:["Declare package main","Import github.com/charmbracelet/bubbletea","Invoke tea.NewProgram event loop","Initialize alt screen buffer"],code:`package main

import (
	"fmt"
	"os"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v
", err)
		os.Exit(1)
	}
}`},2:{title:"Level 2: The UI Blueprint (State)",desc:`<p>Go is a <strong>statically typed</strong> language with strict compile-time checks. Unlike object-oriented languages, Go has no classes or classical inheritance. Instead, we use <strong><code>struct</code></strong> composition to define the shape and properties of our data.</p>
<p>Our program's state is encapsulated within the <strong><code>type model struct</code></strong>. We define slices (resizable arrays) to store stories, and maps (key-value hash tables) to cache comments. In Go, maps must be allocated in memory using the built-in <strong><code>make()</code></strong> function before write operations; writing to an unallocated map results in a runtime panic. The constructor function <code>initialModel()</code> returns our initialized state model.</p>`,checklist:["Declare type model struct","Define fields for list cursor, category, loading states","Write initialModel() helper function","Use make() to allocate comments map"],code:`type model struct {
	stories  []hnapi.Item
	comments map[int][]comment
	cursor   int
	loading  bool
}

func initialModel() model {
	return model{
		loading:  true,
		comments: make(map[int][]comment),
		cursor:   0,
	}
}`},3:{title:"Level 3: Intercepting Key Inputs",desc:"<p>Events (keyboard inputs, mouse clicks, resize events) are processed by the <strong><code>Update()</code></strong> receiver method. In Go, methods are attached to structs using a receiver parameter: <code>func (m model) Update(msg tea.Msg)</code>.</p>\n<p>The message `msg` is passed as a <code>tea.Msg</code>, which is an empty interface (`interface{}` or `any`), meaning it can contain any value. We use a <strong>type switch</strong> (<code>switch msg := msg.(type)</code>) to inspect the concrete type of the message. If it is a <strong><code>tea.KeyMsg</code></strong>, we match on keys (like Vim <code>j</code>/<code>k</code> keys to scroll or <code>q</code> to exit), modify our cursor index, and return the updated model state.</p>",checklist:["Implement Update(msg tea.Msg) method","Check if message is tea.KeyMsg type","Switch on keys: q, ctrl+c to tea.Quit","Modify cursor variable for Vim navigation keys"],code:`func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 { m.cursor-- }
		case "down", "j":
			m.cursor++
		}
	}
	return m, nil
}`},4:{title:"Level 4: Rendering Graphics to Screen",desc:`<p>The <strong><code>View()</code></strong> receiver method is called automatically after every state update. It takes a copy of our model and returns a single formatted string that represents the entire terminal display.</p>
<p>We build our layout programmatically. For complex layouts, we use <strong><code>strings.Builder</code></strong> for memory-efficient string concatenation (preventing unnecessary garbage collection allocations). We style text with colors, padding, and alignments using <strong>Lip Gloss</strong>, which compiles CSS-like declarations into standard ANSI terminal escape sequences that shell buffers render natively.</p>`,checklist:["Implement View() string receiver method","Verify model.loading state before drawing list","Iterate stories slice with for-range loop","Apply orange highlight styling to the selected cursor item"],code:`func (m model) View() string {
	if m.loading {
		return "⌛ Loading stories..."
	}
	var s string
	s += "─── Hacker News ───

"
	for i, story := range m.stories {
		if m.cursor == i {
			s += fmt.Sprintf("❯ %s
", story.Title)
		} else {
			s += fmt.Sprintf("  %s
", story.Title)
		}
	}
	s += "
(q: quit | j/k: navigate)"
	return s
}`},5:{title:"Level 5: Going Async & Launching w3m",desc:"<p>Go handles asynchronous operations concurrently via <strong>Goroutines</strong>. In Bubble Tea, background jobs are encapsulated as commands: <strong><code>tea.Cmd</code></strong> (a function returning a message). We run our Hacker News API HTTP calls in a command, which runs concurrently without blocking the main UI thread.</p>\n<p>When the background task completes, its return value is safely dispatched back into the main `Update()` loop. To launch terminal web browsers like <code>w3m</code> for posting replies, we instantiate an OS process command via Go's <strong><code>os/exec</code></strong> package and pass it to <strong><code>tea.ExecProcess</code></strong>, which pauses our AltScreen loop, hands full terminal control to `w3m`, and restores the TUI when `w3m` terminates.</p>",checklist:["Create asynchronous fetchStories() command","Process statusMsg payload in Update loop",'Setup exec.Command("w3m", url) process',"Use tea.ExecProcess to suspend TUI and handoff to w3m"],code:`func fetchStories(category string) tea.Cmd {
	return func() tea.Msg {
		stories, err := hnapi.GetStories(category)
		if err != nil {
			return errMsg{err}
		}
		return statusMsg(stories)
	}
}

func openURL(url string) tea.Cmd {
	c := exec.Command("w3m", url)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil { return errMsg{err} }
		return nil
	})
}`}};let w=1;const R=document.querySelectorAll(".level-tab"),U=document.getElementById("level-title"),P=document.getElementById("level-desc"),O=document.getElementById("level-checklist"),j=document.getElementById("level-code-block"),S=document.getElementById("lvl-prev-btn"),A=document.getElementById("lvl-next-btn");function M(e){w=e;const s=oe[e];s&&(R.forEach(t=>{parseInt(t.getAttribute("data-level")||"1")===e?t.classList.add("active"):t.classList.remove("active")}),U&&(U.innerText=s.title),P&&(P.innerHTML=s.desc),j&&(j.innerText=s.code),O&&(O.innerHTML=s.checklist.map(t=>`<li class="checked">${t}</li>`).join("")),S&&(S.disabled=e===1),A&&(A.disabled=e===5))}R.forEach(e=>{e.addEventListener("click",()=>{const s=parseInt(e.getAttribute("data-level")||"1");M(s)})});S?.addEventListener("click",()=>{w>1&&M(w-1)});A?.addEventListener("click",()=>{w<5&&M(w+1)});const W=document.getElementById("os-detection-badge");if(W){let e="Unknown OS";const s=window.navigator.userAgent.toLowerCase();s.indexOf("win")!==-1?e="Windows":s.indexOf("mac")!==-1?e="macOS":s.indexOf("linux")!==-1?e="Linux":s.indexOf("x11")!==-1&&(e="Unix"),W.innerHTML=`💻 Detected OS: <strong>${e}</strong> — Get started with the installation below.`}const D=document.querySelectorAll("[data-tui-theme]");D.forEach(e=>{e.addEventListener("click",()=>{D.forEach(t=>t.classList.remove("active")),e.classList.add("active");const s=e.getAttribute("data-tui-theme");a&&(a.className="terminal-window",s!=="orange"&&a.classList.add(`theme-${s}`))})});let l=!1;const F=()=>{l=!0};a?.addEventListener("mousedown",F);a?.addEventListener("keydown",F);const u=e=>new Promise(s=>setTimeout(s,e)),ne=async()=>{await u(2e3),!l&&(a&&a.dispatchEvent(new KeyboardEvent("keydown",{key:"Tab"})),await u(1800),!l&&(a&&a.dispatchEvent(new KeyboardEvent("keydown",{key:"Tab"})),await u(1800),!l&&(a&&a.dispatchEvent(new KeyboardEvent("keydown",{key:"j"})),await u(1e3),!l&&(a&&a.dispatchEvent(new KeyboardEvent("keydown",{key:"j"})),await u(1e3),!l&&(a&&a.dispatchEvent(new KeyboardEvent("keydown",{key:"Enter"})),await u(3500),!l&&(a&&a.dispatchEvent(new KeyboardEvent("keydown",{key:"j"})),await u(800),!l&&(a&&a.dispatchEvent(new KeyboardEvent("keydown",{key:"j"})),await u(1200),!l&&a&&a.dispatchEvent(new KeyboardEvent("keydown",{key:"Escape"})))))))))},V=document.getElementById("terminal-demo");if(V){const e=new IntersectionObserver(s=>{s.forEach(t=>{t.isIntersecting&&(ne(),e.disconnect())})},{threshold:.3});e.observe(V)}const q=document.getElementById("mobile-menu-toggle"),Y=document.querySelector(".nav");q?.addEventListener("click",()=>{q.classList.toggle("active"),Y?.classList.toggle("active")});document.querySelectorAll(".nav .nav-link, .nav .nav-github").forEach(e=>{e.addEventListener("click",()=>{q?.classList.remove("active"),Y?.classList.remove("active")})});
