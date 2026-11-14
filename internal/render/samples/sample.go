package samples

import (
	"github.com/crumbyte/noxmem/internal/render/table"
	"github.com/crumbyte/noxmem/pkg/pprofx"

	tea "charm.land/bubbletea/v2"
)

type (
	BlurSamples  struct{}
	FocusSamples struct{}
)

// Sample defines a common set of functions for sample tables and must be embedded
// into the actual implementation.
type Sample struct {
	styles         SampleStyles
	locationsDelta pprofx.Delta[pprofx.Locations, int]
	table          table.Model
	cachedView     tea.View
	width          int
	height         int
	useCache       bool
	scheduleCache  bool
}

func (s *Sample) Init() tea.Cmd {
	return nil
}

func (s *Sample) View() tea.View {
	return tea.View{}
}

func (s *Sample) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width, s.height = max(0, msg.Width), max(0, msg.Height)

		s.table.SetWidth(s.width)
		s.table.SetHeight(s.height)
	case BlurSamples:
		s.table.MarkSelected()
		s.table.Blur()

		defer func() { s.scheduleCache = true }()
	case FocusSamples:
		s.scheduleCache, s.useCache = false, false
		s.table.ResetMarked()
		s.table.Focus()
	}

	s.table, cmd = s.table.Update(msg)

	return s, cmd
}

func (s *Sample) SetStyles(style SampleStyles) {
	s.styles = style
}

// SetLocations sets a new list of sample locations. The next View call will
// render the table with a new set of sample locations. The new sample locations
// must be set explicitly each time to update the table's content.
func (s *Sample) SetLocations(l pprofx.Locations) {
	s.locationsDelta.Update(l)
}

// SetHeight sets a new height for the model. The provided height will also be
// updated for the table model. If the height was not set explicitly, it will
// be taken from the current window size.
func (s *Sample) SetHeight(height int) {
	s.height = height

	s.table.SetHeight(s.height)
}

// SelectedRow returns a current selected table row according to its cursor.
func (s *Sample) SelectedRow() *table.Row {
	return s.table.SelectedRow()
}

// Cursor returns a current table's cursor.
func (s *Sample) Cursor() int {
	return s.table.Cursor()
}

// CurrentSampleID returns pprofx.SampleID according to the selected table's row.
// If no row is selected or the sample ID cannot be retrieved, an empty string
// will be returned.
func (s *Sample) CurrentSampleID() pprofx.SampleID {
	selectedSample := s.table.SelectedRow()

	if selectedSample == nil || len(selectedSample.Cols) == 0 {
		return ""
	}

	return pprofx.SampleID(selectedSample.Cols[0])
}

func (s *Sample) SkipRender() (tea.View, bool) {
	if s.useCache {
		return s.cachedView, true
	}

	locations := s.locationsDelta.Current()

	// skip rendering if the window width wasn't set yet or no samples were
	// retrieved.
	if s.width == 0 || len(locations) == 0 {
		return s.cachedView, true
	}

	return tea.View{}, false
}
