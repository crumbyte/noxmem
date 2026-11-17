package pprofx_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/crumbyte/noxmem/pkg/pprofx"
)

func TestGoroutineParser_Parse(t *testing.T) {
	type data struct {
		name             string
		input            string
		expectGoroutines int
		verify           func(pprofx.Locations)
	}

	table := []data{
		{
			name: "single_goroutine",
			input: `goroutine 1 [running]:
runtime.goexit()
    G:/Program Files/Go/src/runtime/asm_amd64.s:1650 +0x1
`,
			expectGoroutines: 1,
			verify: func(l pprofx.Locations) {
				t.Helper()

				g := firstGoroutine(t, l)

				require.Equal(t, 1, g.GoroutineSample.ID)
				require.Equal(t, pprofx.Running, g.GoroutineSample.State)
				require.Equal(t, 1, g.TotalSamples)
			},
		},
		{
			name: "select_with_wait",
			input: `goroutine 147 [select, 32 minutes]:
runtime.goexit()
    G:/Program Files/Go/src/runtime/asm_amd64.s:1650 +0x1
`,
			expectGoroutines: 1,
			verify: func(l pprofx.Locations) {
				t.Helper()

				g := firstGoroutine(t, l)
				require.Equal(t, 147, g.GoroutineSample.ID)
				require.Equal(t, pprofx.Selected, g.GoroutineSample.State)
				require.Equal(t, "32 minutes", g.GoroutineSample.Wait)
			},
		},
		{
			name: "syscall_with_wait",
			input: `goroutine 42 [syscall, 5 minutes]:
syscall.Syscall()
    G:/Program Files/Go/src/syscall/syscall.go:14 +0x68
`,
			expectGoroutines: 1,
			verify: func(l pprofx.Locations) {
				t.Helper()

				g := firstGoroutine(t, l)
				require.Equal(t, pprofx.Syscall, g.GoroutineSample.State)
				require.Equal(t, "5 minutes", g.GoroutineSample.Wait)
			},
		},
		{
			name: "sleep_state",
			input: `goroutine 7 [sleep]:
time.Sleep()
    G:/Program Files/Go/src/time/sleep.go:90 +0x123
`,
			expectGoroutines: 1,
			verify: func(l pprofx.Locations) {
				t.Helper()

				g := firstGoroutine(t, l)
				require.Equal(t, pprofx.Sleep, g.GoroutineSample.State)
			},
		},
		{
			name: "io_wait_state",
			input: `goroutine 99 [IO wait]:
runtime.IOwait()
    G:/Program Files/Go/src/runtime/sys_linux_amd64.go:245 +0x2b
`,
			expectGoroutines: 1,
			verify: func(l pprofx.Locations) {
				g := firstGoroutine(t, l)
				require.Equal(t, pprofx.IOWait, g.GoroutineSample.State)
			},
		},
		{
			name: "multiple_goroutines",
			input: `goroutine 148 [running]:
runtime/pprof.writeGoroutineStacks({0x7ff78b552018, 0x37450cca82d0})
    G:/Program Files/Go/src/runtime/pprof/pprof.go:816 +0x69
runtime/pprof.writeGoroutine({0x7ff78b552018?, 0x37450cca82d0?}, 0x37450866c6e0?)
    G:/Program Files/Go/src/runtime/pprof/pprof.go:779 +0x25
goroutine 149 [select]:
net/http.(*connReader).startBackgroundRead()
    G:/Program Files/Go/src/net/http/server.go:1234 +0x45
`,
			expectGoroutines: 2,
			verify: func(l pprofx.Locations) {
				g148 := mustFindGoroutine(t, l, 148)
				require.Len(t, g148.Locations, 2)

				g149 := mustFindGoroutine(t, l, 149)
				require.Len(t, g149.Locations, 1)
				require.Equal(t, pprofx.Selected, g149.GoroutineSample.State)
			},
		},
		{
			name: "created_by_with_source",
			input: `goroutine 148 [running]:
runtime.goexit()
    G:/Program Files/Go/src/runtime/asm_amd64.s:1650 +0x1
created by net/http.(*Server).Serve in goroutine 9
    G:/Program Files/Go/src/net/http/server.go:3604 +0x4fd
`,
			expectGoroutines: 1,
			verify: func(l pprofx.Locations) {
				g := firstGoroutine(t, l)
				require.Equal(t, "9", g.GoroutineSample.CreatedByID)
				require.NotNil(t, g.GoroutineSample.CreatedByLine)
				require.Len(t, g.GoroutineSample.CreatedByLine.Line, 1)
				require.Equal(t, "net/http.(*Server).Serve", g.GoroutineSample.CreatedByLine.Line[0].Function.Name)
			},
		},
		{
			name: "full",
			input: `goroutine 148 [running, 38 minutes]:
runtime/pprof.writeGoroutineStacks({0x7ff78b552018, 0x37450cca82d0})
    G:/Program Files/Go/src/runtime/pprof/pprof.go:816 +0x69
runtime/pprof.writeGoroutine({0x7ff78b552018?, 0x37450cca82d0?}, 0x37450866c6e0?)
    G:/Program Files/Go/src/runtime/pprof/pprof.go:779 +0x25
runtime/pprof.(*Profile).WriteTo({0x7ff78b6142d0?, 0x37450cca82d0?}, 0xc?)
    G:/Program Files/Go/src/runtime/pprof/pprof.go:405 +0x149
net/http/pprof.handler.ServeHTTP({0x3745084905b1, 0x9}, {0x7ff78b5552b8, 0x37450cca82d0}, 0xc?)
    G:/Program Files/Go/src/net/http/pprof/pprof.go:272 +0x554
net/http/pprof.Index({0x7ff78b5552b8, 0x37450cca82d0}, 0x3745085903c0?)
    G:/Program Files/Go/src/net/http/pprof/pprof.go:391 +0xdc
net/http.HandlerFunc.ServeHTTP(0x7ff78b633920?, {0x7ff78b5552b8?, 0x37450cca82d0?}, 0x7ff78abb8b3a?)
    G:/Program Files/Go/src/net/http/server.go:2358 +0x29
net/http.(*ServeMux).ServeHTTP(0x7ff78a949d79?, {0x7ff78b5552b8, 0x37450cca82d0}, 0x3745085903c0)
    G:/Program Files/Go/src/net/http/server.go:2923 +0x1cf
net/http.serverHandler.ServeHTTP({0x37450ccba080?}, {0x7ff78b5552b8?, 0x37450cca82d0?}, 0x1?)
    G:/Program Files/Go/src/net/http/server.go:3436 +0x8e
net/http.(*conn).serve(0x374508636090, {0x7ff78b5564d8, 0x3745084a3110})
    G:/Program Files/Go/src/net/http/server.go:2157 +0x6dc
created by net/http.(*Server).Serve in goroutine 9
    G:/Program Files/Go/src/net/http/server.go:3604 +0x4fd
`,
			expectGoroutines: 1,
			verify: func(l pprofx.Locations) {
				g := mustFindGoroutine(t, l, 148)
				require.Equal(t, 148, g.GoroutineSample.ID)
				require.Equal(t, "38 minutes", g.GoroutineSample.Wait)
				require.Equal(t, pprofx.Running, g.GoroutineSample.State)
				require.Equal(t, 1, g.TotalSamples)
				require.Len(t, g.Locations, 9)
				require.Equal(t, "runtime/pprof.writeGoroutineStacks", g.Locations[0].Line[0].Function.Name)
				require.Equal(t, "net/http.(*conn).serve", g.Locations[len(g.Locations)-1].Line[0].Function.Name)
				require.Equal(t, "9", g.GoroutineSample.CreatedByID)
				require.NotNil(t, g.GoroutineSample.CreatedByLine)
				require.Equal(t, "net/http.(*Server).Serve", g.GoroutineSample.CreatedByLine.Line[0].Function.Name)
			},
		},
		{
			name: "deduplication",
			input: `goroutine 1 [running]:
runtime.goexit()
    G:/Program Files/Go/src/runtime/asm_amd64.s:1650 +0x1

goroutine 2 [running]:
runtime.goexit()
    G:/Program Files/Go/src/runtime/asm_amd64.s:1650 +0x1
`,
			expectGoroutines: 1,
			verify: func(l pprofx.Locations) {
				for _, g := range l {
					require.Equal(t, 2, g.TotalSamples)
				}
			},
		},
		{
			name: "filename_and_line",
			input: `goroutine 1 [running]:
main.main()
    G:/Program Files/Go/src/cmd/app.go:42 +0x2a
`,
			expectGoroutines: 1,
			verify: func(l pprofx.Locations) {
				g := firstGoroutine(t, l)
				loc := g.Locations[0].Line[0].Function
				require.Equal(t, "G:/Program Files/Go/src/cmd/app.go", loc.Filename)
			},
		},
		{
			name: "multiple_with_created_by",
			input: `goroutine 148 [running]:
runtime.goexit()
    G:/Program Files/Go/src/runtime/asm_amd64.s:1650 +0x1
created by net/http.(*Server).Serve in goroutine 9
    G:/Program Files/Go/src/net/http/server.go:3604 +0x4fd
goroutine 9 [runnable]:
main.main()
    G:/Program Files/Go/src/cmd/app.go:10 +0x2a
goroutine 10 [sleep]:
time.Sleep()
    G:/Program Files/Go/src/time/sleep.go:90 +0x123
`,
			expectGoroutines: 3,
			verify: func(l pprofx.Locations) {
				g148 := mustFindGoroutine(t, l, 148)
				require.Equal(t, "9", g148.GoroutineSample.CreatedByID)
				require.NotNil(t, g148.GoroutineSample.CreatedByLine)
				require.Equal(t, "net/http.(*Server).Serve", g148.GoroutineSample.CreatedByLine.Line[0].Function.Name)

				g9 := mustFindGoroutine(t, l, 9)
				require.Empty(t, g9.GoroutineSample.CreatedByID)

				g10 := mustFindGoroutine(t, l, 10)
				require.Equal(t, pprofx.Sleep, g10.GoroutineSample.State)
			},
		},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			parser := &pprofx.GoroutineParser{}
			result, err := parser.Parse(strings.NewReader(tc.input))
			require.NoError(t, err)
			require.Len(t, result, tc.expectGoroutines)

			if tc.verify != nil {
				tc.verify(result)
			}
		})
	}
}

