package pprofx

import (
	"cmp"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/google/pprof/profile"
)

var ErrEmptyLocations = errors.New("nil Locations instance")

type SortType int

const (
	NoSort SortType = iota
	HeapSort
	AllocSort
	ObjectsSort
	GoroutineIDSort
	GoroutineStateSort
)

// SampleID defines a custom type representing a unique profile sample
// identification string.
type SampleID string

func (id SampleID) String() string {
	return string(id)
}

// Locations defines a custom map type where each key, represented by a
// unique sample ID, is mapped to the corresponding list of its locations.
type Locations map[SampleID]*Sample

func (l *Locations) Sort(desc bool, st SortType) ([]*Sample, error) {
	if l == nil {
		return nil, ErrEmptyLocations
	}

	sorted := make([]*Sample, 0, len(*l))

	for _, loc := range *l {
		sorted = append(sorted, loc)
	}

	sign := 1

	if desc {
		sign = -1
	}

	slices.SortFunc(
		sorted,
		func(a, b *Sample) int {
			var compared int

			switch st {
			case HeapSort:
				compared = cmp.Compare(
					a.MemorySample.InUse, b.MemorySample.InUse,
				)
			case AllocSort:
				compared = cmp.Compare(
					a.MemorySample.Alloc, b.MemorySample.Alloc,
				)
			case ObjectsSort:
				compared = cmp.Compare(
					a.MemorySample.InUseObjects, b.MemorySample.InUseObjects,
				)
			case GoroutineIDSort:
				compared = cmp.Compare(
					a.GoroutineSample.ID, b.GoroutineSample.ID,
				)
			case GoroutineStateSort:
				compared = cmp.Compare(
					a.GoroutineSample.State, b.GoroutineSample.State,
				)

				if compared == 0 {
					compared = cmp.Compare(
						a.GoroutineSample.ID, b.GoroutineSample.ID,
					)
				}
			default:
				compared = 0
			}

			return compared * sign
		},
	)

	return sorted, nil
}

type MemoryProfileParser struct {
	WithFreedSamples bool
}

func (m *MemoryProfileParser) Parse(r io.Reader) (Locations, error) {
	source, err := profile.Parse(r)
	if err != nil {
		return nil, fmt.Errorf(
			"could not parse binary heap profile response: %w", err,
		)
	}

	locations := make(Locations)

	for _, sample := range source.Sample {
		// skip already freed objects
		if !m.WithFreedSamples && sample.Value[3] == 0 {
			continue
		}

		sampleID := GenerateSampleID(sample.Location)

		if _, ok := locations[sampleID]; ok {
			locations[sampleID].AddMemorySample(sample)

			continue
		}

		locations[sampleID] = NewMemorySample(sampleID, sample)
	}

	return locations, nil
}

func GenerateSampleID(locations []*profile.Location) SampleID {
	locationsHash := sha256.New()

	for _, loc := range locations {
		if len(loc.Line) == 0 || loc.Line[0].Function == nil {
			continue
		}

		locationsHash.Write([]byte(loc.Line[0].Function.Filename))
	}

	return SampleID(locationsHash.Sum(nil))
}

type MemorySample struct {
	TotalSamples int
	AllocObjects int
	Alloc        int
	InUseObjects int
	InUse        int
}

type GoroutineSample struct {
	ID            int
	CreatedByID   string
	Wait          string
	CreatedByLine *profile.Location
	State         GoroutineState
}

type Sample struct {
	SampleID        SampleID
	TotalSamples    int
	MemorySample    *MemorySample
	GoroutineSample *GoroutineSample
	Locations       []*profile.Location
}

func NewMemorySample(id SampleID, s *profile.Sample) *Sample {
	sl := &Sample{
		SampleID:     id,
		TotalSamples: 1,
		MemorySample: &MemorySample{
			AllocObjects: int(s.Value[0]),
			Alloc:        int(s.Value[1]),
			InUseObjects: int(s.Value[2]),
			InUse:        int(s.Value[3]),
		},
		Locations: make([]*profile.Location, len(s.Location)),
	}

	copy(sl.Locations, s.Location)

	return sl
}

func (s *Sample) AddMemorySample(ps *profile.Sample) {
	s.TotalSamples++

	if s.MemorySample == nil {
		s.MemorySample = &MemorySample{}
	}

	s.MemorySample.AllocObjects += int(ps.Value[0])
	s.MemorySample.Alloc += int(ps.Value[1])

	s.MemorySample.InUseObjects += int(ps.Value[2])
	s.MemorySample.InUse += int(ps.Value[3])
}
