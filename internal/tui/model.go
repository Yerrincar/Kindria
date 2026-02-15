package tui

import (
	metadata "Kindria/internal/core/api/books"
	"Kindria/internal/utils"
	kindle "Kindria/tools"
	"context"
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/blacktop/go-termimg"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/disintegration/imaging"
	"golang.org/x/sys/unix"
)

type sessionState int
type focusArea int

const (
	homeState sessionState = iota
	librayState
	fileState
	kindleState
	sideFocus focusArea = iota
	contentFocus
)

var (
	normal    = lipgloss.Color("#EEEEEE")
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	borders   = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#8a2be2"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}

	list = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, true, true, true).
		BorderForeground(subtle)
)

type MainModel struct {
	state         sessionState
	library       *Model
	sideBarWidth  int
	filePicker    filepicker.Model
	selectedFiles map[string]struct{}
	failedBooks   []string
	selectedOrder []string
	err           error
	fileInput     textinput.Model
	showFileInput bool
	importing     bool
	showLoader    bool
	importStatus  string
	kindleBooks   []string
	kindleDocsURI string
	kindleCursor  int
	kindleSelect  bool
	kindlePicked  map[string]struct{}
	kindleSyncing bool
	kindleLoader  bool
	kindleStatus  string
}

type Model struct {
	books              []*metadata.Package
	allBooks           []*metadata.Package
	cursor             int
	sideBarCursor      int
	activeArea         int
	width              int
	sideBarWidth       int
	screenHeight       int
	contentWidth       int
	height             int
	lowBarHeight       int
	dynamicCardWidth   int
	dynamicCardHeight  int
	cellPixelWidth     int
	cellPixelHeight    int
	cols               int
	paginator          paginator.Model
	covers             map[int]string
	coverRenderCache   map[string]string
	coverRenderPending map[string]struct{}
	handler            metadata.Handler
	MenuOptions        []string
	start              int
	end                int
	ratingInput        textinput.Model
	showRatingInput    bool
}

type coversLoadedMsg map[int]string

type coverLoadedMsg struct {
	index int
	key   string
	data  string
}

type importLoaderDelayMsg struct{}

type importFinishedMsg struct {
	successfulCopies []string
	failedBooks      []string
	duplicateCount   int
	refreshedBooks   []*metadata.Package
	err              error
}

type kindleBooksLoadedMsg struct {
	docsURI string
	books   []string
	err     error
}

type kindleSyncDelayMsg struct{}

type kindleSyncFinishedMsg struct {
	inserted      int
	failed        int
	duplicated    int
	refreshedBook []*metadata.Package
	err           error
}

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

	t := textinput.New()
	t.Placeholder = "0.0-5.0"
	t.Focus()
	t.CharLimit = 10
	t.Width = 10

	f := textinput.New()
	f.Placeholder = "Introduce .epub' folder path"
	f.CharLimit = 60
	f.Width = 60

	fp := filepicker.New()
	fp.AllowedTypes = []string{".epub"}
	fp.CurrentDirectory = "/home/yeray/Downloads/"
	fp.AutoHeight = false
	fp.ShowPermissions = false
	fp.ShowSize = false
	library := &Model{
		books:              b,
		paginator:          p,
		covers:             make(map[int]string),
		coverRenderCache:   make(map[string]string),
		coverRenderPending: make(map[string]struct{}),
		handler:            *h,
		activeArea:         int(sideFocus),
		MenuOptions:        []string{"Home", "Books", "To-Be Read", "Add Book", "Synchronize \nKindle"},
		showRatingInput:    false,
		ratingInput:        t,
		allBooks:           b,
	}
	return &MainModel{
		state:         homeState,
		library:       library,
		filePicker:    fp,
		fileInput:     f,
		showFileInput: false,
		selectedFiles: make(map[string]struct{}),
		failedBooks:   make([]string, 0),
		selectedOrder: []string{},
		kindleBooks:   []string{},
		kindlePicked:  make(map[string]struct{}),
	}
}

func (m *MainModel) Init() tea.Cmd {
	return m.filePicker.Init()
}

