package format_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/crumbyte/noxmem/internal/render/format"

	"github.com/stretchr/testify/require"
)

func TestFmtSize(t *testing.T) {
	tableData := []struct {
		expected string
		bytes    uint64
		width    int
	}{
		{"0.00          B", 0, 15},
		{"1.00          B", 1, 15},
		{"1023.00    B", 1023, 12},
		{"1.00      KB", 1024, 12},
		{"1.00 MB", 1024 << 10, 0},
		{"1.00 GB", 1024 << 20, 0},
		{"1.00 TB", 1024 << 30, 0},
		{"1.00 PB", 1024 << 40, 0},
		{"1.00 EB", 1024 << 50, 0},
		{"512.00 KB", 1024 << 10 / 2, 0},
		{"512.00 MB", 1024 << 20 / 2, 0},
		{"512.00 GB", 1024 << 30 / 2, 0},
		{"512.00 TB", 1024 << 40 / 2, 0},
		{"512.00 PB", 1024 << 50 / 2, 0},
	}

	for _, data := range tableData {
		require.Equal(t, data.expected, format.BytesSizeWidth(data.bytes, data.width))
	}
}

func TestWrapPath(t *testing.T) {
	tableData := []struct {
		path     string
		limit    int
		expected string
	}{
		{filepath.Join("a", "b", "c", "d"), 0, "..."},
		{filepath.Join("a", "b", "c", "d"), 1, "...d"},
		{filepath.Join("a", "b", "c", "d"), 2, filepath.Join("...", "d")},
		{filepath.Join("a", "b", "c", "d"), 3, filepath.Join("...", "d")},
		{filepath.Join("a", "b", "c", "d"), 4, filepath.Join("...", "c", "d")},
		{filepath.Join("a", "b", "c", "d"), 5, filepath.Join("...", "c", "d")},
		{filepath.Join("a", "b", "c", "d"), 6, filepath.Join("...", "b", "c", "d")},
		{filepath.Join("a", "b", "c", "d"), 7, filepath.Join("a", "b", "c", "d")},
		{filepath.Join("a", "b", "c", "d"), 8, filepath.Join("a", "b", "c", "d")},
		{filepath.Join("a", "b", "c", "d"), 10, filepath.Join("a", "b", "c", "d")},
		{filepath.Join("a", "b", "c", "d"), -1, filepath.Join("a", "b", "c", "d")},
		{filepath.Join("a", "b", "c", "d"), -10, filepath.Join("a", "b", "c", "d")},
		{"longPathName", 4, "...Name"},
		{"longPathName/subPath", 4, "...Path"},
	}

	for _, data := range tableData {
		require.Equal(
			t,
			data.expected,
			format.PrefixWrapString(data.path, os.PathSeparator, data.limit),
		)
	}
}

func TestBytesSize(t *testing.T) {
	tests := []struct {
		input          int64
		expectedSize   string
		expectedSuffix string
	}{
		{-1, "0.00", "B"},
		{1, "1.00", "B"},
		{1023, "1023.00", "B"},
		{1024, "1.00", "KB"},
		{1024.0, "1.00", "KB"},
		{1536, "1.50", "KB"},
		{1048576, "1.00", "MB"},
		{1073741824, "1.00", "GB"},
		{1099511627776, "1.00", "TB"},
		{1125899906842624, "1.00", "PB"},
		{1152921504606846976, "1.00", "EB"},
		{1152921504606846977, "1.00", "EB"},
	}

	for _, tt := range tests {
		formattedSize, suffix := format.BytesSize(tt.input)

		require.Equal(t, tt.expectedSize, formattedSize)
		require.Equal(t, tt.expectedSuffix, suffix)
	}
}
