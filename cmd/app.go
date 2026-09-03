package cmd

import (
	"errors"
	"os"
	"runtime/debug"
	"time"

	"github.com/crumbyte/noxmem/internal/render"
	"github.com/crumbyte/noxmem/pkg/pprofx"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

var (
	ErrUnknown = errors.New("unknown error")

	target      string
	appVersion  string
	refreshRate int
	showVersion bool

	appCmd = &cobra.Command{
		Use:   "noxmem",
		Short: "Displays the stack and heap memory allocation status of the target application.",
		Long: `
A terminal UI for live Go heap profiling. Connect it to your application's pprof 
endpoint and see the memory runtime state that is otherwise scattered across 
profiling data — heap and stack usage, allocation samples with full traces, GC activity, 
and memory pressure.

🔗 Learn more: https://github.com/crumbyte/noxmem`,
		RunE: run,
	}
)

func init() {
	appCmd.SetVersionTemplate("{{.Version}}\n")

	appCmd.PersistentFlags().
		StringVarP(
			&target,
			"target",
			"t",
			"",
			`Target is a mandatory parameter which specifies the application's pprof endpoint.
The pprof server must be up and running for noxmem to fetch heap/stack stats.

Example: --target="http://localhost:6060"`,
		)

	appCmd.PersistentFlags().
		IntVarP(
			&refreshRate,
			"rate",
			"r",
			2,
			`Rate specifies the refresh rate in seconds at which noxmem will fetch the target
application memory stats. For example, setting this value to 5 will result in 
noxmem fetching the pprof server stats every 5 seconds. Min allowed value is 1 second.

Example: --rate=5`,
		)

	appCmd.PersistentFlags().
		BoolVarP(
			&showVersion,
			"version",
			"v",
			false,
			`Print the application version and exit.`,
		)

	_ = appCmd.MarkPersistentFlagRequired("target")
}

func Execute(version string) {
	appVersion = version
	appCmd.Version = version

	if err := appCmd.Execute(); err != nil {
		if cliErr, ok := errors.AsType[*CLIError](err); ok {
			printError(cliErr.Error())
		}

		os.Exit(1)
	}
}

func run(_ *cobra.Command, _ []string) error {
	client, err := pprofx.NewPProfClient(target, nil)
	if err != nil {
		return err
	}

	style := initStyle()

	render.InitKeyMap(style)

	vm, err := render.NewViewModel(
		target,
		client,
		render.WithVersion(appVersion),
		render.WithRefreshRate(
			time.Second*time.Duration(max(1, refreshRate)),
		),
	)
	if err != nil {
		return err
	}

	teaProg := tea.NewProgram(vm, tea.WithoutCatchPanics())

	defer func() {
		if r := recover(); r != nil {
			var ok bool

			_ = teaProg.ReleaseTerminal()

			if err, ok = r.(error); !ok {
				err = ErrUnknown
			}

			printError(render.ReportError(err, debug.Stack()))
		}
	}()

	if _, err = teaProg.Run(); err != nil {
		return err
	}

	return nil
}

func printError(errMsg string) {
	if _, err := os.Stdout.WriteString(errMsg + "\n"); err != nil {
		return
	}
}

func initStyle() *render.Style {
	return render.InitStyle(render.DefaultColorSchema())
}
