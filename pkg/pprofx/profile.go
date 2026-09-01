package pprofx

import (
	"cmp"
	"crypto/sha256"
	"errors"
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
)

type Profile struct {
	source *profile.Profile
}

// SampleID defines a custom type representing a unique profile sample
// identification string.
type SampleID string

func (id SampleID) String() string {
	return string(id)
}

// Locations defines a custom map type where each key, represented by a
// unique sample ID, is mapped to the corresponding list of its locations.
type Locations map[SampleID]*SampleLocation

func (l *Locations) Sort(desc bool, st SortType) ([]*SampleLocation, error) {
	if l == nil {
		return nil, ErrEmptyLocations
	}

	sorted := make([]*SampleLocation, 0, len(*l))

	for _, loc := range *l {
		sorted = append(sorted, loc)
	}

	sign := 1

	if desc {
		sign = -1
	}

	slices.SortFunc(
		sorted,
		func(a, b *SampleLocation) int {
			var compared int

			switch st {
			case HeapSort:
				compared = cmp.Compare(a.InUse, b.InUse)
			case AllocSort:
				compared = cmp.Compare(a.Alloc, b.Alloc)
			case ObjectsSort:
				compared = cmp.Compare(a.InUseObjects, b.InUseObjects)
			default:
				compared = 0
			}

			return compared * sign
		},
	)

	return sorted, nil
}

type SampleLocation struct {
	SampleID     SampleID
	TotalSamples int
	AllocObjects int
	Alloc        int
	InUseObjects int
	InUse        int
	Locations    []*profile.Location
}

func NewSampleLocation(id SampleID, s *profile.Sample) *SampleLocation {
	sl := &SampleLocation{
		SampleID:     id,
		TotalSamples: 1,
		AllocObjects: int(s.Value[0]),
		Alloc:        int(s.Value[1]),
		InUseObjects: int(s.Value[2]),
		InUse:        int(s.Value[3]),
		Locations:    make([]*profile.Location, len(s.Location)),
	}

	copy(sl.Locations, s.Location)

	return sl
}

func (sl *SampleLocation) AddSample(s *profile.Sample) {
	sl.TotalSamples++

	sl.AllocObjects += int(s.Value[0])
	sl.Alloc += int(s.Value[1])

	sl.InUseObjects += int(s.Value[2])
	sl.InUse += int(s.Value[3])
}

func (p *Profile) Locations(withFreedSamples bool) Locations {
	if p.source == nil {
		return nil
	}

	locations := make(Locations)

	for _, sample := range p.source.Sample {
		// skip already freed objects
		if !withFreedSamples && sample.Value[3] == 0 {
			continue
		}

		sampleID := p.generateSampleID(sample)

		if _, ok := locations[sampleID]; ok {
			locations[sampleID].AddSample(sample)

			continue
		}

		locations[sampleID] = NewSampleLocation(sampleID, sample)
	}

	return locations
}

func (p *Profile) generateSampleID(s *profile.Sample) SampleID {
	locationsHash := sha256.New()

	for _, loc := range s.Location {
		locationsHash.Write([]byte(loc.Line[0].Function.Filename))
	}

	return SampleID(locationsHash.Sum(nil))
}