func (m *MainModel) View() string {
	fig := utils.Fig()
	if m.state == homeState {
		return lipgloss.Place(m.library.width, m.library.height, lipgloss.Center, lipgloss.Center,
			fig+"\n  󱉟 Library"+strings.Repeat(" ", 15)+"l/L"+"\n  󱉟 To-Be Read"+strings.Repeat(" ", 12)+
				"t/T"+"\n  󱉟 Add Book"+strings.Repeat(" ", 14)+"a/A"+"\n  󱉟 Synchronize Kindle"+strings.Repeat(" ", 4)+"k/K")
	}
	if m.state == fileState {
		return m.FilePickerView()
	}
	if m.state == kindleState {
		return m.KindleView()
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, m.SideBarView(), m.library.View())
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case importLoaderDelayMsg:
		if m.importing {
			m.showLoader = true
		}
		return m, nil
	case importFinishedMsg:
		m.importing = false
		m.showLoader = false
		if msg.err != nil {
			m.importStatus = "Insert failed: " + msg.err.Error()
			return m, nil
		}
		if len(msg.successfulCopies) > 0 {
			for _, book := range msg.successfulCopies {
				delete(m.selectedFiles, book)
				index := -1
				for i, val := range m.selectedOrder {
					if val == book {
						index = i
						break
					}
				}
				if index != -1 {
					m.selectedOrder = utils.Delete_at_index(m.selectedOrder, index)
				}
			}
		}
		if len(msg.failedBooks) > 0 {
			m.failedBooks = append(m.failedBooks, msg.failedBooks...)
		}
		if len(msg.refreshedBooks) > 0 {
			m.library.allBooks = msg.refreshedBooks
			m.library.books = msg.refreshedBooks
		}
		m.importStatus = fmt.Sprintf("Inserted: %d | Failed: %d | Duplicated: %d", len(msg.successfulCopies), len(msg.failedBooks), msg.duplicateCount)
		return m, nil
	case kindleBooksLoadedMsg:
		if msg.err != nil {
			m.kindleStatus = "Kindle error: " + msg.err.Error()
			return m, nil
		}
		m.kindleDocsURI = msg.docsURI
		m.kindleBooks = msg.books
		m.kindleCursor = 0
		m.kindleStatus = fmt.Sprintf("Found %d books", len(msg.books))
		m.kindlePicked = make(map[string]struct{})
		return m, nil
	case kindleSyncDelayMsg:
		if m.kindleSyncing {
			m.kindleLoader = true
		}
		return m, nil
	case kindleSyncFinishedMsg:
		m.kindleSyncing = false
		m.kindleLoader = false
		if msg.err != nil {
			m.kindleStatus = "Sync failed: " + msg.err.Error()
			return m, nil
		}
		if len(msg.refreshedBook) > 0 {
			m.library.allBooks = msg.refreshedBook
			m.library.books = msg.refreshedBook
		}
		m.kindleStatus = fmt.Sprintf("Inserted: %d | Failed: %d | Duplicated: %d", msg.inserted, msg.failed, msg.duplicated)
		return m, nil
	}

	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.library.width = msg.Width
		m.library.screenHeight = msg.Height
		m.sideBarWidth = msg.Width / 8
		m.library.sideBarWidth = m.sideBarWidth
		m.library.height = msg.Height - 5
		m.library.cols = 6
		m.library.lowBarHeight = 4
		m.library.contentWidth = m.library.width - m.sideBarWidth - 6
		contentHeight := m.library.height - m.library.lowBarHeight
		m.library.dynamicCardWidth = (m.library.contentWidth / m.library.cols) - 2
		m.library.dynamicCardHeight = int(float64(m.library.dynamicCardWidth)*0.74) - 2

		if m.library.dynamicCardWidth < 10 {
			m.library.cols = 2
			m.library.dynamicCardWidth = (m.library.contentWidth / 2) - 3
			m.library.dynamicCardHeight = int(float64(m.library.dynamicCardWidth)*0.74) - 3
		}
		m.library.cellPixelWidth, m.library.cellPixelHeight = getCellPixelSize(m.library.width, m.library.height)
		m.library.paginator.PerPage = m.library.cols * (contentHeight / (m.library.dynamicCardHeight))
		m.library.paginator.SetTotalPages(len(m.library.books))

	}

	if m.state == fileState {
		panelHeight := m.library.height + 2
		pickerHeight, _ := m.filePickerLayout(panelHeight)
		m.filePicker.SetHeight(pickerHeight)

		if !m.showFileInput && len(m.selectedFiles) > 0 {
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch msg.String() {
				case "s":
					if m.importing {
						break
					}
					selected := make([]string, 0, len(m.selectedFiles))
					for book := range m.selectedFiles {
						selected = append(selected, book)
					}
					m.importing = true
					m.showLoader = false
					m.importStatus = ""
					return m, tea.Batch(
						m.importBooksCmd(selected),
						tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
							return importLoaderDelayMsg{}
						}),
					)
				}
			}
		}

		if m.showFileInput {
			switch msg := msg.(type) {
			case tea.KeyMsg:
				switch msg.String() {
				case "ctrl+c":
					return m, tea.Quit
				case "esc":
					m.showFileInput = false
					m.fileInput.Blur()
					m.fileInput.Reset()
					return m, nil
				case "enter":
					fileText := strings.TrimSpace(m.fileInput.Value())
					if fileText != "" {
						if _, err := os.ReadDir(fileText); err != nil {
							m.err = err
							m.fileInput.Placeholder = "Invalid directory"
							m.fileInput.Reset()
							return m, nil
						}
						m.err = nil
						m.filePicker.CurrentDirectory = fileText
						m.showFileInput = false
						m.fileInput.Blur()
						return m, m.filePicker.Init()
					}
				}
			}
			var cmd tea.Cmd
			m.fileInput, cmd = m.fileInput.Update(msg)
			return m, cmd
		}

		if m.library.activeArea == int(sideFocus) {
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				switch keyMsg.String() {
				case "ctrl+l":
					m.library.activeArea = int(contentFocus)
					return m, nil
				case "enter":
					selectedOption := m.library.MenuOptions[m.library.sideBarCursor]
					switch selectedOption {
					case "Home":
						m.state = homeState
						return m, tea.ClearScreen
					case "Books", "To-Be Read":
						m.state = librayState
						m.library.activeArea = int(contentFocus)
						return m, m.library.SetView(selectedOption)
					case "Add Book":
						m.state = fileState
						m.library.activeArea = int(contentFocus)
						return m, tea.Batch(tea.ClearScreen, m.filePicker.Init())
					case "Synchronize Kindle", "Synchronize \nKindle":
						m.state = kindleState
						m.library.activeArea = int(contentFocus)
						return m, tea.Batch(tea.ClearScreen, m.loadKindleBooksCmd())
					}
				}
			}
			newLib, cmd := m.library.Update(msg)
			m.library = newLib.(*Model)
			return m, cmd
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "i", "I":
				m.showFileInput = true
				m.fileInput.Focus()
				return m, nil

			case "esc":
				m.library.activeArea = int(sideFocus)
				return m, nil
			}
		}

		var cmdPicker tea.Cmd
		m.filePicker, cmdPicker = m.filePicker.Update(msg)
		if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
			m.addSelectedFile(path)
			m.err = nil
			panelHeight := m.library.height + 2
			pickerHeight, _ := m.filePickerLayout(panelHeight)
			m.filePicker.SetHeight(pickerHeight)
			if !strings.Contains(ansi.Strip(m.filePicker.View()), m.filePicker.Cursor) {
				m.filePicker, _ = m.filePicker.Update(tea.KeyMsg{Type: tea.KeyUp})
			}
		}
		if didSelect, path := m.filePicker.DidSelectDisabledFile(msg); didSelect {
			m.err = os.ErrPermission
			log.Printf("File type not allowed for selection: %s", path)
		}
		return m, cmdPicker
	}

	if m.state == kindleState {
		if m.library.activeArea == int(sideFocus) {
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				switch keyMsg.String() {
				case "ctrl+l":
					m.library.activeArea = int(contentFocus)
					return m, nil
				case "enter":
					selectedOption := m.library.MenuOptions[m.library.sideBarCursor]
					switch selectedOption {
					case "Home":
						m.state = homeState
						return m, tea.ClearScreen
					case "Books", "To-Be Read":
						m.state = librayState
						m.library.activeArea = int(contentFocus)
						return m, m.library.SetView(selectedOption)
					case "Add Book":
						m.state = fileState
						m.library.activeArea = int(contentFocus)
						return m, tea.Batch(tea.ClearScreen, m.filePicker.Init())
					case "Synchronize Kindle", "Synchronize \nKindle":
						m.state = kindleState
						m.library.activeArea = int(contentFocus)
						return m, tea.Batch(tea.ClearScreen, m.loadKindleBooksCmd())
					}
				}
			}
			newLib, cmd := m.library.Update(msg)
			m.library = newLib.(*Model)
			return m, cmd
		}

		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.library.activeArea = int(sideFocus)
				return m, nil
			case "up", "k":
				if m.kindleCursor > 0 {
					m.kindleCursor--
				}
				return m, nil
			case "down", "j":
				if m.kindleCursor < len(m.kindleBooks)-1 {
					m.kindleCursor++
				}
				return m, nil
			case "i", "I":
				m.kindleSelect = !m.kindleSelect
				if !m.kindleSelect {
					m.kindlePicked = make(map[string]struct{})
				}
				return m, nil
			case " ", "enter":
				if m.kindleSelect && len(m.kindleBooks) > 0 {
					name := m.kindleBooks[m.kindleCursor]
					if _, ok := m.kindlePicked[name]; ok {
						delete(m.kindlePicked, name)
					} else {
						m.kindlePicked[name] = struct{}{}
					}
				}
				return m, nil
			case "s":
				if m.kindleSyncing {
					return m, nil
				}
				var selected []string
				if m.kindleSelect {
					if len(m.kindlePicked) == 0 {
						m.kindleStatus = "No books selected"
						return m, nil
					}
					selected = make([]string, 0, len(m.kindlePicked))
					for name := range m.kindlePicked {
						selected = append(selected, name)
					}
				}
				m.kindleSyncing = true
				m.kindleLoader = false
				m.kindleStatus = ""
				return m, tea.Batch(
					m.kindleSyncCmd(m.kindleDocsURI, selected),
					tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
						return kindleSyncDelayMsg{}
					}),
				)
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "l", "L":
			if m.state == homeState {
				m.state = librayState
				m.library.sideBarCursor = 1
				m.library.activeArea = int(contentFocus)
				return m, m.library.SetView("Books")
			}
		case "t", "T":
			if m.state == homeState {
				m.state = librayState
				m.library.sideBarCursor = 2
				m.library.activeArea = int(contentFocus)
				return m, m.library.SetView("To-Be Read")
			}
		case "a", "A":
			if m.state == homeState {
				m.state = fileState
				m.library.sideBarCursor = 3
				m.library.activeArea = int(contentFocus)
				return m, m.filePicker.Init()
			}
		case "k", "K":
			if m.state == homeState {
				m.state = kindleState
				m.library.sideBarCursor = 4
				m.library.activeArea = int(contentFocus)
				return m, tea.Batch(tea.ClearScreen, m.loadKindleBooksCmd())
			}
		case "ctrl+h", "esc":
			if m.library.activeArea == int(contentFocus) {
				m.library.activeArea = int(sideFocus)
			}
		case "ctrl+l":
			if m.library.activeArea == int(sideFocus) {
				m.library.activeArea = int(contentFocus)
			}
		case "enter":
			if m.library.activeArea == int(sideFocus) {
				selectedOption := m.library.MenuOptions[m.library.sideBarCursor]
				switch selectedOption {
				case "Home":
					m.state = homeState
					return m, tea.ClearScreen
				case "Books", "To-Be Read":
					m.state = librayState
					m.library.activeArea = int(contentFocus)
					return m, m.library.SetView(selectedOption)
				case "Add Book":
					m.state = fileState
					m.library.activeArea = int(contentFocus)
					return m, tea.Batch(tea.ClearScreen, m.filePicker.Init())
				case "Synchronize Kindle", "Synchronize \nKindle":
					m.state = kindleState
					m.library.activeArea = int(contentFocus)
					return m, tea.Batch(tea.ClearScreen, m.loadKindleBooksCmd())
				}
			}
		}
	}

	if m.state == librayState || (m.state == fileState && m.library.activeArea == int(sideFocus)) {
		newLib, cmd := m.library.Update(msg)
		m.library = newLib.(*Model)
		return m, cmd
	}

	return m, nil
}

