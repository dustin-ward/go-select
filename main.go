package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var GO_DIR string

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4).PaddingRight(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).PaddingRight(2).Foreground(lipgloss.Color("#ff5500"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4)
    borderStyle = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())
)

type versionInfo struct {
	name    string
	version string
}

var SELECTED versionInfo

func (v versionInfo) FilterValue() string { return v.name }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(versionInfo)
	if !ok {
		return
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	str := fmt.Sprintf("%s - %s", i.name, i.version)
	fmt.Fprint(w, fn(str))
}

type (
	selectMsg struct{}
	errMsg    struct{ error }
)

type model struct {
	quitting bool
	chosen   bool
	choice   int
	list     list.Model
	err      error
}

func initModel(versionList []list.Item) *model {
	l := list.New(versionList, itemDelegate{}, 40, 10)
	l.Title = "Select Go Version"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	return &model{
		false,
		false,
		0,
		l,
		nil,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		k := msg.String()
		if k == "ctrl+c" || k == "q" || k == "esc" {
			m.quitting = true
			return m, tea.Quit
		}
		if k == "enter" {
			m.chosen = true
			SELECTED = m.list.SelectedItem().(versionInfo)
			return m, selectVersion
		}

	case selectMsg:
		m.quitting = true
		return m, tea.Quit

	case errMsg:
		m.err = msg
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %s", m.err)
	}
	return borderStyle.Render(m.list.View())
}

func selectVersion() tea.Msg {
	cmd := fmt.Sprintf("export GOROOT=%s/%s", GO_DIR, SELECTED.name)
	err := os.WriteFile(fmt.Sprintf("%s/selected", GO_DIR), []byte(cmd), 0666)
	if err != nil {
		return errMsg{err}
	}

	return selectMsg{}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("No directory provided...\nUsage: %s <directory-containing-go-installs>", os.Args[0])
	}
	GO_DIR = os.Args[1]

	f, err := os.Open(GO_DIR)
	if err != nil {
		log.Fatalf("Unable to open directory %s: %v", GO_DIR, err)
	}

	files, err := f.Readdir(0)
	if err != nil {
		log.Fatal("Readdir:", err)
	}

	versionList := make([]list.Item, 0)
	for _, v := range files {
		if v.IsDir() {
			b, err := os.ReadFile(fmt.Sprintf("%s/%s/VERSION", GO_DIR, v.Name()))
			if err != nil {
				b = []byte("unknown")
			}
			versionString := string(b)

			// Only get first line
			n := len(versionString)
			if i := strings.IndexByte(versionString, byte('\n')); i != -1 {
				n = i
			}
			versionList = append(versionList, versionInfo{
				name:    v.Name(),
				version: versionString[:n],
			})
		}
	}

	if len(versionList) == 0 {
		log.Fatal("No Go installations found...")
	}

    // Reverse sorted because I assume the newer version will be selected most...
	sort.Slice(versionList, func(i, j int) bool {
		a, ok1 := versionList[i].(versionInfo)
		b, ok2 := versionList[j].(versionInfo)
		if !ok1 || !ok2 {
			log.Fatal("Unable to sort...")
		}
		return a.name > b.name
	})

	m := initModel(versionList)

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
    print("\n")
}
