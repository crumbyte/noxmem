package samples

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// FunctionNameSuffix formats the function's package path by highlighting the
// function name. It applies the provided style to the function's name suffix.
func FunctionNameSuffix(path string, style lipgloss.Style) string {
	idx := strings.LastIndex(strings.ToLower(path), ".")
	if idx == -1 {
		return path
	}

	return path[:idx+1] + style.Render(path[idx+1:])
}

// FilepathSuffix formats the full file path by highlighting the file's name.
// It applies the provided style to the file's name only.
func FilepathSuffix(path string, style lipgloss.Style) string {
	idx := strings.LastIndex(strings.ToLower(path), "/")
	if idx == -1 {
		return path
	}

	return path[:idx+1] + style.Render(path[idx+1:])
}
