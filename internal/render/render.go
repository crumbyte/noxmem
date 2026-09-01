package render

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/crumbyte/noxmem/internal/render/chart"
	"github.com/crumbyte/noxmem/internal/render/samples"
	"github.com/crumbyte/noxmem/internal/render/table"
	"github.com/crumbyte/noxmem/pkg/explore"
	"github.com/crumbyte/noxmem/pkg/pprofx"
)

type TickMsg struct{}

const (
	defaultRefreshRate   = time.Second
	stackBlockWidthRatio = 0.3

	progressBarPadding = 2
)

type ViewModel struct {
	width  int
	height int

	sessionStartTime time.Time
	refreshRate      time.Duration

	pprofClient      *pprofx.PProfClient
	initMemStats     runtime.MemStats
	memStatsDelta    pprofx.Delta[runtime.MemStats, uint64]
	samplesLocations pprofx.Locations
	maxGCPause       uint64

	allocationsTable *samples.Allocations
	traceTable       *samples.TraceTable

	scatterChart *chart.Scatter
	statusBar    *StatusBar

	help help.Model

	host string
}

type Option func(*ViewModel)

func WithRefreshRate(d time.Duration) Option {
	return func(v *ViewModel) {
		v.refreshRate = d
	}
}

func NewViewModel(host string, client *pprofx.PProfClient, opts ...Option) (*ViewModel, error) {
	var err error

	scatterChart := chart.NewScatter(
		chart.WithPointChar('◍'),
		chart.WithYUnits("ms"),
		chart.WithYLabelFormatter(func(value float64, units string) string {
			return strconv.FormatFloat(value/1_000_000, 'f', 2, 64) + units
		}),
		chart.WithXLabelFormatter(func(value float64, _ string) string {
			sincePauseEnd := time.Since(time.Unix(0, int64(value)))

			return sincePauseEnd.Round(time.Second).String()
		}),
	)

	vm := &ViewModel{
		host:             host,
		refreshRate:      defaultRefreshRate,
		pprofClient:      client,
		sessionStartTime: time.Now(),
		statusBar:        NewStatusBar(),
		traceTable:       samples.NewTraceTable(buildTable()),
		scatterChart:     scatterChart,
		help:             help.New(),
	}

	vm.initMemStats, err = client.MemStats(context.Background())
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(vm)
	}

	vm.allocationsTable = samples.NewSamplesTable(buildTable(), vm.refreshRate)
	vm.allocationsTable.SetStyles(
		samples.AllocationsStyles{
			FileName:           *style.SamplesFileSuffixStyle(),
			FunctionName:       *style.SamplesFileSuffixStyle(),
			BytesStyleResolver: style.SizeUnit,
		},
	)

	vm.traceTable.SetStyles(samples.TraceStyles{
		FileName:     *style.SamplesFileSuffixStyle(),
		FunctionName: *style.SamplesFileSuffixStyle(),
	})

	if err = vm.updateSamples(context.Background()); err != nil {
		return nil, err
	}

	vm.allocationsTable.SetLocations(vm.samplesLocations)
	vm.traceTable.SetLocations(vm.samplesLocations)

	return vm, nil
}

func (vm *ViewModel) Init() tea.Cmd {
	return vm.tick()
}

func (vm *ViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var commands []tea.Cmd

	ctx := context.Background()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		vm.width, vm.height = max(0, msg.Width), max(0, msg.Height)

		vm.scatterChart.SetWidth(vm.width)
		vm.scatterChart.SetHeight(10)
		vm.scatterChart.SetXTickInterval(20)
		vm.scatterChart.SetYTickInterval(3)
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, Bindings.Quit):
			return vm, tea.Quit
		case key.Matches(msg, Bindings.SelectSample):
			vm.allocationsTable.Update(samples.BlurSamples{})
			vm.traceTable.Update(samples.FocusSamples{})
		case key.Matches(msg, Bindings.QuitTrace):
			vm.allocationsTable.Update(samples.FocusSamples{})
			vm.traceTable.Update(samples.BlurSamples{})
		case key.Matches(msg, Bindings.SortKeys):
			vm.allocationsTable.SetSortKey(msg.String())
		case key.Matches(msg, Bindings.Explore):
			_ = explore.Explore(vm.traceTable.SelectedFilename())
		}
	case TickMsg:
		if err := vm.updateSamples(ctx); err != nil {
			panic(err)
		}

		vm.allocationsTable.SetLocations(vm.samplesLocations)
		vm.traceTable.SetLocations(vm.samplesLocations)

		commands = append(commands, vm.tick())
	}

	_, samplesCmd := vm.allocationsTable.Update(msg)
	_, sampleDataCmd := vm.traceTable.Update(msg)

	commands = append(commands, samplesCmd, sampleDataCmd)

	return vm, tea.Batch(commands...)
}

