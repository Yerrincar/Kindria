package tui

import (
	metadata "Kindria/internal/core/api/books"
	"Kindria/internal/utils"
	"log"
	"os"
	"strings"

	"image/color"

	"github.com/blacktop/go-termimg"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/disintegration/imaging"
	"golang.org/x/sys/unix"
)

type sessionState int
type focusArea int

const (
	homeState sessionState = iota
	librayState
	sideFocus focusArea = iota
	contentFocus
)

var (
	normal    = lipgloss.Color("#EEEEEE")
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	borders   = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#8a2be2"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	list = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, true, true, true).
		BorderForeground(subtle)

	listFocused = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, true, true, true).
			BorderForeground(highlight).
			MarginRight(1)
)

type MainModel struct {
	state   sessionState
	library *Model
}

type Model struct {
	books             []*metadata.Package
	cursor            int
	sideBarCursor     int
	activeArea        int
	selected          map[int]struct{}
	width             int
	height            int
	sideBarWidth      int
	dynamicCardWidth  int
	dynamicCardHeight int
	cellPixelWidth    int
	cellPixelHeight   int
	cols              int
	paginator         paginator.Model
	covers            map[int]string
	handler           metadata.Handler
}

type coversLoadedMsg map[int]string

var debugEnabled = os.Getenv("KINDRIA_DEBUG") != ""

func debugLog(format string, args ...any) {
	if !debugEnabled {
		return
	}
	log.Printf("tui: "+format, args...)
}

/* ----- MainModel ----- */

func InitialModel(b []*metadata.Package, h *metadata.Handler) *MainModel {

	p := paginator.New()
	p.Type = paginator.Dots
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
	debugLog("InitialModel: books=%d", len(b))
	library := &Model{
		books:     b,
		selected:  make(map[int]struct{}),
		paginator: p,
		covers:    make(map[int]string),
		handler:   *h,
	}
	return &MainModel{
		state:   homeState,
		library: library,
	}
}

func (m *MainModel) Init() tea.Cmd {
	debugLog("Init")
	return nil
}

func (m *MainModel) View() string {
	fig := utils.Fig()
	if m.state == homeState {
		return lipgloss.Place(m.library.width, m.library.height, lipgloss.Center, lipgloss.Center,
			fig+"\n  󱉟 Library"+strings.Repeat(" ", 15)+"l")
	}
	return m.library.View()
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "l":
			if m.state == homeState {
				m.state = librayState
				return m, m.library.syncVisibleWidget()
			}
		case "ctrl+h", "esc":
			if m.library.activeArea == int(contentFocus) {
				m.library.activeArea = int(sideFocus)
			}
		}

	case tea.WindowSizeMsg:
		m.library.width = msg.Width
		m.library.sideBarWidth = msg.Width / 8
		m.library.height = msg.Height - 3
		m.library.cols = 6
		contentWidth := m.library.width - m.library.sideBarWidth - 6
		contentHeight := m.library.height
		m.library.dynamicCardWidth = (contentWidth / m.library.cols) - 2
		m.library.dynamicCardHeight = int(float64(m.library.dynamicCardWidth)*0.74) - 2
		if m.library.dynamicCardWidth < 10 {
			m.library.cols = 2
			m.library.dynamicCardWidth = (contentWidth / 2) - 3
			m.library.dynamicCardHeight = int(float64(m.library.dynamicCardWidth)*0.74) - 3
		}
		m.library.cellPixelWidth, m.library.cellPixelHeight = getCellPixelSize(m.library.width, m.library.height)
		if m.library.cellPixelWidth > 0 && m.library.cellPixelHeight > 0 {
			debugLog("CellPixels: w=%d h=%d", m.library.cellPixelWidth, m.library.cellPixelHeight)
		} else {
			debugLog("CellPixels: unavailable, using fallback")
		}
		m.library.paginator.PerPage = m.library.cols * (contentHeight / (m.library.dynamicCardHeight))
		m.library.paginator.SetTotalPages(len(m.library.books))
	}

	if m.state == librayState {
		newLib, cmd := m.library.Update(msg)
		m.library = newLib.(*Model)
		return m, cmd
	}

	return m, nil
}

/* ----- Library ----- */
func (m *Model) Init() tea.Cmd {
	debugLog("Init")
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmdSync tea.Cmd
	var cmdPaginator tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		debugLog("WindowSize: w=%d h=%d", msg.Width, msg.Height)

	case tea.KeyMsg:
		debugLog("Key: %s", msg.String())
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.books)-1 {
				m.cursor++
			}
		case "right", "l":
			m.paginator.NextPage()
			cmdSync = m.syncVisibleWidget()
		case "left", "h":
			m.paginator.PrevPage()
			cmdSync = m.syncVisibleWidget()
		}

	case coversLoadedMsg:
		debugLog("coversLoadedMsg: count=%d", len(msg))
		m.covers = msg
		return m, nil
	}

	m.paginator, cmdPaginator = m.paginator.Update(msg)

	return m, tea.Batch(cmdPaginator, cmdSync)
}

