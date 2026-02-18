package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/parsely/parsely/internal/ai"
	"github.com/parsely/parsely/internal/core"
	"github.com/parsely/parsely/internal/db"
)

type view int

const (
	viewMenu view = iota
	viewInput
	viewLoading
	viewList
	viewResults
)

type inputMode int

const (
	inputModeFilePath inputMode = iota
	inputModeExportPath
)

// processResultMsg carries the result of an async document processing operation
type processResultMsg struct {
	result *core.ProcessingResult
	err    error
}

type model struct {
	view       view
	cursor     int
	processor  *core.Processor
	vocabulary []*db.Vocabulary
	result     *core.ProcessingResult
	err        error
	input      textinput.Model
	inputMode  inputMode
	spinner    spinner.Model
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	menuStyle = lipgloss.NewStyle().
			Padding(1, 2)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	normalStyle = lipgloss.NewStyle()

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)
)

func initialModel() model {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: ANTHROPIC_API_KEY environment variable not set")
		os.Exit(1)
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "parsely.db"
	}

	language := os.Getenv("LANGUAGE")
	if language == "" {
		language = "auto-detect"
	}

	database, err := db.NewDatabase(dbPath)
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}

	aiClient, err := ai.NewClaudeClient(apiKey)
	if err != nil {
		fmt.Printf("Error initializing AI client: %v\n", err)
		os.Exit(1)
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		view:      viewMenu,
		processor: core.NewProcessor(database, aiClient, language),
		input:     textinput.New(),
		spinner:   s,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case processResultMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.result = msg.result
		}
		m.view = viewResults
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.view == viewMenu {
				return m, tea.Quit
			}
			// Return to menu from other views
			m.view = viewMenu
			m.cursor = 0
			m.err = nil
			m.input.Reset()
			return m, nil

		case "up", "k":
			if m.view == viewMenu && m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.view == viewMenu && m.cursor < 3 {
				m.cursor++
			}

		case "enter":
			switch m.view {
			case viewMenu:
				return m.handleMenuSelection()
			case viewInput:
				return m.handleInputSubmission()
			case viewResults, viewList:
				m.view = viewMenu
				m.cursor = 0
			}
		}

	}

	// Handle text input when in input view
	if m.view == viewInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) handleMenuSelection() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 0: // Parse new document
		m.view = viewInput
		m.inputMode = inputModeFilePath
		m.input.Placeholder = "Enter file path (PDF or DOCX)"
		m.input.Focus()
		return m, textinput.Blink

	case 1: // View all vocabulary
		vocab, err := m.processor.GetVocabularyList()
		if err != nil {
			m.err = err
		} else {
			m.vocabulary = vocab
		}
		m.view = viewList

	case 2: // Export to JSON
		m.view = viewInput
		m.inputMode = inputModeExportPath
		m.input.Placeholder = "Enter export file path (default: vocabulary_export.json)"
		m.input.Focus()
		return m, textinput.Blink

	case 3: // Exit
		return m, tea.Quit
	}

	return m, nil
}

func (m model) handleInputSubmission() (tea.Model, tea.Cmd) {
	inputValue := m.input.Value()
	m.input.Reset()

	switch m.inputMode {
	case inputModeFilePath:
		m.view = viewLoading
		m.err = nil
		processCmd := func() tea.Msg {
			result, err := m.processor.ProcessDocument(inputValue)
			return processResultMsg{result: result, err: err}
		}
		return m, tea.Batch(processCmd, m.spinner.Tick)

	case inputModeExportPath:
		if inputValue == "" {
			inputValue = "vocabulary_export.json"
		}

		err := m.processor.ExportVocabulary(inputValue)
		if err != nil {
			m.err = err
		} else {
			m.result = &core.ProcessingResult{}
		}
		m.view = viewResults
	}

	return m, nil
}

func (m model) View() string {
	switch m.view {
	case viewMenu:
		return m.renderMenu()
	case viewInput:
		return m.renderInput()
	case viewLoading:
		return m.renderLoading()
	case viewList:
		return m.renderVocabularyList()
	case viewResults:
		return m.renderResults()
	}
	return m.renderMenu()
}

func (m model) renderMenu() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Parsely - Language Learning Tool"))
	s.WriteString("\n\n")

	menuItems := []string{
		"Parse new document",
		"View all vocabulary",
		"Export to JSON",
		"Exit",
	}

	for i, item := range menuItems {
		if m.cursor == i {
			s.WriteString(selectedStyle.Render("> " + item))
		} else {
			s.WriteString(normalStyle.Render("  " + item))
		}
		s.WriteString("\n")
	}

	s.WriteString("\n\n")
	s.WriteString("Use ↑/↓ arrows or j/k to navigate, Enter to select, q to quit")

	return menuStyle.Render(s.String())
}

func (m model) renderLoading() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Parsely - Language Learning Tool"))
	s.WriteString("\n\n")
	s.WriteString(m.spinner.View())
	s.WriteString(" Extracting vocabulary with AI...")
	s.WriteString("\n\n")
	s.WriteString("This may take a moment depending on document size.")

	return menuStyle.Render(s.String())
}

func (m model) renderInput() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Parsely - Language Learning Tool"))
	s.WriteString("\n\n")

	s.WriteString(m.input.View())
	s.WriteString("\n\n")
	s.WriteString("Press Enter to submit, Ctrl+C to cancel")

	return menuStyle.Render(s.String())
}

func (m model) renderVocabularyList() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Vocabulary List"))
	s.WriteString("\n\n")

	if m.err != nil {
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	} else if len(m.vocabulary) == 0 {
		s.WriteString("No vocabulary items found.\n")
	} else {
		s.WriteString(fmt.Sprintf("Total items: %d\n\n", len(m.vocabulary)))
		for i, vocab := range m.vocabulary {
			if i >= 20 {
				s.WriteString(fmt.Sprintf("\n... and %d more items\n", len(m.vocabulary)-20))
				break
			}
			s.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, vocab.Text, vocab.Language))
		}
	}

	s.WriteString("\n\nPress Enter to return to menu")

	return menuStyle.Render(s.String())
}

func (m model) renderResults() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Results"))
	s.WriteString("\n\n")

	if m.err != nil {
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	} else if m.result != nil {
		if m.result.TotalProcessed > 0 {
			s.WriteString(successStyle.Render("Success!"))
			s.WriteString("\n\n")
			s.WriteString(fmt.Sprintf("New vocabulary added: %d\n", m.result.NewVocabulary))
			s.WriteString(fmt.Sprintf("Duplicates skipped: %d\n", m.result.SkippedDuplicates))
			s.WriteString(fmt.Sprintf("Total processed: %d\n", m.result.TotalProcessed))
			if m.result.Language != "" {
				s.WriteString(fmt.Sprintf("Language: %s\n", m.result.Language))
			}
		} else {
			s.WriteString(successStyle.Render("Export completed successfully!"))
		}
	}

	s.WriteString("\n\nPress Enter to return to menu")

	return menuStyle.Render(s.String())
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
