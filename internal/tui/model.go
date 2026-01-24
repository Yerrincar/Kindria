package tui

import (
	metadata "Kindria/internal/core/api/books"
	"log"
	"os"
	"strings"

	"github.com/blacktop/go-termimg"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/disintegration/imaging"
)

var (
	normal    = lipgloss.Color("#EEEEEE")
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	list = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, true, true, true).
		BorderForeground(subtle).
		MarginRight(1)

	listFocused = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, true, true, true).
			BorderForeground(highlight).
			MarginRight(1)
)

type Model struct {
	books             []*metadata.Package
	cursor            int
	selected          map[int]struct{}
	width             int
	height            int
	dynamicCardWidth  int
	dynamicCardHeight int
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

func InitialModel(b []*metadata.Package, h *metadata.Handler) *Model {

	p := paginator.New()
	p.Type = paginator.Dots
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
	debugLog("InitialModel: books=%d", len(b))
	return &Model{
		books:     b,
		selected:  make(map[int]struct{}),
		paginator: p,
		covers:    make(map[int]string),
		handler:   *h,
	}
}

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
		m.width, m.height = msg.Width, msg.Height
		m.cols = 6
		m.dynamicCardWidth = (m.width / m.cols) - 3
		m.dynamicCardHeight = int(float64(m.dynamicCardWidth) * 0.66)
		if m.dynamicCardWidth < 10 {
			m.cols = 2 // Fallback if too small
			m.dynamicCardWidth = (m.width / 2) - 3
			m.dynamicCardHeight = int(float64(m.dynamicCardWidth) * 1.0)
		}
		m.paginator.PerPage = m.cols * (m.height / (m.dynamicCardHeight + 2))
		m.paginator.SetTotalPages(len(m.books))
		cmdSync = m.syncVisibleWidget()

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

	for i, _ := range m.books[start:end] {
		absoluteIndex := i + start
		cover, ok := m.covers[absoluteIndex]
		if !ok || cover == "" {
			cover += strings.Repeat("\n", 10)
		}
		style := list.Copy().
			Width(m.dynamicCardWidth).
			Height(m.dynamicCardHeight)

		if absoluteIndex == m.cursor {
			style = style.BorderForeground(highlight)
		}
		//renderedBook := m.renderBookCard(book, absoluteIndex == m.cursor, cover)
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

	b.WriteString(lipgloss.JoinVertical(lipgloss.Left, rows...))
	b.WriteString("\n  " + m.paginator.View())
	return b.String()
}

//func (m Model) renderBookCard(p *metadata.Package, isFocused bool, imageString string) string {
//	lipglossStyle := list
//	if isFocused {
//		lipglossStyle = listFocused
//	}
//
//	//line11 := truncString(p.Metadata.Title, 18)
//	//line12 := truncString(p.Metadata.Author, 18)
//
//	finalCard := lipglossStyle.Render(lipgloss.JoinVertical(lipgloss.Left, imageString))
//	return finalCard
//}

//func truncString(s string, max int) string {
//	if len(s) > max {
//		return s[:max]
//	}
//	return s
//}

func (m *Model) syncVisibleWidget() tea.Cmd {
	start, end := m.paginator.GetSliceBounds(len(m.books))
	localCovers := make(map[int]string)
	booksToLoad := m.books[start:end]
	curWidth := m.dynamicCardWidth
	curHeight := m.dynamicCardHeight
	debugLog("syncVisibleWidget queued: start=%d end=%d books=%d", start, end, len(booksToLoad))
	return func() tea.Msg {
		debugLog("syncVisibleWidget start: start=%d end=%d books=%d", start, end, len(booksToLoad))
		for i, _ := range booksToLoad {
			//	path, err := m.handler.SelectBookPath(book.BookFile)
			//	if err != nil {
			//		log.Printf("Error getting book path for book: "+book.BookFile+"%v: ", err)
			//		continue
			//	}
			targetPixelWidth := curWidth * 48
			targetPixelHeight := curHeight * 72
			srcImage, err := imaging.Open("./cache/covers/Reyes_de_la_Tierra_Salvaje.jpg")
			if err != nil {
				continue
			}
			resizedImage := imaging.Fit(srcImage, targetPixelWidth, targetPixelHeight, imaging.CatmullRom)
			resizedImage = imaging.Sharpen(resizedImage, 1.5)
			cover := termimg.NewImageWidgetFromImage(resizedImage)

			cover.SetSize(curWidth, curHeight).SetProtocol(termimg.Auto)
			finalCover := ""
			if cover != nil {
				coverRendered, err := cover.Render()
				coverRendered = lipgloss.NewStyle().Width(curWidth).Height(curHeight).Render(coverRendered)
				if err != nil {
					log.Printf("Err rendering cover: %v ", err)
				}
				finalCover += coverRendered
			} else {
				finalCover += ""
			}
			localCovers[i+start] = finalCover
		}
		debugLog("syncVisibleWidget end: loaded=%d", len(localCovers))
		return coversLoadedMsg(localCovers)
	}
}