func (m Model) View() string {
	debugLog("View")
	if m.cols <= 0 || m.dynamicCardWidth <= 0 {
		return "\n  Initializing Kindria..."
	}
	var b strings.Builder
	b.WriteString("\n Kindria, your TUI e-book library\n")

	start, end := m.paginator.GetSliceBounds(len(m.books))
	booksCards := make([]string, 0)

	for i := range m.books[start:end] {
		absoluteIndex := i + start
		cover, _ := m.covers[absoluteIndex]

		//b.WriteString(string(strings.Count(cover, "\n")))
		//if !ok || cover == "" {
		//	cover += strings.Repeat("\n", 10)
		//}
		style := list.
			Width(m.dynamicCardWidth).
			Height(m.dynamicCardHeight)

		if absoluteIndex == m.cursor {
			style = style.BorderForeground(highlight)
		}
		booksCards = append(booksCards, style.Render(cover))
	}

	var rows []string
	for i := 0; i < len(booksCards); i += m.cols {
		endIdx := i + m.cols

		if endIdx > len(booksCards) {
			endIdx = len(booksCards)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, booksCards[i:endIdx]...))
	}
	libraryBorderStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true, true, true, true).
		BorderForeground(borders).Width(m.width - m.sideBarWidth - 4).Height(m.height).PaddingLeft(2).
		PaddingTop(1)
	book := lipgloss.JoinVertical(lipgloss.Top, rows...)
	books := lipgloss.JoinHorizontal(lipgloss.Top, book)
	library := libraryBorderStyle.Render(books)
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, m.SideBarView(), library))
	b.WriteString("\n  " + m.paginator.View())
	return b.String()
}

func (m *Model) syncVisibleWidget() tea.Cmd {
	start, end := m.paginator.GetSliceBounds(len(m.books))
	localCovers := make(map[int]string)
	booksToLoad := m.books[start:end]
	curWidth := m.dynamicCardWidth
	curHeight := m.dynamicCardHeight
	curCellPixelWidth := m.cellPixelWidth
	curCellPixelHeight := m.cellPixelHeight

	return func() tea.Msg {
		debugLog("syncVisibleWidget start: start=%d end=%d books=%d", start, end, len(booksToLoad))
		protocol := termimg.DetectProtocol()
		features := termimg.QueryTerminalFeatures()
		for i, book := range booksToLoad {
			path, err := m.handler.SelectBookPath(book.BookFile)
			if err != nil {
				log.Printf("Error getting book path for book: "+book.BookFile+"%v: ", err)
				continue
			}
			if path == "" {
				continue
			}
			targetPixelWidth := curWidth
			targetPixelHeight := curHeight
			if curCellPixelWidth > 0 && curCellPixelHeight > 0 {
				targetPixelWidth = curWidth * curCellPixelWidth
				targetPixelHeight = curHeight * curCellPixelHeight
			} else if features != nil && features.FontWidth > 0 && features.FontHeight > 0 {
				targetPixelWidth = curWidth * features.FontWidth
				targetPixelHeight = curHeight * features.FontHeight
			}
			srcImage, err := imaging.Open(path)
			if err != nil {
				continue
			}

			resizedImage := imaging.Fit(srcImage, targetPixelWidth, targetPixelHeight, imaging.Lanczos)
			if resizedImage.Bounds().Dx() != targetPixelWidth || resizedImage.Bounds().Dy() != targetPixelHeight {
				canvas := imaging.New(targetPixelWidth, targetPixelHeight, color.NRGBA{R: 10, G: 10, B: 10, A: 255})
				resizedImage = imaging.PasteCenter(canvas, resizedImage)
			}

			img := termimg.New(resizedImage).Scale(termimg.ScaleNone)
			if protocol == termimg.Halfblocks {
				img = img.Dither(true).DitherMode(termimg.DitherFloydSteinberg)
			}

			cover := termimg.NewImageWidget(img)
			cover.SetSize(curWidth, curHeight).SetProtocol(protocol)
			finalCover := ""
			if cover != nil {
				coverRendered, err := cover.Render()
				if err != nil {
					log.Printf("Err rendering cover: %v ", err)
				}
				coverRendered = strings.TrimLeft(coverRendered, "\n")
				finalCover += coverRendered
			} else {
				finalCover += ""
			}
			localCovers[i+start] = finalCover
		}
		return coversLoadedMsg(localCovers)
	}
}

func getCellPixelSize(cols, rows int) (int, int) {
	if cols <= 0 || rows <= 0 {
		return 0, 0
	}
	var f *os.File
	var err error
	if f, err = os.OpenFile("/dev/tty", unix.O_NOCTTY|unix.O_CLOEXEC|unix.O_NDELAY|unix.O_RDWR, 0666); err != nil {
		return 0, 0
	}
	defer f.Close()
	sz, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0
	}
	debugLog("Winsize: rows=%d cols=%d xpixel=%d ypixel=%d", sz.Row, sz.Col, sz.Xpixel, sz.Ypixel)
	if sz.Col == 0 || sz.Row == 0 || sz.Xpixel == 0 || sz.Ypixel == 0 {
		return 0, 0
	}
	cellW := int(sz.Xpixel) / int(sz.Col)
	cellH := int(sz.Ypixel) / int(sz.Row)
	if cellW <= 0 || cellH <= 0 {
		return 0, 0
	}
	return cellW, cellH
}

func (m *Model) SideBarView() string {
	var options string

	rawOptionsList := []string{"Home", "Books", "To-Be Read"}
	renderedOptionsList := make([]string, len(rawOptionsList))

	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true, true, true, true).
		BorderForeground(borders).Width(m.sideBarWidth).Height(m.height).PaddingTop(1)

	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(240)).
		PaddingLeft(1).MarginBottom(1)
	activeStyle := inactiveStyle.Copy().Foreground(lipgloss.Color("#7D56F4")).
		PaddingLeft(0)

	for i, word := range rawOptionsList {
		if i == m.sideBarCursor {
			text := "> " + utils.ToSansBold(word)
			renderedOptionsList[i] = activeStyle.Render(text)
		} else {
			text := utils.ToSansBold(word)
			renderedOptionsList[i] = inactiveStyle.Render(text)
		}
	}

	items := lipgloss.JoinVertical(lipgloss.Left, renderedOptionsList...)
	options = style.Render(items)
	return options
}