func TestGoroutineParser_Parse_EdgeCases(t *testing.T) {
	t.Run("empty_input", func(t *testing.T) {
		parser := &pprofx.GoroutineParser{}
		result, err := parser.Parse(strings.NewReader(""))
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("no_stack_frames", func(t *testing.T) {
		parser := &pprofx.GoroutineParser{}
		result, err := parser.Parse(strings.NewReader("goroutine 1 [runnable]:\n"))
		require.NoError(t, err)
		require.Len(t, result, 1)
		g := firstGoroutine(t, result)
		require.Equal(t, 1, g.GoroutineSample.ID)
		require.Empty(t, g.Locations)
	})

	t.Run("created_by_no_source_line", func(t *testing.T) {
		input := `goroutine 1 [running]:
runtime.goexit()
    G:/Program Files/Go/src/runtime/asm_amd64.s:1650 +0x1
created by main.init in goroutine 0
`
		parser := &pprofx.GoroutineParser{}
		result, err := parser.Parse(strings.NewReader(input))
		require.NoError(t, err)

		g := firstGoroutine(t, result)
		require.Equal(t, "0", g.GoroutineSample.CreatedByID)
		require.NotNil(t, g.GoroutineSample.CreatedByLine)
		require.Len(t, g.GoroutineSample.CreatedByLine.Line, 1)
		require.Equal(t, "main.init", g.GoroutineSample.CreatedByLine.Line[0].Function.Name)
	})
}

func TestGoroutineState_String(t *testing.T) {
	tests := []struct {
		state  pprofx.GoroutineState
		expect string
	}{
		{pprofx.Running, "running"},
		{pprofx.Selected, "select"},
		{pprofx.Syscall, "syscall"},
		{pprofx.Sleep, "sleep"},
		{pprofx.IOWait, "IO wait"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			require.Equal(t, tt.expect, tt.state.String())
		})
	}
}

func firstGoroutine(t *testing.T, l pprofx.Locations) *pprofx.Sample {
	t.Helper()

	for _, g := range l {
		return g
	}

	t.Fatal("no goroutines in result")

	return nil
}

// mustFindGoroutine finds a goroutine by ID in the map, failing the test if not found.
func mustFindGoroutine(t *testing.T, l pprofx.Locations, id int) *pprofx.Sample {
	t.Helper()

	for _, g := range l {
		if g.GoroutineSample.ID == id {
			return g
		}
	}

	t.Fatalf("goroutine %d not found", id)

	return nil
}