/* ----- Library ----- */
func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmdSync tea.Cmd
	var cmdPaginator tea.Cmd
	var cmds []tea.Cmd
	skipPaginatorUpdate := false
	if m.showRatingInput {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				m.showRatingInput = false
				m.ratingInput.Blur()
				m.ratingInput.Reset()
				return m, nil
			case "enter":
				ratingText := strings.TrimSpace(m.ratingInput.Value())
				if ratingText != "" {
					rating, err := strconv.ParseFloat(ratingText, 64)
					if err != nil || rating > 5.0 || rating < 0.0 {
						m.ratingInput.Reset()
						m.ratingInput.Placeholder = "Invalid!"
						return m, nil
					}
					if err := m.handler.UpdateBookRating(rating, m.books[m.cursor].BookFile); err != nil {
						log.Printf("Error trying to update rating: %v", err)
					} else {
						m.books[m.cursor].Rating = rating
					}
				}
				m.showRatingInput = false
				m.ratingInput.Blur()
				m.ratingInput.Reset()
				m.ratingInput.Placeholder = "0.0-5.0"
				return m, tea.ClearScreen
			}
		}
		var cmdRating tea.Cmd
		m.ratingInput, cmdRating = m.ratingInput.Update(msg)
		return m, cmdRating
	}

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		cmds = append(cmds, tea.ClearScreen)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			skipPaginatorUpdate = true
			if m.activeArea == int(contentFocus) {
				pageStart, _ := m.paginator.GetSliceBounds(len(m.books))
				if m.cursor > 0 {
					if m.cursor == pageStart && !m.paginator.OnFirstPage() {
						m.paginator.PrevPage()
						prevStart, prevEnd := m.paginator.GetSliceBounds(len(m.books))
						if prevEnd > prevStart {
							m.cursor = prevEnd - 1
						} else {
							m.cursor--
						}
						cmdSync = m.syncVisibleWidget()
						cmds = append(cmds, tea.ClearScreen)
					} else {
						m.cursor--
					}
				}
			}
			if m.activeArea == int(sideFocus) {
				if m.sideBarCursor > 0 {
					m.sideBarCursor--
				}
			}
		case "down", "j":
			skipPaginatorUpdate = true
			if m.activeArea == int(contentFocus) {
				pageStart, pageEnd := m.paginator.GetSliceBounds(len(m.books))
				if m.cursor < len(m.books)-1 {
					if m.cursor == pageEnd-1 && !m.paginator.OnLastPage() {
						m.paginator.NextPage()
						nextStart, _ := m.paginator.GetSliceBounds(len(m.books))
						if nextStart >= pageStart {
							m.cursor = nextStart
						} else {
							m.cursor++
						}
						cmdSync = m.syncVisibleWidget()
						cmds = append(cmds, tea.ClearScreen)
					} else {
						m.cursor++
					}
				}
			}
			if m.activeArea == int(sideFocus) {
				if m.sideBarCursor < len(m.MenuOptions)-1 {
					m.sideBarCursor++
				}
			}
		case "right", "l":
			skipPaginatorUpdate = true
			if !m.paginator.OnLastPage() {
				m.paginator.NextPage()
				nextStart, _ := m.paginator.GetSliceBounds(len(m.books))
				m.cursor = nextStart
				cmdSync = m.syncVisibleWidget()
				cmds = append(cmds, tea.ClearScreen)
			}
		case "left", "h":
			skipPaginatorUpdate = true
			if !m.paginator.OnFirstPage() {
				m.paginator.PrevPage()
				prevStart, _ := m.paginator.GetSliceBounds(len(m.books))
				m.cursor = prevStart
				cmdSync = m.syncVisibleWidget()
				cmds = append(cmds, tea.ClearScreen)
			}
		case "r", "R":
			err := m.handler.UpdateBookStatus("Read", m.books[m.cursor].BookFile)
			if err != nil {
				log.Printf("Error trying to update status: %v", err)
			}
			m.books[m.cursor].Status = "Read"
		case "u", "U":
			err := m.handler.UpdateBookStatus("Unread", m.books[m.cursor].BookFile)
			if err != nil {
				log.Printf("Error trying to update status: %v", err)
			}
			m.books[m.cursor].Status = "Unread"
		case "t", "T":
			err := m.handler.UpdateBookStatus("To Be Read", m.books[m.cursor].BookFile)
			if err != nil {
				log.Printf("Error trying to update status: %v", err)
			}
			m.books[m.cursor].Status = "To Be Read"
		case "s":
			m.showRatingInput = true
			m.ratingInput.Reset()
			m.ratingInput.Focus()
			return m, tea.ClearScreen
		}
	case coversLoadedMsg:
		m.covers = msg
		return m, nil
	case coverLoadedMsg:
		delete(m.coverRenderPending, msg.key)
		m.coverRenderCache[msg.key] = msg.data
		m.covers[msg.index] = msg.data
		return m, nil
	}

	if !skipPaginatorUpdate {
		m.paginator, cmdPaginator = m.paginator.Update(msg)
	}
	cmds = append(cmds, cmdPaginator, cmdSync)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.cols <= 0 || m.dynamicCardWidth <= 0 {
		return "\n  Initializing Kindria..."
	}
	var b strings.Builder

	m.start, m.end = m.paginator.GetSliceBounds(len(m.books))
	booksCards := make([]string, 0)
	type coverRender struct {
		row  int
		col  int
		data string
	}
	coverRenders := make([]coverRender, 0)

	for i := range m.books[m.start:m.end] {
		absoluteIndex := i + m.start
		cover, _ := m.covers[absoluteIndex]

		style := list.
			Width(m.dynamicCardWidth).
			Height(m.dynamicCardHeight)

		if absoluteIndex == m.cursor {
			if m.activeArea == int(contentFocus) {
				style = style.BorderForeground(highlight)
			}
		}

		if m.showRatingInput && absoluteIndex == m.cursor {
			popupContext := "Enter rating:\n" + m.ratingInput.View()
			popupBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(highlight).
				Padding(0, 1).
				Render(popupContext)
			cardContent := lipgloss.Place(m.dynamicCardWidth, m.dynamicCardHeight, lipgloss.Center, lipgloss.Center, popupBox)
			booksCards = append(booksCards, style.Render(cardContent))
			continue
		}

		booksCards = append(booksCards, style.Render(""))
		if cover != "" {
			rowIdx := i / m.cols
			colIdx := i % m.cols
			coverRenders = append(coverRenders, coverRender{
				row:  rowIdx,
				col:  colIdx,
				data: cover,
			})
		}
	}

	var rows []string
	for i := 0; i < len(booksCards); i += m.cols {
		endIdx := i + m.cols
		if endIdx > len(booksCards) {
			endIdx = len(booksCards)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, booksCards[i:endIdx]...))
	}
	sideWidth := m.sideBarWidth
	libraryBorderStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true, true, true, true).
		BorderForeground(subtle).
		Width(m.width - sideWidth - 4).Height(m.height - m.lowBarHeight).PaddingLeft(2).
		PaddingTop(1)
	if m.activeArea == int(contentFocus) {
		libraryBorderStyle = libraryBorderStyle.BorderForeground(borders)
	}

	book := lipgloss.JoinVertical(lipgloss.Top, rows...)
	books := lipgloss.JoinHorizontal(lipgloss.Top, book)
	library := libraryBorderStyle.Render(books)
	contentSide := (lipgloss.JoinVertical(lipgloss.Bottom, library, m.lowBarView()))
	b.WriteString(contentSide)
	b.WriteString("\n  " + m.paginator.View())
	rendered := b.String()

	cardWidth := m.dynamicCardWidth
	cardHeight := m.dynamicCardHeight
	if len(booksCards) > 0 {
		cardWidth = lipgloss.Width(booksCards[0])
		cardHeight = lipgloss.Height(booksCards[0])
	}
	contentStartRow := 1
	gridStartRow := contentStartRow + 2
	gridStartCol := sideWidth + 1 + 3
	coverRowOffset := 1
	coverColOffset := 3

	var overlay strings.Builder
	for _, c := range coverRenders {
		row := gridStartRow + (c.row * cardHeight) + coverRowOffset
		col := gridStartCol + (c.col * cardWidth) + coverColOffset

		overlay.WriteString("\x1b[")
		overlay.WriteString(strconv.Itoa(row))
		overlay.WriteString(";")
		overlay.WriteString(strconv.Itoa(col))
		overlay.WriteString("H")
		overlay.WriteString(c.data)
	}

	base := rendered
	if len(coverRenders) > 0 {
		base = rendered + overlay.String()
	}
	return base
}

