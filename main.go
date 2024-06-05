package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

const maxWidth = 80

var (
	red    = lipgloss.AdaptiveColor{Light: "#FE5F86", Dark: "#FE5F86"}
	indigo = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7571F9"}
	green  = lipgloss.AdaptiveColor{Light: "#02BA84", Dark: "#02BF87"}
)

type Styles struct {
	Base,
	HeaderText,
	Status,
	StatusHeader,
	Highlight,
	ErrorHeaderText,
	Help lipgloss.Style
}

func NewStyles(lg *lipgloss.Renderer) *Styles {
	s := Styles{}
	s.Base = lg.NewStyle().
		Padding(1, 4, 0, 1)
	s.HeaderText = lg.NewStyle().
		Foreground(indigo).
		Bold(true).
		Padding(0, 1, 0, 2)
	s.Status = lg.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(indigo).
		PaddingLeft(1).
		MarginTop(1)
	s.StatusHeader = lg.NewStyle().
		Foreground(green).
		Bold(true)
	s.Highlight = lg.NewStyle().
		Foreground(lipgloss.Color("212"))
	s.ErrorHeaderText = s.HeaderText.
		Foreground(red)
	s.Help = lg.NewStyle().
		Foreground(lipgloss.Color("240"))
	return &s
}

type input struct {
	key      string
	required bool
	valType  string
}

type Model struct {
	lg     *lipgloss.Renderer
	styles *Styles
	form   *huh.Form
	width  int
	inputs []input
}

func NewModel() Model {
	m := Model{width: maxWidth}
	m.lg = lipgloss.DefaultRenderer()
	m.styles = NewStyles(m.lg)

	m.inputs = []input{
		{key: "key", required: true, valType: "string"},
		{key: "name", required: true, valType: "string"},
		{key: "description", required: false, valType: "string"},
		{key: "includeInSnippet", required: false, valType: "boolean"},
	}

	fields := make([]huh.Field, 0)
	for _, i := range m.inputs {
		var field huh.Field
		switch i.valType {
		case "boolean":
			field = huh.NewSelect[string]().
				Key(i.key).
				Options(huh.NewOptions("true", "false")...).
				Title(i.key + " ").
				Inline(true)

		default:
			field = huh.NewInput().
				Key(i.key).
				Title(i.key + " ").
				Prompt("").
				Inline(true).
				Validate(func(t string) error {
					if t == "" {
						return fmt.Errorf("%s is a required field", i.key)
					}
					return nil
				})
		}

		fields = append(fields, field)
	}

	fields = append(fields, huh.NewConfirm().
		Key("done").
		Title("All done?").
		Validate(func(v bool) error {
			if !v {
				return fmt.Errorf("Welp, finish up then")
			}
			return nil
		}).
		Affirmative("Yep").
		Negative("Wait, no"),
	)

	m.form = huh.NewForm(
		huh.NewGroup(fields...),
	).
		WithWidth(45).
		WithShowHelp(false).
		WithShowErrors(false)
	return m
}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = min(msg.Width, maxWidth) - m.styles.Base.GetHorizontalFrameSize()
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd

	// Process the form
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	s := m.styles

	switch m.form.State {
	case huh.StateCompleted:

		var b strings.Builder
		data := m.getData()
		dataJson, _ := json.MarshalIndent(data, "", "  ")
		fmt.Fprintf(&b, string(dataJson))
		return s.Status.Margin(0, 1).Padding(1, 2).Width(48).Render(b.String()) + "\n\n"
	default:
		v := strings.TrimSuffix(m.form.View(), "\n\n")
		form := m.lg.NewStyle().Margin(1, 0).Render(v)

		errors := m.form.Errors()
		header := m.appBoundaryView("{Create} a {thing}}")
		if len(errors) > 0 {
			header = m.appErrorBoundaryView(m.errorView())
		}
		body := lipgloss.JoinHorizontal(lipgloss.Top, form)

		footer := m.appBoundaryView(m.form.Help().ShortHelpView(m.form.KeyBinds()))
		if len(errors) > 0 {
			footer = m.appErrorBoundaryView("")
		}

		return s.Base.Render(header + "\n" + body + "\n\n" + footer)
	}
}

func (m Model) errorView() string {
	var s string
	for _, err := range m.form.Errors() {
		s += err.Error()
	}
	return s
}

func (m Model) appBoundaryView(text string) string {
	return lipgloss.PlaceHorizontal(
		m.width,
		lipgloss.Left,
		m.styles.HeaderText.Render(text),
		lipgloss.WithWhitespaceChars("/"),
		lipgloss.WithWhitespaceForeground(indigo),
	)
}

func (m Model) appErrorBoundaryView(text string) string {
	return lipgloss.PlaceHorizontal(
		m.width,
		lipgloss.Left,
		m.styles.ErrorHeaderText.Render(text),
		lipgloss.WithWhitespaceChars("/"),
		lipgloss.WithWhitespaceForeground(red),
	)
}

func (m Model) getData() map[string]any {
	log.Println(m.form)

	formData := make(map[string]any)
	for _, i := range m.inputs {
		log.Println(i.key)
		inputValue := m.form.GetString(i.key)
		log.Println(inputValue)
		if inputValue != "" {
			var val any
			switch i.valType {
			case "string":
				val = inputValue
			case "array":
				val = strings.Split(inputValue, ",")
			case "boolean":
				val, _ = strconv.ParseBool(inputValue)
				// TODO: handle error
			case "integer":
				val, _ = strconv.Atoi(inputValue)
				// TODO: handle error
			case "object":
				// TODO
			}
			formData[i.key] = val
		}
	}

	return formData
}

func main() {
	f, err := tea.LogToFile("debug.log", "")
	if err != nil {
		fmt.Println("could not open file for debuggin", err)
		os.Exit(1)
	}
	defer f.Close()
	_, err = tea.NewProgram(NewModel(), tea.WithAltScreen()).Run()
	if err != nil {
		fmt.Println("Oh no:", err)
		os.Exit(1)
	}
}
