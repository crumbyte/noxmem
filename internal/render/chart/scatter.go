package chart

import (
	"bytes"
	"cmp"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	defaultPointChar = '█'
	defaultEmptyChar = ' '

	yAxisTickChar = "┤"
	xAxisTickChar = '┬'
	crossAxesChar = '└'
	noTickChar    = '─'

	defaultXTickInterval = 3
	defaultYTickInterval = 2

	yAxisMarksPadding = 3
)

// LabelFormatter defines a custom function type for applying custom formatting
// to the axes tick labels. The first argument contains the current value on an
// axis, and the second represents the value units.
//
// Example:
//
//	chart.WithYLabelFormatter(func(value float64, units string) string {
//		return fmt.Sprintf("%.2f %s", value/1000, units)
//	}
type LabelFormatter func(value float64, units string) string

// DefaultLabelFormatter defines a default formatting function for axes tick
// labels.
var DefaultLabelFormatter = func(value float64, units string) string {
	return strconv.FormatFloat(value, 'f', 1, 64) + units
}

// Point represents a single point on the chart. The Point will be displayed on
// the chart only if it lies within the axes' defined limits.
type Point struct {
	X, Y float64
}

// Axis represents an axis limit pair: the first element contains the starting
// value, and the second contains the axis limit.
type Axis [2]float64

func (a *Axis) IsValid() bool {
	return a[0] <= a[1]
}

// ScatterOption defines a type for providing configuration options for Scatter
// instance.
type ScatterOption func(*Scatter)

// WithXTickInterval sets a label tick interval for the X-axis. The interval
// defines that the label will appear on each Nth tick. For example, setting
// this value to 3 means that the label will appear on each third tick of the
// axis.
func WithXTickInterval(interval int) ScatterOption {
	return func(s *Scatter) {
		s.xTickInterval = max(1, min(interval, s.width))
	}
}

// WithYTickInterval same as WithXTickInterval, it sets the interval for the
// tick label but for the Y-axis.
func WithYTickInterval(interval int) ScatterOption {
	return func(s *Scatter) {
		s.yTickInterval = max(1, min(interval, s.height))
	}
}

func WithXUnits(unit string) ScatterOption {
	return func(s *Scatter) {
		s.xUnits = unit
	}
}

func WithYUnits(unit string) ScatterOption {
	return func(s *Scatter) {
		s.yUnits = unit

		s.width = s.totalWidth - s.yTickLabelMaxWidth()
	}
}

// WithPointChar sets a custom character that will be rendered in place of a
// point on the graph.
func WithPointChar(char rune) ScatterOption {
	return func(s *Scatter) {
		s.pointChar = char
	}
}

// WithEmptyChar sets a custom character that will be rendered in an empty space
// on the graph.
func WithEmptyChar(char rune) ScatterOption {
	return func(s *Scatter) {
		s.emptyChar = char
	}
}

// WithXLabelFormatter sets a custom string formatter function for the X-axis
// label value.
func WithXLabelFormatter(lf LabelFormatter) ScatterOption {
	return func(s *Scatter) {
		s.xLabelFormatter = lf
	}
}

// WithYLabelFormatter sets a custom string formatter function for the Y-axis
// label value.
func WithYLabelFormatter(lf LabelFormatter) ScatterOption {
	return func(s *Scatter) {
		s.yLabelFormatter = lf
	}
}

type Scatter struct {
	xAxis Axis
	yAxis Axis

	xUnits string
	yUnits string
	points []Point

	xLabelFormatter LabelFormatter
	yLabelFormatter LabelFormatter

	xTickInterval int
	yTickInterval int

	totalWidth int
	width      int
	height     int

	pointChar rune
	emptyChar rune
}