func (m *Model) syncVisibleWidget() tea.Cmd {
	m.start, m.end = m.paginator.GetSliceBounds(len(m.books))
	if m.end <= m.start {
		return nil
	}
	booksToLoad := m.books[m.start:m.end]
	curWidth := m.dynamicCardWidth
	curHeight := m.dynamicCardHeight
	curCellPixelWidth := m.cellPixelWidth
	curCellPixelHeight := m.cellPixelHeight
	protocol := termimg.DetectProtocol()
	features := termimg.QueryTerminalFeatures()
	targetPixelWidth := curWidth
	targetPixelHeight := curHeight
	if curCellPixelWidth > 0 && curCellPixelHeight > 0 {
		targetPixelWidth = curWidth * curCellPixelWidth
		targetPixelHeight = curHeight * curCellPixelHeight
	} else if features != nil && features.FontWidth > 0 && features.FontHeight > 0 {
		targetPixelWidth = curWidth * features.FontWidth
		targetPixelHeight = curHeight * features.FontHeight
	}
	if targetPixelWidth <= 0 {
		targetPixelWidth = curWidth
	}
	if targetPixelHeight <= 0 {
		targetPixelHeight = curHeight
	}

	cmds := make([]tea.Cmd, 0, len(booksToLoad))
	for i, book := range booksToLoad {
		absoluteIndex := i + m.start
		path, err := m.handler.SelectBookPath(book.BookFile)
		if err != nil || path == "" {
			continue
		}
		cacheKey := fmt.Sprintf("%s|%s|%dx%d|%dx%d|%v", book.BookFile, path, curWidth, curHeight, targetPixelWidth, targetPixelHeight, protocol)
		if cached, ok := m.coverRenderCache[cacheKey]; ok {
			m.covers[absoluteIndex] = cached
			continue
		}
		if _, pending := m.coverRenderPending[cacheKey]; pending {
			continue
		}
		m.coverRenderPending[cacheKey] = struct{}{}

		idx := absoluteIndex
		coverPath := path
		key := cacheKey
		cmds = append(cmds, func() tea.Msg {
			srcImage, err := imaging.Open(coverPath)
			if err != nil {
				return coverLoadedMsg{index: idx, key: key, data: ""}
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
			if cover == nil {
				return coverLoadedMsg{index: idx, key: key, data: ""}
			}
			coverRendered, err := cover.Render()
			if err != nil {
				log.Printf("Err rendering cover: %v ", err)
				return coverLoadedMsg{index: idx, key: key, data: ""}
			}
			return coverLoadedMsg{index: idx, key: key, data: coverRendered}
		})
	}

	return tea.Batch(cmds...)
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

func (m *MainModel) SideBarView() string {
	var options string
	renderedOptionsList := make([]string, len(m.library.MenuOptions))

	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true, true, true, true).
		BorderForeground(subtle).Width(m.sideBarWidth).Height(m.library.height + 2)

	itemWidth := m.sideBarWidth - 2
	if itemWidth < 0 {
		itemWidth = 0
	}
	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(240)).
		Width(itemWidth).
		PaddingLeft(1).
		MarginBottom(1)
	activeStyle := inactiveStyle.Foreground(lipgloss.Color("#7D56F4"))

	if m.library.activeArea == int(sideFocus) {
		style = style.BorderForeground(borders)
	}

	for i, word := range m.library.MenuOptions {
		prefix := "  "
		if i == m.library.sideBarCursor {
			prefix = "> "
		}
		text := prefix + utils.ToSansBold(word)
		if i == m.library.sideBarCursor {
			renderedOptionsList[i] = activeStyle.Render(text)
		} else {
			renderedOptionsList[i] = inactiveStyle.Render(text)
		}
	}

	items := lipgloss.JoinVertical(lipgloss.Left, renderedOptionsList...)
	options = style.Render(items)
	return options
}