func (vm *ViewModel) View() tea.View {
	if vm.height <= 0 || vm.width <= 0 {
		return tea.View{}
	}

	statusBar := vm.renderStatusBar()
	availableHeight := vm.height - lipgloss.Height(statusBar)

	// calculate available stack block width
	stackBlockWidth := int(float64(vm.width) * stackBlockWidthRatio)
	stackBlock := renderInlineBlock(
		"Stack",
		vm.renderStackStats(stackBlockWidth),
		stackBlockWidth,
	)

	// calculate available GC block width
	gcBlockWidth := vm.width - lipgloss.Width(stackBlock)
	gcBlock := renderInlineBlock(
		"Garbage Collector",
		vm.renderGCStats(gcBlockWidth),
		gcBlockWidth,
	)

	// combine and render stack plus GC blocks
	gcStackBlock := lipgloss.JoinHorizontal(lipgloss.Top, stackBlock, gcBlock)
	availableHeight -= lipgloss.Height(gcStackBlock)

	heapBlock := renderInlineBlock("Heap", vm.renderHeapStats(), vm.width)
	availableHeight -= lipgloss.Height(heapBlock) + 3

	vm.scatterChart.SetHeight(int(float64(availableHeight) * 0.2))
	vm.scatterChart.SetWidth(vm.width - 1)

	chartBlock := renderInlineBlock(
		"GC Pause Graph",
		style.GCPauseGraphStyle().Render(vm.scatterChart.View().Content),
		vm.width,
	)

	availableHeight -= lipgloss.Height(chartBlock)

	allocSamplesBlock := vm.renderAllocSamples(availableHeight / 2)
	availableHeight -= lipgloss.Height(allocSamplesBlock)

	selectedSampleBlock := vm.renderCurrentSample(availableHeight - 1)

	keyBindings := vm.help.ShortHelpView(Bindings.ListBindings())

	return tea.View{
		AltScreen: true,
		Content: lipgloss.NewStyle().
			Height(vm.height).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					heapBlock,
					gcStackBlock,
					chartBlock,
					allocSamplesBlock,
					selectedSampleBlock,
					statusBar,
					keyBindings,
				),
			),
	}
}

func (vm *ViewModel) renderAllocSamples(height int) string {
	vm.allocationsTable.SetHeight(height)

	title := "Allocation Samples [" +
		strconv.Itoa(vm.allocationsTable.Cursor()+1) + "|" +
		strconv.Itoa(len(vm.samplesLocations)) + "]"

	return renderInlineBlock(
		title,
		vm.allocationsTable.View().Content,
		vm.width,
	)
}

func (vm *ViewModel) renderCurrentSample(height int) string {
	vm.traceTable.SetHeight(height)
	vm.traceTable.SetSampleID(vm.allocationsTable.CurrentSampleID())

	title := "Sample Trace [" +
		strconv.Itoa(vm.traceTable.Cursor()+1) + "|" +
		strconv.Itoa(len(vm.samplesLocations[vm.allocationsTable.CurrentSampleID()].Locations)) + "]"

	return renderInlineBlock(
		title,
		vm.traceTable.View().Content,
		vm.width,
	)
}

func renderInline(title, content string) string {
	return style.InlineBlockContentStyle().Render(title + content)
}

func renderInlineBlock(title, content string, width int) string {
	blockStyle := style.InlineBlockStyle().Width(width)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		topBorderLine(title, width),
		blockStyle.Render(content),
	)
}

func topBorderLine(title string, width int) string {
	border := " " + strings.Repeat("─", max(10, width-3-lipgloss.Width(title)))

	borderStyle := style.BlockBorderStyle()

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		borderStyle.Render("─ "),
		(style.BlockBorderTextStyle()).Render(title),
		borderStyle.Render(border),
	)
}

func (vm *ViewModel) tick() tea.Cmd {
	return tea.Tick(
		vm.refreshRate,
		func(_ time.Time) tea.Msg { return TickMsg{} },
	)
}

