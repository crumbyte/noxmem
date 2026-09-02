package samples

import (
	"strconv"
	"time"

	"github.com/crumbyte/noxmem/internal/render/format"
	"github.com/crumbyte/noxmem/internal/render/table"
	"github.com/crumbyte/noxmem/pkg/pprofx"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	HeapSort    table.SortKey = "1"
	AllocSort   table.SortKey = "2"
	ObjectsSort table.SortKey = "3"
)

type AllocationsStyles struct {
	LineText           lipgloss.Style
	FileName           lipgloss.Style
	FunctionName       lipgloss.Style
	BytesStyleResolver func(suffix string) *lipgloss.Style
}

type Allocations struct {
	styles         AllocationsStyles
	columns        table.Columns
	locationsDelta pprofx.Delta[pprofx.Locations, int]
	allocTable     table.Model
	sortState      table.SortState
	cachedView     tea.View
	width          int
	height         int
	useCache       bool
	scheduleCache  bool
}

func NewSamplesTable(t table.Model, refreshRate time.Duration) *Allocations {
	return &Allocations{
		allocTable: t,
		columns: table.Columns{
			{Title: "Sample ID"},
			{Title: "In Use", WidthRatio: 0.07, SortKey: HeapSort},
			{Title: "Total", WidthRatio: 0.07, SortKey: AllocSort},
			{Title: "Rate / " + refreshRate.String(), WidthRatio: 0.07},
			{Title: "Samples", WidthRatio: 0.05},
			{Title: "Objects", WidthRatio: 0.05, SortKey: ObjectsSort},
			{Title: "Line", WidthRatio: 0.05},
			{Title: "Function Name", MinWidth: 30, WidthRatio: 0.25},
			{Title: "Location", Full: true},
		},
		sortState: table.SortState{
			Key:  HeapSort,
			Desc: true,
		},
	}
}

func (a *Allocations) Init() tea.Cmd {
	return nil
}

func (a *Allocations) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = max(0, msg.Width), max(0, msg.Height)

		a.allocTable.SetWidth(a.width)
		a.allocTable.SetHeight(a.height)
	case BlurSamples:
		a.allocTable.MarkSelected()
		a.allocTable.Blur()

		defer func() { a.scheduleCache = true }()
	case FocusSamples:
		a.scheduleCache, a.useCache = false, false
		a.allocTable.ResetMarked()
		a.allocTable.Focus()
	}

	a.allocTable, cmd = a.allocTable.Update(msg)

	return a, cmd
}

func (a *Allocations) View() tea.View {
	return a.renderAllocSamples()
}

func (a *Allocations) SetStyles(s AllocationsStyles) {
	a.styles = s
}

// SetLocations sets a new list of sample locations. The next View call will
// render the table with a new set of sample locations. The new sample locations
// must be set explicitly each time to update the table's content.
func (a *Allocations) SetLocations(l pprofx.Locations) {
	a.locationsDelta.Update(l)
}

// SetHeight sets a new height for the model. The provided height will also be
// updated for the table model. If the height was not set explicitly, it will
// be taken from the current window size.
func (a *Allocations) SetHeight(height int) {
	a.height = height

	a.allocTable.SetHeight(a.height)
}

// SelectedRow returns a current selected table row according to its cursor.
func (a *Allocations) SelectedRow() *table.Row {
	return a.allocTable.SelectedRow()
}

// Cursor returns a current table's cursor.
func (a *Allocations) Cursor() int {
	return a.allocTable.Cursor()
}

// CurrentSampleID returns pprofx.SampleID according to the selected table's row.
// If no row is selected or the sample ID cannot be retrieved, an empty string
// will be returned.
func (a *Allocations) CurrentSampleID() pprofx.SampleID {
	selectedSample := a.allocTable.SelectedRow()

	if selectedSample == nil || len(selectedSample.Cols) == 0 {
		return ""
	}

	return pprofx.SampleID(selectedSample.Cols[0])
}

func (a *Allocations) SetSortKey(key string) {
	sortKey := table.SortKey(key)

	if sortKey != HeapSort && sortKey != AllocSort && sortKey != ObjectsSort {
		return
	}

	if a.sortState.Key == sortKey {
		a.sortState.Desc = !a.sortState.Desc

		return
	}

	a.sortState.Key = sortKey
}

func (a *Allocations) renderAllocSamples() tea.View {
	locations := a.locationsDelta.Current()
	if a.useCache {
		return a.cachedView
	}

	// skip rendering if the window width wasn't set yet or no samples were
	// retrieved.
	if a.width == 0 || len(locations) == 0 {
		return a.cachedView
	}

	// prepare the table columns to fit the width of the window, adjusting for
	// horizontal spacing between cells.
	columns := a.columns.TableColumns(a.width-len(a.columns), a.sortState)

	a.allocTable.SetColumns(columns)

	rows := make([]table.Row, 0, len(locations))

	sortedLocations, _ := locations.Sort(
		a.sortState.Desc, a.resolveSortType(),
	)

	prevLocations := a.locationsDelta.Prev()

	for _, loc := range sortedLocations {
		// skip the current sample if it has no locations
		if len(loc.Locations) == 0 {
			continue
		}

		allocRate := loc.Alloc

		if prevLoc, ok := prevLocations[loc.SampleID]; ok {
			allocRate = loc.Alloc - prevLoc.Alloc
		}

		locationLine := loc.Locations[0].Line[0]

		allocLine := a.styles.LineText.Render(
			strconv.Itoa(int(locationLine.Line)),
		)

		faint := lipgloss.NewStyle().Faint(true)

		row := table.Row{
			Cols: []string{
				loc.SampleID.String(),
				format.BytesSizeColor(loc.InUse, 10, a.styles.BytesStyleResolver),
				format.BytesSizeColor(loc.Alloc, 10, a.styles.BytesStyleResolver),
				format.BytesSizeColor(allocRate, 10, a.styles.BytesStyleResolver),
				faint.Render(strconv.Itoa(loc.TotalSamples)),
				faint.Render(strconv.Itoa(loc.InUseObjects)),
				allocLine,
				FunctionNameSuffix(
					format.PrefixWrapString(
						locationLine.Function.Name,
						'/',
						columns[len(columns)-2].Width-3,
					),
					a.styles.FunctionName,
				),
				FilepathSuffix(
					format.PrefixWrapString(
						locationLine.Function.Filename,
						'/',
						columns[len(columns)-1].Width-3,
					),
					a.styles.FileName,
				),
			},
		}

		rows = append(rows, row)
	}

	a.allocTable.SetRows(rows)

	a.cachedView = a.allocTable.View()

	if a.scheduleCache {
		a.useCache = true
	}

	return a.cachedView
}

func (a *Allocations) resolveSortType() pprofx.SortType {
	sortMap := map[table.SortKey]pprofx.SortType{
		HeapSort:    pprofx.HeapSort,
		AllocSort:   pprofx.AllocSort,
		ObjectsSort: pprofx.ObjectsSort,
	}

	if sortType, ok := sortMap[a.sortState.Key]; ok {
		return sortType
	}

	return pprofx.NoSort
}
