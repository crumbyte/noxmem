package render

import (
	"encoding/json"
	"fmt"
	"os"
)

type StatusBarColors struct {
	Text      string `json:"text"`
	BlockText string `json:"blockText"`
	BG        string `json:"background"`
	VersionBG string `json:"versionBackground"`
}

type SizeUnitColors struct {
	B  string `json:"b"`
	KB string `json:"kb"`
	MB string `json:"mb"`
	GB string `json:"gb"`
	TB string `json:"tb"`
	PB string `json:"pb"`
	EB string `json:"eb"`
}

type Samples struct {
	Line         string `json:"line"`
	FileName     string `json:"fileName"`
	FunctionName string `json:"functionName"`
}

// The ColorSchema schema contains color values for most UI elements, such as
// text and border colors, element backgrounds, etc. The Style component uses
// the ColorSchema instance during rendering elements. Each color value must be
// represented as a hex ("#FFBF69") or ANSI ("240") string color code.
//
// DefaultColorSchema always used as a base schema and all customizations are
// applied over it.
type ColorSchema struct {
	StatusBar         StatusBarColors `json:"statusBar"`
	SizeUnit          SizeUnitColors  `json:"sizeUnit"`
	Samples           Samples         `json:"samples"`
	CellText          string          `json:"cellText"`
	TableHeaderBorder string          `json:"tableHeaderBorder"`
	TableHeaderText   string          `json:"tableHeaderText"`
	SelectedRowText   string          `json:"selectedRowText"`
	SelectedRowBG     string          `json:"selectedRowBackground"`
	MarkedRowText     string          `json:"markedRowText"`
	MarkedRowBG       string          `json:"markedRowBackground"`
	GrowDeltaText     string          `json:"growDeltaText"`
	ReduceDeltaText   string          `json:"reduceDeltaText"`
	StatTitleText     string          `json:"statTitleText"`
	StatText          string          `json:"statText"`
	BlockBorder       string          `json:"blockBorder"`
	BlockBorderText   string          `json:"blockText"`
	GCPauseGraph      string          `json:"gcPauseGraph"`
	HelpText          string          `json:"helpText"`
	BindingText       string          `json:"bindingText"`
	StatusBarBorder   bool            `json:"statusBarBorder"`
}

// DecodeColorSchema reads the color schema from the file by the provided
// path and applies it to the *ColorSchema instance. An error will be returned
// if the path is invalid or the JSON color schema content cannot be decoded.
func DecodeColorSchema(path string, cs *ColorSchema) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open color schema file %s: %w", path, err)
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(file)

	if err = json.NewDecoder(file).Decode(cs); err != nil {
		return fmt.Errorf("decode color schema file %s: %w", path, err)
	}

	return nil
}

func DefaultColorSchema() ColorSchema {
	return ColorSchema{
		StatusBar: StatusBarColors{
			VersionBG: "#8338EC",
			Text:      "#C1C6B2",
			BG:        "#353533",
			BlockText: "#FFFDF5",
		},
		SizeUnit: SizeUnitColors{
			B:  "#C4FFCE",
			KB: "#A1E162",
			MB: "#FFEB6A",
			GB: "#f48c06",
			TB: "#dc2f02",
			PB: "#9d0208",
			EB: "#6a040f",
		},
		Samples: Samples{
			Line:         "#FFBE0B",
			FileName:     "#FFBE0B",
			FunctionName: "#FFBE0B",
		},
		CellText:          "#F4F1DE",
		TableHeaderBorder: "240",
		TableHeaderText:   "#FAA275",
		SelectedRowText:   "#FFBE0B",
		SelectedRowBG:     "#472D30",
		MarkedRowText:     "#262626",
		MarkedRowBG:       "#FF8C61",
		GrowDeltaText:     "#E26D5C",
		ReduceDeltaText:   "#81B29A",
		StatTitleText:     "#FAA275",
		StatText:          "#F4F1DE",
		BlockBorder:       "#FF8C61",
		BlockBorderText:   "#F2CC8F",
		GCPauseGraph:      "#FAEDCD",
		HelpText:          "#696868",
		BindingText:       "#FFBF69",
		StatusBarBorder:   true,
	}
}

func SimpleColorSchema() ColorSchema {
	dcs := DefaultColorSchema()

	dcs.StatusBarBorder = false

	return dcs
}
