// Package cli implements the earth command line interface.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oscarhugopaz/earth-cli/internal/apperr"
	"github.com/oscarhugopaz/earth-cli/internal/config"
	"github.com/oscarhugopaz/earth-cli/internal/engine"
	"github.com/oscarhugopaz/earth-cli/internal/output"
	"github.com/oscarhugopaz/earth-cli/internal/provider"
)

// Environment carries process dependencies so the CLI can be tested without
// touching the real stdout, stdin, or environment.
type Environment struct {
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	LookupEnv func(string) (string, bool)
}

func (e Environment) withDefaults() Environment {
	if e.Stdin == nil {
		e.Stdin = os.Stdin
	}
	if e.Stdout == nil {
		e.Stdout = os.Stdout
	}
	if e.Stderr == nil {
		e.Stderr = os.Stderr
	}
	if e.LookupEnv == nil {
		e.LookupEnv = os.LookupEnv
	}
	return e
}

// Run executes the CLI and returns the process exit code.
func Run(ctx context.Context, args []string, env Environment) int {
	env = env.withDefaults()
	root := newRootCommand(env)
	root.SetArgs(args)

	if err := root.ExecuteContext(ctx); err != nil {
		printError(env, err)
		return exitCode(err)
	}
	return apperr.ExitOK
}

func printError(env Environment, err error) {
	fmt.Fprintf(env.Stderr, "earth: %s\n", err.Error())
	if exitCode(err) == apperr.ExitUsage {
		fmt.Fprintln(env.Stderr, "Run 'earth --help' for usage.")
	}
}

// exitCode classifies errors that Cobra produces before our own typed errors
// are involved (unknown commands, bad flags, missing arguments).
func exitCode(err error) int {
	var appError *apperr.Error
	if errors.As(err, &appError) {
		return appError.Code
	}
	message := err.Error()
	for _, fragment := range []string{
		"unknown command",
		"unknown flag",
		"unknown shorthand flag",
		"required flag",
		"invalid argument",
		"accepts ",
		"flag needs an argument",
	} {
		if strings.Contains(message, fragment) {
			return apperr.ExitUsage
		}
	}
	return apperr.ExitError
}

// session bundles configuration, engine, and output for a single command.
type session struct {
	cfg      config.Config
	engine   *engine.Engine
	printer  *output.Printer
	provider string
	json     bool
}

func newSession(cmd *cobra.Command, env Environment) (*session, error) {
	jsonOut, _ := cmd.Flags().GetBool("json")
	providerName, _ := cmd.Flags().GetString("provider")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	noColor, _ := cmd.Flags().GetBool("no-color")
	verbose, _ := cmd.Flags().GetBool("verbose")
	debug, _ := cmd.Flags().GetBool("debug")
	verbose = verbose || debug

	cfg, err := config.Load(env.LookupEnv)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(providerName) == "" {
		providerName = cfg.DefaultProvider
	}

	options := []engine.Option{engine.WithHTTPTimeout(timeout)}
	if verbose {
		options = append(options, engine.WithDebug(func(format string, args ...any) {
			fmt.Fprintf(env.Stderr, "debug: "+format+"\n", args...)
		}))
	}

	return &session{
		cfg:      cfg,
		engine:   engine.New(cfg, options...),
		printer:  output.New(env.Stdout, env.Stderr, jsonOut, output.SupportsColor(env.Stdout, noColor, env.LookupEnv)),
		provider: providerName,
		json:     jsonOut,
	}, nil
}

func (s *session) selectedProvider() (provider.Provider, error) {
	p, err := s.engine.Provider(s.provider)
	if err != nil {
		return nil, apperr.Usage("%s", err)
	}
	return p, nil
}
