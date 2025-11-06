package main

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")) // Neon cyan
	vaultDir    string
)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	currentFile            *os.File
	newFileInput           textinput.Model
	noteTextArea           textarea.Model
	list                   list.Model
	showingList            bool
	createFileInputVisible bool
	width, height          int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msgType := msg.(type) {
	case tea.KeyMsg:
		switch msgType.String() {
		case "esc":
			if m.createFileInputVisible {
				m.createFileInputVisible = false
				m.newFileInput.SetValue("")
			}

			if m.currentFile != nil {
				m.noteTextArea.SetValue("")
				m.currentFile = nil
			}

			if m.showingList {
				if m.list.FilterState() == list.Filtering {
					break
				}
				m.showingList = false
			}

			return m, nil
		case "ctrl+c", "ctrl+q":
			return m, tea.Quit

		case "ctrl+n":
			if m.currentFile != nil {
				break
			}
			m.createFileInputVisible = true
			m.noteTextArea.Blur()
			return m, nil

		case "ctrl+l":
			noteLists := listFiles()
			m.list.SetItems(noteLists)
			m.showingList = true
			return m, nil

		case "ctrl+s":
			if m.currentFile == nil {
				break
			}

			if err := m.currentFile.Truncate(0); err != nil {
				fmt.Println("Unable to save the file :(")
				return m, nil
			}

			if _, err := m.currentFile.Seek(0, 0); err != nil {
				fmt.Println("Unable to save the file :(")
				return m, nil
			}

			if _, err := m.currentFile.WriteString(m.noteTextArea.Value()); err != nil {
				fmt.Println("Unable to save the file :(")
				return m, nil
			}

			if err := m.currentFile.Close(); err != nil {
				fmt.Println("Unable to close the file :(")
			}

			m.currentFile = nil
			m.noteTextArea.SetValue("")
			return m, nil

		case "ctrl+d":
			if m.showingList && len(m.list.Items()) > 0 {
				if item, ok := m.list.SelectedItem().(item); ok {
					filePath := fmt.Sprintf("%s/%s", vaultDir, item.title)
					if err := os.Remove(filePath); err != nil {
						log.Printf("Error deleting file: %v", err)
						return m, nil
					}
					noteList := listFiles()
					m.list.SetItems(noteList)

					if len(noteList) > 0 {
						m.list.Select(0)
					}
				}
			}
			return m, nil

		case "enter":
			if m.currentFile != nil {
				break
			}

			if m.showingList {
				item, ok := m.list.SelectedItem().(item)
				if ok {
					filePath := fmt.Sprintf("%s/%s", vaultDir, item.title)
					content, err := os.ReadFile(filePath)
					if err != nil {
						log.Printf("Error reading file : %v\n", err)
						return m, nil
					}
					m.noteTextArea.SetValue(string(content))
					file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
					if err != nil {
						log.Printf("Error reading file : %v\n", err)
						return m, nil
					}
					m.showingList = false
					m.currentFile = file
					m.noteTextArea.Focus()
				}
				return m, nil
			}

			filename := m.newFileInput.Value()
			if filename != "" {
				filePath := fmt.Sprintf("%s/%s", vaultDir, filename)

				if _, err := os.Stat(filePath); err == nil {
					file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
					if err != nil {
						log.Fatal("Unable to open file :(")
					}
					content, _ := os.ReadFile(filePath)
					m.noteTextArea.SetValue(string(content))
					m.createFileInputVisible = false
					m.currentFile = file
					m.noteTextArea.Focus()

					return m, nil
				}
				f, err := os.Create(filePath)
				if err != nil {
					log.Fatal(err)
				}
				m.currentFile = f
				m.createFileInputVisible = false
				m.newFileInput.SetValue("")
				m.noteTextArea.Focus()
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msgType.Width
		m.height = msgType.Height

		// Reserve top lines for header/help
		headerHeight := 8
		usableHeight := m.height - headerHeight
		if usableHeight < 10 {
			usableHeight = 10
		}

		usableWidth := m.width - 4
		if usableWidth < 20 {
			usableWidth = 20
		}

		// Update textarea size dynamically
		m.noteTextArea.SetHeight(usableHeight)
		m.noteTextArea.SetWidth(usableWidth - 2)

	}

	if m.createFileInputVisible {
		m.newFileInput, cmd = m.newFileInput.Update(msg)
	}

	if m.currentFile != nil {
		m.noteTextArea, cmd = m.noteTextArea.Update(msg)
	}

	if m.showingList {
		m.list, cmd = m.list.Update(msg)
	}

	return m, cmd
}

func (m model) View() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF")).
		Background(lipgloss.Color("#111133")).
		Padding(1, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF00FF")).
		MarginBottom(1).
		MarginTop(3)

	welcome := headerStyle.Render("🚀 Totion — Futuristic Notes 🧠")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7DF9FF")).
		Italic(true).
		MarginBottom(1).
		MarginLeft(2).
		MarginTop(4)

	help := helpStyle.Render("Ctrl+N: new | Ctrl+S: save | Ctrl+L: list | Esc: back | Ctrl+Q: quit")

	// Dynamically calculate space for textarea/list
	headerHeight := 5 // roughly header + help + spacing
	usableHeight := m.height - headerHeight
	if usableHeight < 10 {
		usableHeight = 10
	}

	// Make textarea/list nearly full width (keep 4-column padding)
	usableWidth := m.width - 4
	if usableWidth < 20 {
		usableWidth = 20
	}

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#39FF14")).
		Background(lipgloss.Color("#0A0F25")).
		Padding(1, 2).
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("#FF00FF")).
		MarginLeft(2).
		MarginBottom(1)
		// Width(usableWidth)

	view := ""

	switch {
	case m.createFileInputVisible:
		view = inputStyle.Render(m.newFileInput.View())

	case m.showingList:
		m.list.SetSize(usableWidth-4, usableHeight-4)
		listStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF00FF")).
			Padding(1, 2).
			MarginLeft(2).
			Width(usableWidth).
			Height(usableHeight)
		view = listStyle.Render(m.list.View())

	case m.currentFile != nil:
		editorStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00FFFF")).
			Padding(1, 2).
			MarginLeft(2).
			Width(usableWidth).
			Height(usableHeight)
		view = editorStyle.Render(m.noteTextArea.View())
	}

	content := fmt.Sprintf("%s\n%s\n%s\n", welcome, help, view)
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, content)
}

