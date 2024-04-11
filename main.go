package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/dustin-ward/go-select/versions"

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

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(versions.Info)
	if !ok {
		return
	}

	var str string

	if index == m.Index() {
		str = selectedItemStyle.Render(fmt.Sprintf("> %s - %s", i.Name, i.Version))
	} else {
		str = fmt.Sprintf("%s %s", itemStyle.Render(i.Name), versionStyle.Render("- "+i.Version))
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
			v, _ := m.list.SelectedItem().(versions.Info)
			versions.SELECTED = &v
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
		if versions.SELECTED != nil {
			return fmt.Sprintf("Go version: %s (named: %s)\n", versions.SELECTED.Version, versions.SELECTED.Name)
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
	cmd := fmt.Sprintf("export GOROOT=%s/%s", GO_DIR, versions.SELECTED.Name)
	err := os.WriteFile(fmt.Sprintf("%s/selected", GO_DIR), []byte(cmd), 0666)
	if err != nil {
		return errMsg{err}
	}

	return selectMsg{}
}

const MAX_DEPTH = 1

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("No directory provided...\nUsage: %s <directory-containing-go-installs>", os.Args[0])
	}

	GO_DIR = os.Args[1]
	separators := strings.Count(GO_DIR, string(os.PathSeparator))

	// Walk directory provided from args to get all available go versions
	version_list := make([]list.Item, 0)
	err := filepath.WalkDir(GO_DIR, func(path string, info fs.DirEntry, err error) error {
		if info.IsDir() {
			// Skip depth > 1
			//TODO: Parameterize?
			if strings.Count(path, string(os.PathSeparator))-separators > MAX_DEPTH {
				return fs.SkipDir
			}

			// Only directories with 'go-build-zos/' are valid go versions
			var isGo bool = false
			if _, err := os.Stat(fmt.Sprintf("%s/go-build-zos", path)); err == nil {
				isGo = true
			}
			if _, err := os.Stat(fmt.Sprintf("%s/IBM_README.txt", path)); err == nil {
				isGo = true
			}
			if !isGo {
				return nil
			}

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

			version_list = append(version_list, versions.Info{
				Name:    info.Name(),
				Version: versionString[:n],
			})
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error searching '%s': %v", GO_DIR, err)
	}

	if len(version_list) == 0 {
		log.Fatal("No Go installations found...")
	}

	// Filepath/walk produces the dirs in lexicographical order... I assume I would want
	// the newest versions at the top of the list, so lets reverse the order
	slices.Reverse(version_list)

	// Start up bubbletea
	m := initModel(version_list)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