func (vm *ViewModel) renderHeapStats() string {
	inUseAllocPG := progress.New(progress.WithColors(lipgloss.Color("#ffbe0b")))
	inUseAllocPG.SetWidth(vm.width - progressBarPadding)

	prev := vm.memStatsDelta.Prev()
	stats := vm.memStatsDelta.Current()

	contentMetrics := renderInlineBlocks(
		vm.width,
		renderStat(
			"OS Memory Obtained:",
			vm.fmtMemValue(vm.initMemStats.HeapSys, prev.HeapSys, stats.HeapSys),
		),
		renderStat(
			"Allocated:",
			vm.fmtMemValue(vm.initMemStats.Alloc, prev.Alloc, stats.Alloc),
		),
		renderStat(
			"Memory In-use:",
			vm.fmtMemValue(
				vm.initMemStats.HeapInuse, prev.HeapInuse, stats.HeapInuse,
			),
		),
		renderStat(
			"Idle:",
			vm.fmtMemValue(vm.initMemStats.HeapIdle, prev.HeapIdle, stats.HeapIdle),
		),
		renderStat(
			"Released:",
			vm.fmtMemValue(
				vm.initMemStats.HeapReleased, prev.HeapReleased, stats.HeapReleased,
			),
		),
		renderStat(
			"Objects:",
			strconv.FormatUint(stats.HeapObjects, 10)+
				vm.fmtDelta(
					vm.initMemStats.HeapObjects,
					prev.HeapObjects,
					stats.HeapObjects,
				),
		),
	)

	inUseAvailablePGTitle := "In-Use (" + fmtMemBytes(stats.HeapInuse) +
		") | Available (" +
		fmtMemBytes(stats.HeapSys) + "):\n"

	allocInUsePGTitle := "Allocated (" + fmtMemBytes(stats.Alloc) +
		") | In-Use (" +
		fmtMemBytes(stats.HeapInuse) + "):\n"

	inUseAvailablePG := lipgloss.JoinVertical(
		lipgloss.Left,
		renderInline(
			inUseAvailablePGTitle,
			inUseAllocPG.ViewAs(float64(stats.HeapInuse)/float64(stats.HeapSys)),
		),
	)

	allocInUsePG := lipgloss.JoinVertical(
		lipgloss.Left,
		renderInline(
			allocInUsePGTitle,
			inUseAllocPG.ViewAs(float64(stats.Alloc)/float64(stats.HeapInuse)),
		),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		contentMetrics,
		"\n",
		inUseAvailablePG,
		"\n",
		allocInUsePG,
	)
}

func renderStat(title, content string) string {
	buffer := strings.Builder{}
	buffer.Grow(len(content) + len(title) + 5)

	buffer.WriteString(style.StatTitleTextStyle().Render(title))
	buffer.WriteByte('\n')
	buffer.WriteString(style.StatTextStyle().Render(content))

	return buffer.String()
}

func (vm *ViewModel) renderStackStats(width int) string {
	prev := vm.memStatsDelta.Prev()
	stats := vm.memStatsDelta.Current()

	return renderInlineBlocks(
		width,
		renderStat(
			"Stack Size:",
			vm.fmtMemValue(
				vm.initMemStats.StackSys,
				prev.StackSys,
				stats.StackSys,
			),
		),
		renderStat(
			"In-Use:",
			vm.fmtMemValue(
				vm.initMemStats.StackInuse,
				prev.StackInuse,
				stats.StackInuse,
			),
		),
	)
}

func (vm *ViewModel) renderGCStats(width int) string {
	stats := vm.memStatsDelta.Current()

	vm.maxGCPause = pprofx.MaxGCPause(vm.maxGCPause, stats)

	pauseStr := strconv.FormatFloat(
		float64(vm.maxGCPause)/1_000_000, 'f', 2, 64,
	) + "ms"

	return renderInlineBlocks(
		width,
		renderStat(
			"Next GC Goal:", fmtMemBytes(stats.NextGC),
		),
		renderStat(
			"Last GC Ago:",
			//nolint:gosec
			fmtTimeSince(time.Unix(0, int64(stats.LastGC)), time.Millisecond),
		),
		renderStat(
			"Total GC:",
			strconv.Itoa(int(stats.NumGC)),
		),
		renderStat(
			"Forced GC:",
			strconv.Itoa(int(stats.NumForcedGC)),
		),
		renderStat(
			"GC CPU Usage:",
			strconv.FormatFloat(stats.GCCPUFraction*100, 'f', 5, 64)+"%",
		),
		renderStat(
			"Max GC Pause:", pauseStr,
		),
	)
}

