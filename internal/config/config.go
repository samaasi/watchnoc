package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	LogLevel string         `mapstructure:"log_level"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Auth     AuthConfig     `mapstructure:"auth"`
	GitHub   GitHubConfig   `mapstructure:"github"`
	Trello   TrelloConfig   `mapstructure:"trello"`
	Jira     JiraConfig     `mapstructure:"jira"`
}

type ServerConfig struct {
	Port         int `mapstructure:"port"`
	ReadTimeout  int `mapstructure:"read_timeout"`
	WriteTimeout int `mapstructure:"write_timeout"`
	IdleTimeout  int `mapstructure:"idle_timeout"`
}

type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

type RedisConfig struct {
	URL string `mapstructure:"url"`
}

type AuthConfig struct {
	ClerkSecretKey string `mapstructure:"clerk_secret_key"`
	ClerkPublicKey string `mapstructure:"clerk_public_key"`
}

type GitHubConfig struct {
	AppID          int64  `mapstructure:"GITHUB_APP_ID"`
	AppPrivateKey  string `mapstructure:"GITHUB_APP_PRIVATE_KEY"`
	WebhookSecret  string `mapstructure:"GITHUB_WEBHOOK_SECRET"`
	ClientID       string `mapstructure:"GITHUB_CLIENT_ID"`
	ClientSecret   string `mapstructure:"GITHUB_CLIENT_SECRET"`
	BaseURL        string `mapstructure:"GITHUB_API_BASE_URL"`
	WebhookBaseURL string `mapstructure:"GITHUB_WEBHOOK_BASE_URL"`
}

// DecodedPrivateKey decodes the base64/PEM private key from the config.
func (g *GitHubConfig) DecodedPrivateKey() ([]byte, error) {
	if g.AppPrivateKey == "" {
		return nil, fmt.Errorf("github private key is not configured")
	}
	return []byte(g.AppPrivateKey), nil
}

type TrelloConfig struct {
	APIKey             string `mapstructure:"TRELLO_API_KEY"`
	APISecret          string `mapstructure:"TRELLO_API_SECRET"`
	CallbackURL        string `mapstructure:"TRELLO_OAUTH_CALLBACK_URL"`
	WebhookCallbackURL string `mapstructure:"TRELLO_WEBHOOK_CALLBACK_URL"`
	BaseURL            string `mapstructure:"TRELLO_API_BASE_URL"`
}

type JiraConfig struct {
	ClientID         string `mapstructure:"JIRA_CLIENT_ID"`
	ClientSecret     string `mapstructure:"JIRA_CLIENT_SECRET"`
	WebhookSecret    string `mapstructure:"JIRA_WEBHOOK_SECRET"`
	OAuthBaseURL     string `mapstructure:"JIRA_OAUTH_BASE_URL"` // default: "https://auth.atlassian.com"
	APIBaseURL       string `mapstructure:"JIRA_API_BASE_URL"`   // default: "https://api.atlassian.com"
	CallbackURL      string `mapstructure:"JIRA_CALLBACK_URL"`
	TicketKeyPattern string `mapstructure:"JIRA_TICKET_KEY_PATTERN"` // default: "[A-Z][A-Z0-9]+-\d+"
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./internal/config")
	viper.AddConfigPath("./watchnoc/internal/config")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: failed to read config file: %v", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &cfg, nil
}
