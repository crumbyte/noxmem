package pprofx

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/google/pprof/profile"
)

// GoroutineState defines a custom type representing a goroutine's state value.
type GoroutineState string

func (s GoroutineState) String() string {
	return string(s)
}

const (
	Running  GoroutineState = "running"
	Selected GoroutineState = "select"
	Syscall  GoroutineState = "syscall"
	Sleep    GoroutineState = "sleep"
	IOWait   GoroutineState = "IO wait"
)

func NewGoroutineSample() *Sample {
	return &Sample{
		TotalSamples: 1,
		Locations:    make([]*profile.Location, 0),
		GoroutineSample: &GoroutineSample{
			State: Running,
		},
	}
}

// GoroutineParser parses a list of goroutine samples returned by the pprof
// server. It expects a plain text format returned by the debug response from
// the "/debug/pprof/goroutine?debug=2" endpoint.
type GoroutineParser struct{}

func (gp *GoroutineParser) Parse(r io.Reader) (Locations, error) {
	result := make(Locations, 20)

	samples, err := gp.samples(r)
	if err != nil {
		return nil, err
	}

	for _, sample := range samples {
		sampleID := GenerateSampleID(sample.Locations)

		sample.SampleID = sampleID

		if _, ok := result[sampleID]; !ok {
			result[sampleID] = sample

			continue
		}

		result[sampleID].TotalSamples++
	}

	return result, err
}

func (gp *GoroutineParser) samples(r io.Reader) ([]*Sample, error) {
	reader := bufio.NewReader(r)
	samples := make([]*Sample, 0, 20)

	var (
		sample          = NewGoroutineSample()
		readFuncName    = true
		readCreatedBy   = false
		currentLocation = &profile.Location{
			Line: []profile.Line{{Function: &profile.Function{}}},
		}
	)

	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}

			return nil, fmt.Errorf("could not read profile line: %w", readErr)
		}

		switch {
		case strings.HasPrefix(line, "goroutine"):
			sample = NewGoroutineSample()
			if err := gp.parseMetaData(&line, sample.GoroutineSample); err != nil {
				return nil, err
			}

			samples = append(samples, sample)

			readFuncName = true
		case strings.HasPrefix(line, "created by"):
			gp.parseParent(&line, sample.GoroutineSample)

			readCreatedBy = true
		case readCreatedBy:
			err := gp.parseFilename(&line, sample.GoroutineSample.CreatedByLine)
			if err != nil {
				return nil, err
			}

			readCreatedBy = false
		case readFuncName:
			gp.parseFuncName(&line, currentLocation)
			readFuncName = false
		default:
			if err := gp.parseFilename(&line, currentLocation); err != nil {
				return nil, err
			}

			sample.Locations = append(sample.Locations, currentLocation)

			currentLocation = &profile.Location{
				Line: []profile.Line{{Function: &profile.Function{}}},
			}

			readFuncName = true
		}
	}

	return samples, nil
}

// parseFuncName parses a function name that belongs to the goroutine stack trace.
// It takes only the full function name and omits the argument values. Line value
// has the following format: [func_name(args...)]. Example:
// net/http.(*conn).serve(0x374508636090, {0x7ff78b5564d8, 0x3745084a3110}).
func (gp *GoroutineParser) parseFuncName(line *string, l *profile.Location) {
	if line == nil || l == nil {
		return
	}

	if len(l.Line) == 0 || l.Line[0].Function == nil {
		l.Line[0] = profile.Line{Function: &profile.Function{}}
	}

	startBracketIdx := strings.LastIndex(*line, "(")

	if startBracketIdx == -1 {
		startBracketIdx = len(*line) - 1
	}

	l.Line[0].Function.Name = (*line)[:startBracketIdx]
}

// parseMetaData parses the goroutines state and unique ID. The expected line
// value has the following format: [goroutine ID [state, wait_time]:]. Example:
// goroutine 147 [select, 32 minutes]:. The wait time value is optional.
func (gp *GoroutineParser) parseMetaData(line *string, s *GoroutineSample) error {
	startBracketIdx := strings.Index(*line, "[")
	endBracketIdx := strings.Index(*line, "]")

	stateData := strings.Split((*line)[startBracketIdx+1:endBracketIdx], ",")

	intID, err := strconv.Atoi((*line)[10 : startBracketIdx-1])
	if err != nil {
		return fmt.Errorf("could not parse goroutine's ID: %w", err)
	}

	s.ID = intID

	if len(stateData) > 0 {
		s.State = GoroutineState(stateData[0])
	}

	if len(stateData) > 1 {
		s.Wait = strings.TrimSpace(stateData[1])
	}

	return nil
}

// parseParent parses the parent goroutine data. The expected line value has the
// following format: [created by func_name in goroutine goroutine_id]. Example:
// created by net/http.(*connReader).startBackgroundRead in goroutine 103.
func (gp *GoroutineParser) parseParent(line *string, s *GoroutineSample) {
	if line == nil || len(*line) <= 12 {
		return
	}

	trimmedLine := (*line)[11:]

	functionName, _, ok := strings.Cut(trimmedLine, " ")
	if !ok {
		return
	}

	s.CreatedByLine = &profile.Location{
		Line: []profile.Line{{Function: &profile.Function{Name: functionName}}},
	}

	beforeIDIdx := strings.LastIndex(trimmedLine, " ")
	if beforeIDIdx == -1 {
		return
	}

	s.CreatedByID = strings.TrimSpace(trimmedLine[beforeIDIdx+1:])
}

// parseFilename reads the current profile line and tries to parse the function
// filename value and the corresponding line. Parser expects the line of the
// following format: [full_filename_path:line +0hex_instruction], where the
// instruction is optional.
func (gp *GoroutineParser) parseFilename(line *string, l *profile.Location) error {
	if line == nil || l == nil {
		return nil
	}

	if len(l.Line) == 0 || l.Line[0].Function == nil {
		l.Line = append(l.Line, profile.Line{Function: &profile.Function{}})
	}

	// find the beginning of an instruction
	lastSpaceIdx := strings.LastIndex(*line, "+")

	// take the last char idx if the instruction was not found
	if lastSpaceIdx == -1 {
		lastSpaceIdx = len(*line) - 1
	}

	rawFileName := strings.TrimSpace((*line)[:lastSpaceIdx])
	semicolonIdx := strings.LastIndex(rawFileName, ":")

	// handle the path format on Windows platform
	if semicolonIdx == -1 || rawFileName[semicolonIdx+1] == '/' {
		l.Line[0].Function.Filename = rawFileName

		return nil
	}

	l.Line[0].Function.Filename = rawFileName[:semicolonIdx]

	intLine, err := strconv.Atoi(rawFileName[semicolonIdx+1:])
	if err != nil {
		return fmt.Errorf(
			"could not parse function start line %s: %w", *line, err,
		)
	}

	l.Line[0].Line = int64(intLine)

	return nil
}
