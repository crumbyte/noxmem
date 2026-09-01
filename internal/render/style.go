package render

import (
	"fmt"
	"image/color"
	"sync"

	"charm.land/lipgloss/v2"
)

var (
	styleOnce sync.Once
	style     *Style
)

type Style struct {
	cache map[string]*lipgloss.Style
	cs    ColorSchema
}

func InitStyle(cs ColorSchema) *Style {
	styleOnce.Do(func() {
		style = &Style{
			cs:    cs,
			cache: make(map[string]*lipgloss.Style),
		}
	})

	return style
}

func (s *Style) CS() *ColorSchema {
	return &s.cs
}

func (s *Style) StatusBarStyle() *lipgloss.Style {
	cv, ok := s.cache["statusBar"]
	if !ok {
		cs := lipgloss.NewStyle().Margin(1, 0, 1, 0)

		s.cache["statusBar"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) GCPauseGraphStyle() *lipgloss.Style {
	cv, ok := s.cache["gcPauseGraph"]
	if !ok {
		cs := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(s.cs.GCPauseGraph))

		s.cache["gcPauseGraph"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) InlineBlockContentStyle() *lipgloss.Style {
	cv, ok := s.cache["inlineBlockContent"]
	if !ok {
		cs := lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1)

		s.cache["inlineBlockContent"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) InlineBlockStyle() *lipgloss.Style {
	cv, ok := s.cache["inlineBlock"]
	if !ok {
		cs := lipgloss.NewStyle().
			Align(lipgloss.Left).
			PaddingLeft(1).
			PaddingBottom(1).
			PaddingTop(1)

		s.cache["inlineBlock"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) BlockBorderStyle() *lipgloss.Style {
	cv, ok := s.cache["blockBorder"]
	if !ok {
		cs := lipgloss.NewStyle().
			Foreground(lipgloss.Color(s.cs.BlockBorder))

		s.cache["blockBorder"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) BlockBorderTextStyle() *lipgloss.Style {
	cv, ok := s.cache["blockBorderText"]
	if !ok {
		cs := lipgloss.NewStyle().
			Foreground(lipgloss.Color(s.cs.BlockBorderText))

		s.cache["blockBorderText"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) StatTitleTextStyle() *lipgloss.Style {
	cv, ok := s.cache["statTitleText"]
	if !ok {
		cs := lipgloss.NewStyle().
			Foreground(lipgloss.Color(s.cs.StatTitleText))

		s.cache["statTitleText"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) StatTextStyle() *lipgloss.Style {
	cv, ok := s.cache["statText"]
	if !ok {
		cs := lipgloss.NewStyle().
			Foreground(lipgloss.Color(s.cs.StatText))

		s.cache["statText"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) ReduceDeltaStyle() *lipgloss.Style {
	cv, ok := s.cache["reduceDelta"]
	if !ok {
		cs := lipgloss.NewStyle().
			Foreground(lipgloss.Color(s.cs.ReduceDeltaText))

		s.cache["reduceDelta"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) GrowDeltaStyle() *lipgloss.Style {
	cv, ok := s.cache["growDelta"]
	if !ok {
		cs := lipgloss.NewStyle().
			Foreground(lipgloss.Color(s.cs.GrowDeltaText))

		s.cache["growDelta"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) TableBorder() *lipgloss.Style {
	cv, ok := s.cache["tableBorder"]
	if !ok {
		cs := lipgloss.NewStyle().
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color(s.cs.TableHeaderBorder))

		s.cache["tableBorder"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) TableHeader() *lipgloss.Style {
	cv, ok := s.cache["tableHeader"]
	if !ok {
		cs := lipgloss.NewStyle().
			Inherit(*s.TableBorder()).
			Bold(true).
			Foreground(lipgloss.Color(s.cs.TableHeaderText)).
			PaddingRight(1).
			BorderBottom(true)

		s.cache["tableHeader"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) TableCell() *lipgloss.Style {
	cv, ok := s.cache["tableCell"]
	if !ok {
		cs := lipgloss.NewStyle().
			Foreground(lipgloss.Color(style.CS().CellText)).
			PaddingRight(1).
			Align(lipgloss.Right)

		s.cache["tableCell"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) SelectedRow() *lipgloss.Style {
	cv, ok := s.cache["selectedRow"]
	if !ok {
		cs := lipgloss.NewStyle().
			Foreground(lipgloss.Color(s.cs.SelectedRowText)).
			Background(lipgloss.Color(s.cs.SelectedRowBG)).
			PaddingRight(1).
			Bold(false)

		s.cache["selectedRow"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) MarkedRow() *lipgloss.Style {
	cv, ok := s.cache["markedRow"]
	if !ok {
		cs := lipgloss.NewStyle().
			Foreground(lipgloss.Color(s.cs.MarkedRowText)).
			Background(lipgloss.Color(s.cs.MarkedRowBG)).
			PaddingRight(1).
			Bold(false)

		s.cache["markedRow"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) SamplesFileSuffixStyle() *lipgloss.Style {
	cv, ok := s.cache["samplesFileSuffix"]
	if !ok {
		cs := lipgloss.NewStyle().
			Foreground(lipgloss.Color(s.cs.Samples.FileName))

		s.cache["samplesFileSuffix"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) Help() *lipgloss.Style {
	cv, ok := s.cache["help"]
	if !ok {
		cs := lipgloss.NewStyle().Foreground(lipgloss.Color(s.cs.HelpText))

		s.cache["help"] = &cs

		return &cs
	}

	return cv
}

func (s *Style) BindKey() *lipgloss.Style {
	cv, ok := s.cache["bindKey"]
	if !ok {
		cs := lipgloss.NewStyle().Foreground(lipgloss.Color(s.cs.BindingText))

		s.cache["bindKey"] = &cs

		return &cs
	}

	return cv
}

func StatusBarBlockStyle(bgColor color.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFDF5")).
		Background(bgColor).
		Padding(0, 1)
}

func (s *Style) SizeUnit(unit string) *lipgloss.Style {
	ck := fmt.Sprintf("sizeUnit-%s", unit)
	cv, ok := s.cache[ck]
	if !ok {
		sizeUnitColorsMap := map[string]string{
			"B":  s.cs.SizeUnit.B,
			"KB": s.cs.SizeUnit.KB,
			"MB": s.cs.SizeUnit.MB,
			"GB": s.cs.SizeUnit.GB,
			"TB": s.cs.SizeUnit.TB,
			"PB": s.cs.SizeUnit.PB,
			"EB": s.cs.SizeUnit.EB,
		}
		cs := lipgloss.NewStyle().Foreground(
			lipgloss.Color(sizeUnitColorsMap[unit]),
		)

		s.cache[ck] = &cs

		return &cs
	}

	return cv
}