func (m *Model) lowBarView() string {
	contentWidth := m.contentWidth + 2
	info, err := m.handler.SelectBookInfo()
	if err != nil {
		log.Print("No book selected")
	}
	selectedBook := info[m.cursor]
	titleLabel := lipgloss.NewStyle().Foreground(normal).Bold(true).Render("Title:")
	authorLabel := lipgloss.NewStyle().Foreground(normal).Bold(true).Render("Author:")
	genresLabel := lipgloss.NewStyle().Foreground(normal).Bold(true).Render("Genres:")
	ratingLabel := lipgloss.NewStyle().Foreground(normal).Bold(true).Render("Rating:")
	statusLabel := lipgloss.NewStyle().Foreground(normal).Bold(true).Render("Status:")

	title := titleLabel + " " + selectedBook.Metadata.Title
	author := authorLabel + " " + selectedBook.Metadata.Author
	genres := strings.Join(selectedBook.Metadata.Genres, ", ")
	status := statusLabel + " " + selectedBook.Status
	ratingValue := strconv.FormatFloat(selectedBook.Rating, 'f', 1, 64)

	innerWidth := contentWidth - 4
	columnGap := 2
	columnWidth := (innerWidth-columnGap)/3 + 1

	leftCol := lipgloss.NewStyle().Width(columnWidth).Render(
		title + "\n\n" + genresLabel + " " + genres,
	)
	medCol := lipgloss.NewStyle().Width(columnWidth).Render(
		author + "\n\n" + status + " ",
	)
	stars := utils.GetStarRating(selectedBook.Rating)
	rightCol := lipgloss.NewStyle().Width(columnWidth).Render(
		ratingLabel + " " + ratingValue + "\n\n" + stars,
	)

	finalString := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, strings.Repeat(" ", columnGap), medCol, strings.Repeat(" ", columnGap-1), rightCol)
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true, true, true, true).
		BorderForeground(subtle).
		Foreground(normal).
		Width(contentWidth).
		Height(m.lowBarHeight)

	if m.activeArea == int(contentFocus) {
		style = style.BorderForeground(borders)
	}

	return style.Render(finalString)
}

