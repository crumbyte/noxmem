package samples

import (
	"os"
	"strconv"
	"strings"

	"github.com/crumbyte/noxmem/internal/render/format"
	"github.com/crumbyte/noxmem/internal/render/table"
	"github.com/crumbyte/noxmem/pkg/pprofx"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type TraceStyles struct {
	LineText     lipgloss.Style
	FileName     lipgloss.Style
	FunctionName lipgloss.Style
}

type TraceTable struct {
	styles    TraceStyles
	columns   table.Columns
	locations pprofx.Locations
	dataTable table.Model
	sampleID  pprofx.SampleID
	width     int
	height    int
}

func NewTraceTable(t table.Model) *TraceTable {
	t.Blur()

	return &TraceTable{
		dataTable: t,
		columns: table.Columns{
			{Title: "dwdw", Width: 0},
			{Title: "Line", WidthRatio: 0.07},
			{Title: "Function Name", MinWidth: 30, WidthRatio: 0.295},
			{Title: "Location", Full: true},
		},
	}
}

func (tt *TraceTable) Init() tea.Cmd {
	return nil
}

func (tt *TraceTable) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		tt.width, tt.height = max(0, msg.Width), max(0, msg.Height)

		tt.dataTable.SetWidth(tt.width)
		tt.dataTable.SetHeight(tt.height)
	case BlurSamples:
		tt.dataTable.SetCursor(0)
		tt.dataTable.Blur()
	case FocusSamples:
		tt.dataTable.Focus()
	}

	tt.dataTable, cmd = tt.dataTable.Update(msg)

	return tt, cmd
}

func (tt *TraceTable) View() tea.View {
	return tt.renderSampleData()
}

func (tt *TraceTable) SetStyles(s TraceStyles) {
	tt.styles = s
}

// Cursor returns a current table's cursor.
func (tt *TraceTable) Cursor() int {
	return tt.dataTable.Cursor()
}

func (tt *TraceTable) Focus() {
	tt.dataTable.Focus()
}

func (tt *TraceTable) Blur() {
	tt.dataTable.Blur()
}

func (tt *TraceTable) SelectedFilename() string {
	row := tt.dataTable.SelectedRow()
	if row == nil || len(row.Cols) < 1 {
		return ""
	}

	return strings.ReplaceAll(row.Cols[0], "/", string(os.PathSeparator))
}

// SetLocations sets a new list of sample locations. The next View call will
// render the table with a new set of sample locations. The new sample locations
// must be set explicitly each time to update the table's content.
func (tt *TraceTable) SetLocations(l pprofx.Locations) {
	tt.locations = l
}

// SetHeight sets a new height for the model. The provided height will also be
// updated for the table model. If the height was not set explicitly, it will
// be taken from the current window size.
func (tt *TraceTable) SetHeight(height int) {
	tt.height = height

	tt.dataTable.SetHeight(tt.height)
}

func (tt *TraceTable) SetSampleID(id pprofx.SampleID) {
	tt.sampleID = id
}

func (tt *TraceTable) renderSampleData() tea.View {
	columns := tt.columns.TableColumns(tt.width-4, table.NoSort)

	tt.dataTable.SetColumns(columns)

	if len(tt.locations) == 0 || len(tt.sampleID) == 0 {
		return tea.View{}
	}

	selectedSample := tt.locations[tt.sampleID].Locations

	rows := make([]table.Row, 0, len(selectedSample))

	for _, location := range selectedSample {
		row := table.Row{
			Cols: []string{
				location.Line[0].Function.Filename,
				strconv.Itoa(int(location.Line[0].Line)),
				FunctionNameSuffix(
					format.PrefixWrapString(
						location.Line[0].Function.Name, '/', columns[2].Width-3,
					),
					tt.styles.FunctionName,
				),
				FilepathSuffix(
					format.PrefixWrapString(
						location.Line[0].Function.Filename, '/', columns[3].Width-3,
					),
					tt.styles.FileName,
				),
			},
		}

		rows = append(rows, row)
	}

	tt.dataTable.SetRows(rows)

	return tt.dataTable.View()
}