func NewScatter(opts ...ScatterOption) *Scatter {
	s := &Scatter{
		xTickInterval:   defaultXTickInterval,
		yTickInterval:   defaultYTickInterval,
		pointChar:       defaultPointChar,
		emptyChar:       defaultEmptyChar,
		xLabelFormatter: DefaultLabelFormatter,
		yLabelFormatter: DefaultLabelFormatter,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Scatter) Init() tea.Cmd {
	return nil
}

func (s *Scatter) Update(_ tea.Msg) (tea.Model, tea.Cmd) {
	return s, nil
}

func (s *Scatter) View() tea.View {
	return tea.View{
		Content: strings.TrimSpace(
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				s.renderYAxis(),
				s.renderPoints(),
			),
		) + "\n" + s.renderXAxis(),
	}
}

func (s *Scatter) SetAxes(x, y Axis) {
	s.xAxis, s.yAxis = x, y

	s.SetWidth(s.totalWidth)
}

// SetWidth sets the graph width for rendering.
func (s *Scatter) SetWidth(width int) {
	s.totalWidth = max(0, width)

	// check if the X tick interval is not bigger than the provided width.
	s.width = max(0, s.totalWidth-s.yTickLabelMaxWidth())
}

// SetHeight sets the graph height for rendering.
func (s *Scatter) SetHeight(height int) {
	s.height = max(0, height)

	// check if the Y tick interval is not bigger than the provided height.
	s.SetYTickInterval(s.yTickInterval)
}

// SetXTickInterval sets the interval for tick labels on the X-axis. The first
// X-axis label always starts at the zero coordinate, and the following are
// placed according to the defined interval value.
func (s *Scatter) SetXTickInterval(interval int) {
	s.xTickInterval = max(1, min(interval, s.width))
}

// SetYTickInterval sets the interval for tick labels on the Y-axis. The first
// Y-axis label always starts at the top coordinate, and the following are
// placed down according to the defined interval value.
func (s *Scatter) SetYTickInterval(interval int) {
	s.yTickInterval = max(1, min(interval, s.height))
}

func (s *Scatter) SetPoints(points []Point) {
	slices.SortFunc(points, func(a, b Point) int {
		return cmp.Compare(b.Y, a.Y)
	})

	s.points = points
}

func (s *Scatter) renderPoints() string {
	if s.width <= 0 {
		return ""
	}

	yStep := s.extrapolatedYStep()
	xStep := s.extrapolatedXStep()

	lines := make([]string, s.height)

	var pointsIdx int

	for step, line := s.yAxis[1], 0; line < s.height; step -= yStep {
		xAxisLine := make([]rune, s.width)

		// fill up the current line with empty characters
		for lineIdx := range xAxisLine {
			if xAxisLine[lineIdx] == 0 {
				xAxisLine[lineIdx] = s.emptyChar
			}
		}

		for ; pointsIdx < len(s.points); pointsIdx++ {
			point := s.points[pointsIdx]

			if !s.isValidCoordinate(point) || point.Y > step {
				continue
			}

			if point.Y < step-yStep {
				pointsIdx = max(0, pointsIdx-1)

				break
			}

			xCoordinate := min(int((point.X-s.xAxis[0])/xStep), s.width-1)

			xAxisLine[xCoordinate] = s.pointChar
		}

		lines[line] = string(xAxisLine)

		line++
	}

	return strings.Join(lines, "\n")
}

func (s *Scatter) renderYAxis() string {
	if s.height == 0 {
		return ""
	}

	yTickLabelMaxWidth := s.yTickLabelMaxWidth()

	yAxis := strings.Builder{}
	yAxis.Grow(s.height * yTickLabelMaxWidth)

	yStep := s.extrapolatedYStep()
	tickChar := yAxisTickChar

	for i, step := 0, s.yAxis[1]; i < s.height; i++ {
		if i%s.yTickInterval == 0 && step >= 0 {
			newTickLabel := s.yLabelFormatter(step, s.yUnits)

			yAxis.WriteString(newTickLabel)
			yAxis.WriteByte(' ')
		}

		yAxis.WriteString(tickChar)
		yAxis.Write([]byte{' ', '\n'})

		step -= yStep
	}

	return lipgloss.NewStyle().Width(yTickLabelMaxWidth).Align(lipgloss.Right).
		Render(yAxis.String())
}