func (m *Model) SetView(option string) tea.Cmd {
	switch option {
	case "Books":
		m.books = m.allBooks
	case "To-Be Read":
		var filtered []*metadata.Package
		for _, b := range m.allBooks {
			if b.Status == "To Be Read" {
				filtered = append(filtered, b)
			}
		}
		m.books = filtered
	}

	m.paginator.SetTotalPages(len(m.books))
	m.paginator.Page = 0
	m.cursor = 0
	m.covers = make(map[int]string)
	m.coverRenderPending = make(map[string]struct{})
	return tea.Batch(tea.ClearScreen, m.syncVisibleWidget())
}

func (m *MainModel) FilePickerView() string {
	sidebarView := m.SideBarView()
	panelWidth := m.library.width - m.sideBarWidth - 4
	panelHeight := m.library.height + 2
	if panelWidth < 24 {
		panelWidth = 24
	}
	if panelHeight < 12 {
		panelHeight = 12
	}
	filePickerStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true, true, true, true).
		BorderForeground(subtle).
		Width(panelWidth).
		Height(panelHeight)
	if m.library.activeArea == int(contentFocus) {
		filePickerStyle = filePickerStyle.BorderForeground(borders)
	}
	var s strings.Builder
	pickerHeight, start := m.filePickerLayout(panelHeight)
	picker := m.filePicker
	picker.SetHeight(pickerHeight)
	pickerWidth := panelWidth - 2
	if pickerWidth < 10 {
		pickerWidth = 10
	}
	pickerView := truncateViewLines(picker.View(), pickerWidth)

	s.WriteString("  ")
	if m.err != nil {
		s.WriteString(m.filePicker.Styles.DisabledFile.Render("Error: " + m.err.Error()))
		s.WriteString("\n  ")
	}
	s.WriteString("Directory: " + m.filePicker.CurrentDirectory)
	s.WriteString("\n  Press i to edit directory path, Enter to select an .epub file")
	if m.showFileInput {
		s.WriteString("\n\n  " + m.fileInput.View())
	}
	s.WriteString("\n\n  Pick one or more files:")
	s.WriteString("\n\n" + pickerView + "\n")
	s.WriteString("\n  Books to insert:")
	if m.importing && m.showLoader {
		s.WriteString("\n    Inserting books...")
	} else if len(m.selectedOrder) == 0 {
		s.WriteString("\n    (none selected)")
	} else {
		if start > 0 {
			s.WriteString("\n    ... and " + strconv.Itoa(start) + " more")
		}
		maxNameLen := panelWidth - 20
		if maxNameLen < 18 {
			maxNameLen = 18
		}
		for i := start; i < len(m.selectedOrder); i++ {
			name := filepath.Base(m.selectedOrder[i])
			name = ansi.Truncate(name, maxNameLen, "...")
			s.WriteString("\n    " + strconv.Itoa(i+1) + ". " + name)
		}
	}
	if m.importStatus != "" {
		s.WriteString("\n\n  " + m.importStatus)
	}
	contentWidth := panelWidth - 2
	if contentWidth < 10 {
		contentWidth = 10
	}
	contentHeight := panelHeight
	if contentHeight < 6 {
		contentHeight = 6
	}
	viewContent := truncateBlockHeight(truncateViewLines(s.String(), contentWidth), contentHeight)
	fileView := filePickerStyle.Render(viewContent)
	setupView := lipgloss.JoinHorizontal(lipgloss.Left, sidebarView, fileView)
	return setupView
}

