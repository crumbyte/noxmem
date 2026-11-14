package samples

import (
	"strconv"

	"github.com/crumbyte/noxmem/internal/render/format"
	"github.com/crumbyte/noxmem/internal/render/table"
	"github.com/crumbyte/noxmem/pkg/pprofx"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// A group of sort keys for goroutine samples table. Each sort key defines a
// binding key for a sortable column in the table.
const (
	GoroutinesIDSort    table.SortKey = "1"
	GoroutinesStateSort table.SortKey = "2"
)

// GoroutineSamples renders a list of goroutine samples in the form of a table.
// Each row represents a goroutine source and its metadata, such as ID and state.
// The samples table does not show a full goroutine trace, but only the actual
// function call.
type GoroutineSamples struct {
	Sample

	columns   table.Columns
	sortState table.SortState
}

func NewGoroutinesTable(t table.Model) *GoroutineSamples {
	return &GoroutineSamples{
		Sample: Sample{table: t},
		columns: table.Columns{
			{Title: "Sample ID"},
			{Title: "ID", WidthRatio: 0.05, SortKey: GoroutinesIDSort},
			{Title: "State", WidthRatio: 0.095, SortKey: GoroutinesStateSort},
			{Title: "Waiting", WidthRatio: 0.05},
			{Title: "Created By ID", WidthRatio: 0.07},
			{Title: "Samples", WidthRatio: 0.05},
			{Title: "Line", WidthRatio: 0.05},
			{Title: "Function Name", MinWidth: 30, WidthRatio: 0.25},
			{Title: "Location", Full: true},
		},
		sortState: table.SortState{
			Key:  GoroutinesIDSort,
			Desc: false,
		},
	}
}

func (gs *GoroutineSamples) View() tea.View {
	return gs.renderAllocSamples()
}

// SetSortKey sets a new sort key or changes the sorting order in case of the
// same key. All unknown sorting keys will be ignored.
func (gs *GoroutineSamples) SetSortKey(key string) {
	sortKey := table.SortKey(key)

	if sortKey != GoroutinesStateSort && sortKey != GoroutinesIDSort {
		return
	}

	if gs.sortState.Key == sortKey {
		gs.sortState.Desc = !gs.sortState.Desc

		return
	}

	gs.sortState.Key = sortKey
}

func (gs *GoroutineSamples) renderAllocSamples() tea.View {
	if cachedView, skip := gs.SkipRender(); skip {
		return cachedView
	}

	locations := gs.locationsDelta.Current()

	// prepare the table columns to fit the width of the window, adjusting for
	// horizontal spacing between cells.
	columns := gs.columns.TableColumns(gs.width-len(gs.columns), gs.sortState)

	gs.table.SetColumns(columns)

	rows := make([]table.Row, 0, len(locations))

	sortedLocations, _ := locations.Sort(
		gs.sortState.Desc, gs.resolveSortType(),
	)

	for _, loc := range sortedLocations {
		// skip the current sample if it has no locations
		if len(loc.Locations) == 0 {
			continue
		}

		createdByID := "-"

		if len(loc.GoroutineSample.CreatedByID) > 0 {
			createdByID = loc.GoroutineSample.CreatedByID
		}

		locationLine := loc.Locations[0].Line[0]

		allocLine := gs.styles.LineText.Render(
			strconv.Itoa(int(locationLine.Line)),
		)

		faint := lipgloss.NewStyle().Faint(true)

		row := table.Row{
			Cols: []string{
				loc.SampleID.String(),
				strconv.Itoa(loc.GoroutineSample.ID),
				gs.styles.GoroutineState.Render(
					loc.GoroutineSample.State.String(),
				),
				loc.GoroutineSample.Wait,
				faint.Render(createdByID),
				faint.Render(strconv.Itoa(loc.TotalSamples)),
				allocLine,
				FunctionNameSuffix(
					format.PrefixWrapString(
						locationLine.Function.Name,
						'/',
						columns[len(columns)-2].Width-3,
					),
					gs.styles.FunctionName,
				),
				FilepathSuffix(
					format.PrefixWrapString(
						locationLine.Function.Filename,
						'/',
						columns[len(columns)-1].Width-3,
					),
					gs.styles.FileName,
				),
			},
		}

		rows = append(rows, row)
	}

	gs.table.SetRows(rows)

	gs.cachedView = gs.table.View()

	if gs.scheduleCache {
		gs.useCache = true
	}

	return gs.cachedView
}

func (gs *GoroutineSamples) resolveSortType() pprofx.SortType {
	sortMap := map[table.SortKey]pprofx.SortType{
		GoroutinesIDSort:    pprofx.GoroutineIDSort,
		GoroutinesStateSort: pprofx.GoroutineStateSort,
	}

	if sortType, ok := sortMap[gs.sortState.Key]; ok {
		return sortType
	}

	return pprofx.NoSort
}
