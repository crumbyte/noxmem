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

type SampleStyles struct {
	LineText           lipgloss.Style
	FileName           lipgloss.Style
	FunctionName       lipgloss.Style
	GoroutineState     lipgloss.Style
	BytesStyleResolver func(suffix string) *lipgloss.Style
}

type MemorySamples struct {
	Sample

	columns   table.Columns
	sortState table.SortState
}

func NewMemorySamples(t table.Model, refreshRate time.Duration) *MemorySamples {
	return &MemorySamples{
		Sample: Sample{table: t},
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

func (ms *MemorySamples) View() tea.View {
	return ms.renderAllocSamples()
}

func (ms *MemorySamples) SetSortKey(key string) {
	sortKey := table.SortKey(key)

	if sortKey != HeapSort && sortKey != AllocSort && sortKey != ObjectsSort {
		return
	}

	if ms.sortState.Key == sortKey {
		ms.sortState.Desc = !ms.sortState.Desc

		return
	}

	ms.sortState.Key = sortKey
}

func (ms *MemorySamples) renderAllocSamples() tea.View {
	if cachedView, skip := ms.SkipRender(); skip {
		return cachedView
	}

	locations := ms.locationsDelta.Current()

	// prepare the table columns to fit the width of the window, adjusting for
	// horizontal spacing between cells.
	columns := ms.columns.TableColumns(ms.width-len(ms.columns), ms.sortState)

	ms.table.SetColumns(columns)

	rows := make([]table.Row, 0, len(locations))

	sortedLocations, _ := locations.Sort(
		ms.sortState.Desc, ms.resolveSortType(),
	)

	prevLocations := ms.locationsDelta.Prev()

	for _, loc := range sortedLocations {
		// skip the current sample if it has no locations
		if len(loc.Locations) == 0 {
			continue
		}

		allocRate := loc.MemorySample.Alloc

		if prevLoc, ok := prevLocations[loc.SampleID]; ok {
			allocRate = loc.MemorySample.Alloc - prevLoc.MemorySample.Alloc
		}

		locationLine := loc.Locations[0].Line[0]

		allocLine := ms.styles.LineText.Render(
			strconv.Itoa(int(locationLine.Line)),
		)

		faint := lipgloss.NewStyle().Faint(true)

		row := table.Row{
			Cols: []string{
				loc.SampleID.String(),
				format.BytesSizeColor(
					loc.MemorySample.InUse, 10, ms.styles.BytesStyleResolver,
				),
				format.BytesSizeColor(
					loc.MemorySample.Alloc, 10, ms.styles.BytesStyleResolver,
				),
				format.BytesSizeColor(
					allocRate, 10, ms.styles.BytesStyleResolver,
				),
				faint.Render(strconv.Itoa(loc.TotalSamples)),
				faint.Render(strconv.Itoa(loc.MemorySample.InUseObjects)),
				allocLine,
				FunctionNameSuffix(
					format.PrefixWrapString(
						locationLine.Function.Name,
						'/',
						columns[len(columns)-2].Width-3,
					),
					ms.styles.FunctionName,
				),
				FilepathSuffix(
					format.PrefixWrapString(
						locationLine.Function.Filename,
						'/',
						columns[len(columns)-1].Width-3,
					),
					ms.styles.FileName,
				),
			},
		}

		rows = append(rows, row)
	}

	ms.table.SetRows(rows)

	ms.cachedView = ms.table.View()

	if ms.scheduleCache {
		ms.useCache = true
	}

	return ms.cachedView
}

func (ms *MemorySamples) resolveSortType() pprofx.SortType {
	sortMap := map[table.SortKey]pprofx.SortType{
		HeapSort:    pprofx.HeapSort,
		AllocSort:   pprofx.AllocSort,
		ObjectsSort: pprofx.ObjectsSort,
	}

	if sortType, ok := sortMap[ms.sortState.Key]; ok {
		return sortType
	}

	return pprofx.NoSort
}