func (m *MainModel) KindleView() string {
	sidebarView := m.SideBarView()
	panelWidth := m.library.width - m.sideBarWidth - 4
	panelHeight := m.library.height + 2
	if panelWidth < 24 {
		panelWidth = 24
	}
	if panelHeight < 12 {
		panelHeight = 12
	}
	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true, true, true, true).
		BorderForeground(subtle).
		Width(panelWidth).
		Height(panelHeight)
	if m.library.activeArea == int(contentFocus) {
		style = style.BorderForeground(borders)
	}

	var s strings.Builder
	s.WriteString("  Kindle sync\n")
	if m.kindleDocsURI != "" {
		s.WriteString("  Source: " + m.kindleDocsURI + "\n")
	}
	s.WriteString("  s: Synchronize all books\n")
	s.WriteString("  i: Select which books synchronize\n")
	if m.kindleSelect {
		s.WriteString("  space/enter: toggle selection\n")
	}
	s.WriteString("\n  Books found:\n")

	if m.kindleSyncing && m.kindleLoader {
		s.WriteString("\n    Synchronizing Kindle books...\n")
	} else if len(m.kindleBooks) == 0 {
		s.WriteString("\n    (no supported books found)\n")
	} else {
		for i, name := range m.kindleBooks {
			prefix := "    "
			if i == m.kindleCursor {
				prefix = "  > "
			}
			if m.kindleSelect {
				if _, ok := m.kindlePicked[name]; ok {
					prefix += "[x] "
				} else {
					prefix += "[ ] "
				}
			}
			s.WriteString(prefix + ansi.Truncate(name, panelWidth-8, "...") + "\n")
		}
	}
	if m.kindleStatus != "" {
		s.WriteString("\n  " + m.kindleStatus + "\n")
	}

	content := truncateBlockHeight(truncateViewLines(s.String(), panelWidth-2), panelHeight)
	return lipgloss.JoinHorizontal(lipgloss.Left, sidebarView, style.Render(content))
}