func initializeModel() model {
	err := os.MkdirAll(vaultDir, 0750)
	if err != nil {
		log.Fatal(err)
	}

	// textinput
	ti := textinput.New()
	ti.Placeholder = "✨ Name your new file..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50
	ti.Cursor.Style = cursorStyle
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#39FF14"))

	// textarea
	ta := textarea.New()
	ta.Placeholder = "💡 Start writing your note..."
	ta.ShowLineNumbers = false
	ta.Focus()
	ta.CharLimit = 2048
	ta.Cursor.Style = cursorStyle
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color("#39FF14"))

	//list
	noteList := listFiles()
	list := list.New(noteList, list.NewDefaultDelegate(), 40, 20)
	list.Title = "All Notes 📁"
	list.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("ctrl+d"),
				key.WithHelp("ctrl+d", "delete note"),
			),
		}
	}

	return model{
		newFileInput:           ti,
		noteTextArea:           ta,
		list:                   list,
		createFileInputVisible: false,
		width:                  80,
		height:                 24,
	}
}

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	vaultDir = fmt.Sprintf("%s/.totion", homeDir)
}

func main() {
	p := tea.NewProgram(initializeModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas! There's been an error: %v\n", err)
		os.Exit(1)
	}
}

func listFiles() []list.Item {
	items := make([]list.Item, 0)

	entries, err := os.ReadDir(vaultDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			info, _ := entry.Info()

			modificationTime := info.ModTime().Format("02-01-2006 15:04:05")

			items = append(items, item{
				title: info.Name(),
				desc:  fmt.Sprintf("Last modified : %s", modificationTime),
			})
		}
	}

	return items
}
