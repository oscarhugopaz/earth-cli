package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/oscarhugopaz/earth-cli/internal/apperr"
	"github.com/oscarhugopaz/earth-cli/internal/output"
	"github.com/oscarhugopaz/earth-cli/internal/skill"
)

func newSkillCommand(env Environment) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Install the bundled Earth skill for coding agents",
	}
	var agent string
	var global, all, force bool
	install := &cobra.Command{
		Use:   "install",
		Short: "Install the Earth skill in the current directory or user home",
		Long: `Install the portable Earth skill included in this binary. No network or
agent executable is required. The default is .agents/skills/earth/SKILL.md
under the current working directory; --global selects the user home.

  earth skill install
  earth skill install --agent claude
  earth skill install --global --agent pi
  earth skill install --all
  earth skill install --global --all

--all uses shared .agents plus .claude; globally it also installs in Hermes.
Explicit --agent selects the agent's native location (Codex uses .agents).
Identical files are unchanged. Different content requires --force. Symlinks
are rejected and other files are preserved. Agent trust is never modified.
For local Hermes discovery, install from the Git repository root and follow
Hermes' project-trust prompt.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := os.Getwd()
			if err != nil {
				return err
			}
			home, _ := env.LookupEnv("HOME")
			if home == "" {
				home, _ = env.LookupEnv("USERPROFILE")
			}
			dirs, err := skill.Targets(project, home, agent, global, all, env.LookupEnv)
			if err != nil {
				return apperr.Usage("%s", err)
			}
			results, err := skill.Install(dirs, force)
			if err != nil {
				for _, result := range results {
					if result.Status != "unchanged" {
						// Multi-destination writes are not a cross-filesystem transaction.
						cmd.PrintErrf("%s: %s\n", result.Status, result.Path)
					}
				}
				return err
			}
			jsonOut, _ := cmd.Flags().GetBool("json")
			printer := output.New(env.Stdout, env.Stderr, jsonOut, false)
			if jsonOut {
				return printer.JSONValue(results)
			}
			for _, result := range results {
				printer.Line("%s: %s", result.Status, result.Path)
			}
			return nil
		},
	}
	install.Flags().StringVar(&agent, "agent", "", "destination: agents, claude, codex, opencode, pi, or hermes")
	install.Flags().BoolVar(&global, "global", false, "install for the current user instead of this directory")
	install.Flags().BoolVar(&all, "all", false, "install for every supported agent using shared locations")
	install.Flags().BoolVar(&force, "force", false, "replace SKILL.md if its content differs")
	cmd.AddCommand(install)
	return cmd
}