func (m *MainModel) filePickerLayout(panelHeight int) (int, int) {
	const minPickerHeight = 6
	headerLines := 5
	if m.showFileInput {
		headerLines += 2
	}
	maxVisibleSelected := panelHeight - headerLines - minPickerHeight - 7
	if maxVisibleSelected < 1 {
		maxVisibleSelected = 1
	}
	visibleSelected := len(m.selectedOrder)
	start := 0
	if visibleSelected > maxVisibleSelected {
		start = visibleSelected - maxVisibleSelected
		visibleSelected = maxVisibleSelected
	}
	selectedLines := 2
	if visibleSelected == 0 {
		selectedLines += 1
	} else {
		selectedLines += visibleSelected
		if start > 0 {
			selectedLines++
		}
	}
	pickerHeight := panelHeight - headerLines - selectedLines - 4
	if pickerHeight < minPickerHeight {
		pickerHeight = minPickerHeight
	}
	return pickerHeight, start
}

func truncateViewLines(s string, width int) string {
	if width <= 0 {
		return ""
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, width, "...")
	}
	return strings.Join(lines, "\n")
}

func truncateBlockHeight(s string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= maxLines {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[:maxLines], "\n")
}

func (m *MainModel) addSelectedFile(path string) {
	if path == "" {
		return
	}
	if _, exists := m.selectedFiles[path]; exists {
		return
	}
	m.selectedFiles[path] = struct{}{}
	m.selectedOrder = append(m.selectedOrder, path)
}

func (m *MainModel) importBooksCmd(selected []string) tea.Cmd {
	handler := m.library.handler
	return func() tea.Msg {
		booksFolder, err := os.ReadDir("./books")
		if err != nil {
			return importFinishedMsg{err: err}
		}
		existingNames := make(map[string]struct{}, len(booksFolder))
		for _, b := range booksFolder {
			existingNames[b.Name()] = struct{}{}
		}

		successfulCopies := make([]string, 0, len(selected))
		failedBooks := make([]string, 0)
		duplicateCount := 0
		for _, book := range selected {
			filename := filepath.Base(book)
			exist, err := handler.CheckBookExist(filename)
			if err != nil {
				failedBooks = append(failedBooks, book)
				continue
			}
			if exist != 0 {
				duplicateCount++
				continue
			}
			if _, exists := existingNames[filename]; exists {
				continue
			}
			if err := utils.CopyFile(book, "./books/"+filename); err != nil {
				failedBooks = append(failedBooks, book)
				continue
			}
			existingNames[filename] = struct{}{}
			successfulCopies = append(successfulCopies, book)
		}

		if len(successfulCopies) == 0 {
			return importFinishedMsg{
				successfulCopies: successfulCopies,
				failedBooks:      failedBooks,
				duplicateCount:   duplicateCount,
			}
		}

		if _, err := handler.InsertBooks(); err != nil {
			return importFinishedMsg{
				successfulCopies: successfulCopies,
				failedBooks:      failedBooks,
				duplicateCount:   duplicateCount,
				err:              err,
			}
		}
		books, err := handler.SelectBooks()
		if err != nil {
			return importFinishedMsg{
				successfulCopies: successfulCopies,
				failedBooks:      failedBooks,
				duplicateCount:   duplicateCount,
				err:              err,
			}
		}
		return importFinishedMsg{
			successfulCopies: successfulCopies,
			failedBooks:      failedBooks,
			duplicateCount:   duplicateCount,
			refreshedBooks:   books,
		}
	}
}

func (m *MainModel) loadKindleBooksCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		docsURI, books, err := kindle.ScanKindleBooks(ctx)
		return kindleBooksLoadedMsg{
			docsURI: docsURI,
			books:   books,
			err:     err,
		}
	}
}

func (m *MainModel) kindleSyncCmd(docsURI string, selected []string) tea.Cmd {
	handler := m.library.handler
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		res, err := kindle.KindleExtract(ctx, &handler, docsURI, selected)
		return kindleSyncFinishedMsg{
			inserted:      res.Inserted,
			failed:        res.Failed,
			duplicated:    res.Duplicated,
			refreshedBook: res.Refreshed,
			err:           err,
		}
	}
}
