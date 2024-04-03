package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var GO_DIR string

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).Foreground(lipgloss.Color("#ff5500"))
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	versionStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).PaddingRight(2)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).PaddingRight(2).Foreground(lipgloss.Color("#ff5500"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	borderStyle       = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Width(30)
)

type versionInfo struct {
	name    string
	version string
}

var SELECTED *versionInfo

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

	var str string

	if index == m.Index() {
		str = selectedItemStyle.Render(fmt.Sprintf("> %s - %s", i.name, i.version))
	} else {
		str = fmt.Sprintf("%s %s", itemStyle.Render(i.name), versionStyle.Render("- "+i.version))
	}

	fmt.Fprint(w, str)
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

func initModel(version_list []list.Item) *model {
	l := list.New(version_list, itemDelegate{}, 30, 10)
	l.Title = "Select Go Version"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
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
			v, _ := m.list.SelectedItem().(versionInfo)
			SELECTED = &v
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
	if m.quitting {
		if SELECTED != nil {
			return fmt.Sprintf("Go version: %s (named: %s)\n", SELECTED.version, SELECTED.name)
		} else {
			return "No Go version selected"
		}
	}
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

	var max_width int

	// Walk directory provided from args to get all available go versions
	version_list := make([]list.Item, 0)
	err := filepath.Walk(GO_DIR, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			// Only directories with 'go-build-zos/' are valid go versions
			if _, err := os.Stat(fmt.Sprintf("%s/go-build-zos", path)); err == nil {

				// Read version number
				b, err := os.ReadFile(fmt.Sprintf("%s/VERSION", path))
				if err != nil {
					b = []byte("unknown")
				}
				versionString := string(b)

				// Only get first line
				n := len(versionString)
				if i := strings.IndexByte(versionString, byte('\n')); i != -1 {
					n = i
				}

				version_list = append(version_list, versionInfo{
					name:    info.Name(),
					version: versionString[:n],
				})

				// Format: "my_go_ver - go1.23.4"
				max_width = max(max_width, len(info.Name())+n+3)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error searching '%s': %v", GO_DIR, err)
	}

	if len(version_list) == 0 {
		log.Fatal("No Go installations found...")
	}

	// Reverse sorted because I assume the newer version will be selected most...
	sort.Slice(version_list, func(i, j int) bool {
		a, ok1 := version_list[i].(versionInfo)
		b, ok2 := version_list[j].(versionInfo)
		if !ok1 || !ok2 {
			log.Fatal("Unable to sort...")
		}
		return a.name > b.name
	})

	m := initModel(version_list)

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
