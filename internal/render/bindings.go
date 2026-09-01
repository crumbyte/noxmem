package render

import (
	"github.com/crumbyte/noxmem/internal/render/table"

	"charm.land/bubbles/v2/key"
)

var Bindings KeyMap

type KeyMap struct {
	NavigationKeyMap table.KeyMap
	NameFilter       key.Binding
	SortKeys         key.Binding
	Explore          key.Binding
	SelectSample     key.Binding
	QuitTrace        key.Binding
	Quit             key.Binding
	style            *Style
}

func (km *KeyMap) ListBindings() []key.Binding {
	return []key.Binding{
		km.NavigationKeyMap.LineUp,
		km.NavigationKeyMap.LineDown,
		km.NavigationKeyMap.GotoTop,
		km.NavigationKeyMap.GotoBottom,
		km.SelectSample,
		km.QuitTrace,
		km.SortKeys,
		km.Explore,
		km.Quit,
	}
}

//nolint:funlen
func DefaultKeyMap(s *Style) KeyMap {
	return KeyMap{
		style: s,
		NavigationKeyMap: table.KeyMap{
			LineUp: key.NewBinding(
				key.WithKeys("up", "k"),
				key.WithHelp(
					s.BindKey().Render("↑/k"),
					s.Help().Render("up"),
				),
			),
			LineDown: key.NewBinding(
				key.WithKeys("down", "j"),
				key.WithHelp(
					s.BindKey().Render("↓/j"),
					s.Help().Render("down"),
				),
			),
			GotoTop: key.NewBinding(
				key.WithKeys("home", "g"),
				key.WithHelp(
					s.BindKey().Render("g/home"),
					s.Help().Render("go to start"),
				),
			),
			GotoBottom: key.NewBinding(
				key.WithKeys("end", "G"),
				key.WithHelp(
					s.BindKey().Render("G/end"),
					s.Help().Render("go to end"),
				),
			),
		},
		SelectSample: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp(
				s.BindKey().Render("enter"),
				s.Help().Render("select sample"),
			),
		),
		QuitTrace: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp(
				s.BindKey().Render("esc"),
				s.Help().Render("unselect sample"),
			),
		),
		SortKeys: key.NewBinding(
			key.WithKeys("1", "2", "3"),
			key.WithHelp(
				s.BindKey().Render("1/2/3"),
				s.Help().Render("sort inuse/total/objects"),
			),
		),
		NameFilter: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp(
				s.BindKey().Render("ctrl+f"),
				s.Help().Render("toggle sample filter"),
			),
		),
		Explore: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp(
				s.BindKey().Render("e"),
				s.Help().Render("open file"),
			),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp(
				s.BindKey().Render("q/ctrl+c"),
				s.Help().Render("quit"),
			),
		),
	}
}

func InitKeyMap(s *Style) {
	Bindings = DefaultKeyMap(s)
}