func renderInlineBlocks(availableWidth int, content ...string) string {
	result := make([]string, len(content))
	blocksLeft := len(content)

	contentStyle := style.InlineBlockContentStyle()

	for i := range content {
		result[i] = contentStyle.
			Width(availableWidth / blocksLeft).
			Render(content[i])

		availableWidth -= lipgloss.Width(result[i])

		blocksLeft--
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, result...)
}

func (vm *ViewModel) updateSamples(ctx context.Context) error {
	memStatsDelta, err := vm.pprofClient.MemStatsDelta(ctx, &vm.memStatsDelta)
	if err != nil {
		return fmt.Errorf("render: fetch mem stats delta: %w", err)
	}

	vm.memStatsDelta = *memStatsDelta

	profile, err := vm.pprofClient.HeapProfile(ctx)
	if err != nil {
		return fmt.Errorf("render: fetch heap profile: %w", err)
	}

	vm.samplesLocations = profile.Locations(true)

	vm.updateChart()

	return nil
}

func (vm *ViewModel) updateChart() {
	maxPoints := 20

	gcPauseMap := pprofx.GCPauses(vm.memStatsDelta.Current(), maxPoints)

	points := make([]chart.Point, 0, maxPoints)

	var (
		maxPauseNS  uint64
		maxPauseEnd uint64
		minPauseEnd = uint64(math.MaxUint64)
	)

	for _, gcPause := range gcPauseMap {
		maxPauseNS = max(gcPause.Duration, maxPauseNS)
		maxPauseEnd = max(gcPause.EndTime, maxPauseEnd)
		minPauseEnd = min(gcPause.EndTime, minPauseEnd)

		points = append(
			points,
			chart.Point{
				X: float64(gcPause.EndTime),
				Y: float64(gcPause.Duration),
			},
		)
	}

	vm.scatterChart.SetAxes(
		chart.Axis{float64(minPauseEnd), float64(maxPauseEnd)},
		chart.Axis{0, float64(maxPauseNS)},
	)

	vm.scatterChart.SetPoints(points)
}

func (vm *ViewModel) fmtMemValue(init, prev, current uint64) string {
	return fmtMemBytes(current) + vm.fmtDelta(init, prev, current)
}

func (vm *ViewModel) fmtDelta(init, prev, current uint64) string {
	return " (" + fmtMemDelta(vm.memStatsDelta.DiffPercent(prev, current)) +
		" | " + fmtMemDelta(vm.memStatsDelta.DiffPercent(init, current)) + ")"
}

func fmtMemBytes[T int | int64 | uint64](value T) string {
	return strconv.FormatFloat(float64(value)/1024/1024, 'f', 2, 64) + " MB"
}

func fmtMemDelta(value float64) string {
	var sign string

	deltaStyle := style.ReduceDeltaStyle()

	switch {
	case value < 0:
		sign = "↓"
	case value > 0:
		sign = "↑"
		deltaStyle = style.GrowDeltaStyle()
	}

	return deltaStyle.Render(
		sign + strconv.FormatFloat(math.Abs(value), 'f', 2, 64) + "%",
	)
}

func fmtDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	minutes, seconds := seconds/60, seconds%60
	hours, minutes := minutes/60, minutes%60

	return fmt.Sprintf("%02dh:%02dm:%02ds", hours, minutes, seconds)
}

func fmtTimeSince(t time.Time, roundDuration time.Duration) string {
	return time.Since(t).Round(roundDuration).String()
}

func (vm *ViewModel) renderStatusBar() string {
	if vm.statusBar == nil {
		return ""
	}

	vm.statusBar.Clear()

	barItems := []*BarItem{
		{Content: "0.0.1", BGColor: "#472D30"},
		{Content: "TARGET", BGColor: "#E07A5F"},
		{Content: vm.host, BGColor: "#353533", Width: -1},
		{Content: "RATE", BGColor: "#FF8C61"},
		{Content: vm.refreshRate.String(), BGColor: "#353533"},
		{Content: "SESSION", BGColor: "#FF8C61"},
		{Content: fmtDuration(time.Since(vm.sessionStartTime)), BGColor: "#353533"},
	}

	vm.statusBar.Add(barItems)

	return style.StatusBarStyle().Render(vm.statusBar.Render(vm.width))
}

func buildTable() table.Model {
	tbl := table.New()

	s := table.DefaultStyles()

	s.Header = *style.TableHeader()
	s.Selected = *style.SelectedRow()
	s.Marked = *style.MarkedRow()
	s.Cell = *style.TableCell()

	tbl.SetStyles(s)

	tbl.Help = help.New()

	return tbl
}