func (s *Scatter) renderXAxis() string {
	if s.width == 0 {
		return ""
	}

	yTickLabelMaxWidth := s.yTickLabelMaxWidth()

	xAxis := strings.Builder{}
	xAxis.Grow(s.width * 2)

	// add left padding for X axis.
	xAxis.Write(bytes.Repeat([]byte{' '}, yTickLabelMaxWidth-2))

	xAxis.WriteRune(crossAxesChar)
	xAxis.WriteRune(noTickChar)

	// fill the X axis ticks according to ticks interval.
	for i := range s.width {
		if i%s.xTickInterval == 0 {
			xAxis.WriteRune('┼')

			continue
		}

		if i%5 == 0 {
			xAxis.WriteRune(xAxisTickChar)

			continue
		}

		xAxis.WriteRune(noTickChar)
	}

	xAxis.WriteByte('\n')

	// add left padding for X axis labels.
	xAxis.Write(bytes.Repeat([]byte{' '}, yTickLabelMaxWidth))

	xAxis.WriteString(s.renderXAxisLabels())

	return xAxis.String()
}

// renderXAxisLabels prepares the X-axis tick labels according to the defined
// tick intervals and label sizing.
func (s *Scatter) renderXAxisLabels() string {
	xStep := s.extrapolatedXStep()

	var (
		prevLabelLen     int
		tickLabelPadding int
	)

	buffer := strings.Builder{}
	buffer.Grow(s.width)

	for tick, step := 0, s.xAxis[0]; tick < s.width; tick, step = tick+1, step+xStep {
		if tick != 0 {
			tickLabelPadding = s.xTickInterval
		}

		if tick%s.xTickInterval != 0 {
			continue
		}

		buffer.Write(
			bytes.Repeat([]byte{' '}, max(0, tickLabelPadding-prevLabelLen)),
		)

		newTickLabel := s.xLabelFormatter(step, s.xUnits)
		prevLabelLen = len(newTickLabel)

		if buffer.Len()+prevLabelLen >= s.width {
			break
		}

		buffer.WriteString(newTickLabel)
	}

	return buffer.String()
}

// xTickLabelMaxWidth returns the length of the longest tick label on the X-axis.
// nolint:unused
func (s *Scatter) xTickLabelMaxWidth() int {
	return lipgloss.Width(strconv.FormatFloat(s.xAxis[1], 'f', 1, 64)) +
		len(s.xUnits)
}

// yTickLabelMaxWidth returns the length of the longest tick label on the Y-axis.
func (s *Scatter) yTickLabelMaxWidth() int {
	return lipgloss.Width(s.yLabelFormatter(s.yAxis[1], s.yUnits)) +
		yAxisMarksPadding
}

// extrapolatedXStep returns a unit of the X-axis extrapolated on the current
// graph width value. If the current width is zero, the X-axis limit will be
// returned.
func (s *Scatter) extrapolatedXStep() float64 {
	if s.width <= 1 {
		return max(1, s.xAxis[1]-s.xAxis[0])
	}

	return (s.xAxis[1] - s.xAxis[0]) / float64(s.width-1)
}

// extrapolatedYStep returns a unit of the Y-axis extrapolated on the current
// graph height value. If the current height is zero, the Y-axis limit will be
// returned.
func (s *Scatter) extrapolatedYStep() float64 {
	if s.height <= 1 {
		return s.yAxis[1] - s.yAxis[0]
	}

	return (s.yAxis[1] - s.yAxis[0]) / float64(s.height-1)
}

// isValidCoordinate checks whether the provided point is visible on the graph
// according to the axes limits.
func (s *Scatter) isValidCoordinate(p Point) bool {
	return p.X >= s.xAxis[0] && p.X <= s.xAxis[1] &&
		p.Y >= s.yAxis[0] && p.Y <= s.yAxis[1]
}
