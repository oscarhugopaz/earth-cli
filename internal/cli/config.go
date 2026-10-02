package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/oscarhugopaz/earth-cli/internal/config"
)

// providerConfigView is the JSON shape for `earth config`, with secrets
// redacted.
type providerConfigView struct {
	Name            string `json:"name"`
	STACURL         string `json:"stac_url,omitempty"`
	ClientID        string `json:"client_id,omitempty"`
	ClientSecretSet bool   `json:"client_secret_set"`
	TokenURL        string `json:"token_url,omitempty"`
	StatisticsURL   string `json:"statistics_url,omitempty"`
}

type configView struct {
	DefaultProvider string               `json:"default_provider"`
	ConfigFile      string               `json:"config_file,omitempty"`
	ConfigFound     bool                 `json:"config_found"`
	Providers       []providerConfigView `json:"providers"`
}

func newConfigCommand(env Environment) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show the resolved configuration",
		Long: `Show the effective configuration and where it was read from.

Secrets are never printed: client secrets are reported only as a boolean.
Environment variables override the config file.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			session, err := newSession(cmd, env)
			if err != nil {
				return err
			}

			view := configView{
				DefaultProvider: session.cfg.DefaultProvider,
				ConfigFile:      config.Path(env.LookupEnv),
				ConfigFound:     configFileExists(env),
			}
			for _, name := range session.engine.ProviderNames() {
				settings := session.cfg.Provider(name)
				view.Providers = append(view.Providers, providerConfigView{
					Name:            name,
					STACURL:         session.cfg.STACURL(name),
					ClientID:        redactID(settings.ClientID),
					ClientSecretSet: settings.ClientSecret != "",
					TokenURL:        settings.TokenURL,
					StatisticsURL:   settings.StatisticsURL,
				})
			}

			if session.json {
				return session.printer.JSONValue(view)
			}

			session.printer.Field("Default provider", view.DefaultProvider)
			session.printer.Field("Config file", view.ConfigFile)
			session.printer.Field("Config found", yesNo(view.ConfigFound))
			for _, provider := range view.Providers {
				session.printer.Line("")
				session.printer.Field("Provider", provider.Name)
				if provider.STACURL != "" {
					session.printer.Field("STAC URL", provider.STACURL)
				}
				if provider.ClientID != "" {
					session.printer.Field("Client ID", provider.ClientID)
				}
				session.printer.Field("Credentials", yesNo(provider.ClientSecretSet))
				if provider.TokenURL != "" {
					session.printer.Field("Token URL", provider.TokenURL)
				}
				if provider.StatisticsURL != "" {
					session.printer.Field("Statistics URL", provider.StatisticsURL)
				}
			}
			return nil
		},
	}
}

// redactID keeps only the last four characters of an identifier.
func redactID(value string) string {
	if len(value) <= 4 {
		return value
	}
	return "…" + value[len(value)-4:]
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

// configFileExists reports whether the resolved config file exists.
func configFileExists(env Environment) bool {
	path := config.Path(env.LookupEnv)
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
