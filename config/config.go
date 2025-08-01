package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/RedHatInsights/chrome-service-backend/rest/util"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
)

// DefaultMaximumNumberRecentlyUsedWorkspaces sets a default number for the maximum amount of "recently used
// workspaces" we want to store in Chrome's database.
const DefaultMaximumNumberRecentlyUsedWorkspaces = 10

type KafkaSSLCfg struct {
	KafkaCA       string
	KafkaUsername string
	KafkaPassword string
	SASLMechanism string
	Protocol      string
}

type KafkaCfg struct {
	KafkaBrokers   []string
	KafkaTopics    []string
	KafkaSSlConfig KafkaSSLCfg
	BrokerConfig   clowder.BrokerConfig
}

type IntercomConfig struct {
	fallback                string
	acs                     string
	acs_dev                 string
	ansible                 string
	ansible_dev             string
	openshift               string
	openshift_dev           string
	ansibleDashboard        string
	ansibleDashboard_dev    string
	automationHub           string
	automationHub_dev       string
	automationAnalytics     string
	automationAnalytics_dev string
	dbaas                   string
	dbaas_dev               string
	activationKeys          string
	activationKeys_dev      string
	advisor                 string
	advisor_dev             string
	compliance              string
	compliance_dev          string
	connector               string
	connector_dev           string
	contentSources          string
	contentSources_dev      string
	dashboard               string
	dashboard_dev           string
	imageBuilder            string
	imageBuilder_dev        string
	inventory               string
	inventory_dev           string
	malware                 string
	malware_dev             string
	patch                   string
	patch_dev               string
	policies                string
	policies_dev            string
	registration            string
	registration_dev        string
	remediations            string
	remediations_dev        string
	ros                     string
	ros_dev                 string
	tasks                   string
	tasks_dev               string
	vulnerability           string
	vulnerability_dev       string
}

type FeatureFlagsConfig struct {
	ClientAccessToken string
	Hostname          string
	Port              int
	Scheme            string
	FullURL           string
	// ONLY for local use, Clowder won't populate this
	AdminToken string
}

type DebugConfig struct {
	DebugFavoriteIds []string
}

type WidgetDashboardConfig struct {
	TemplatesWD string
}

type ChromeServiceConfig struct {
	WebPort                             int
	OpenApiSpecPath                     string
	DbHost                              string
	DbUser                              string
	DbPassword                          string
	DbPort                              int
	DbName                              string
	MetricsPort                         int
	Test                                bool
	LogLevel                            string
	DbSSLMode                           string
	DbSSLRootCert                       string
	KafkaConfig                         KafkaCfg
	IntercomConfig                      IntercomConfig
	FeatureFlagConfig                   FeatureFlagsConfig
	DebugConfig                         DebugConfig
	DashboardConfig                     WidgetDashboardConfig
	MaximumNumberRecentlyUsedWorkspaces int
}

const RdsCaLocation = "/app/rdsca.cert"

func (c *ChromeServiceConfig) getCert(cfg *clowder.AppConfig) string {
	cert := ""
	if cfg.Database.SslMode != "verify-full" {
		return cert
	}
	if cfg.Database.RdsCa != nil {
		err := os.WriteFile(RdsCaLocation, []byte(*cfg.Database.RdsCa), 0644)
		if err != nil {
			panic(err)
		}
		cert = RdsCaLocation
	}
	return cert
}

var config *ChromeServiceConfig

func init() {
	err := util.LoadEnv()
	if err != nil {
		godotenv.Load()
	}
	options := &ChromeServiceConfig{}

	// Log level will default to "Error". Level should be one of
	// info or debug or error
	level, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		level = logrus.ErrorLevel.String()
	}
	options.LogLevel = level

	if clowder.IsClowderEnabled() {
		cfg := clowder.LoadedConfig
		options.DbName = cfg.Database.Name
		options.DbHost = cfg.Database.Hostname
		options.DbPort = cfg.Database.Port
		options.DbUser = cfg.Database.Username
		options.DbPassword = cfg.Database.Password
		options.MetricsPort = cfg.MetricsPort
		options.WebPort = *cfg.PublicPort
		options.DbSSLMode = cfg.Database.SslMode
		options.DbSSLRootCert = options.getCert(cfg)

		if cfg.FeatureFlags != nil {
			options.FeatureFlagConfig.ClientAccessToken = *cfg.FeatureFlags.ClientAccessToken
			options.FeatureFlagConfig.Hostname = cfg.FeatureFlags.Hostname
			options.FeatureFlagConfig.Scheme = string(cfg.FeatureFlags.Scheme)
			options.FeatureFlagConfig.Port = cfg.FeatureFlags.Port
			options.FeatureFlagConfig.FullURL = fmt.Sprintf("%s://%s:%d/api/", options.FeatureFlagConfig.Scheme, options.FeatureFlagConfig.Hostname, options.FeatureFlagConfig.Port)
		}

		if cfg.Kafka != nil {
			broker := cfg.Kafka.Brokers[0]

			options.KafkaConfig.BrokerConfig = broker
			// pass all required topics names
			for _, topic := range cfg.Kafka.Topics {
				options.KafkaConfig.KafkaTopics = append(options.KafkaConfig.KafkaTopics, topic.Name)
			}

			options.KafkaConfig.KafkaBrokers = clowder.KafkaServers

			// Kafka SSL Config
			if broker.Authtype != nil {
				options.KafkaConfig.KafkaSSlConfig.KafkaUsername = *broker.Sasl.Username
				options.KafkaConfig.KafkaSSlConfig.KafkaPassword = *broker.Sasl.Password
				options.KafkaConfig.KafkaSSlConfig.SASLMechanism = *broker.Sasl.SaslMechanism
				options.KafkaConfig.KafkaSSlConfig.Protocol = *broker.Sasl.SecurityProtocol
			}

			if broker.Cacert != nil {
				caPath, err := cfg.KafkaCa(broker)
				if err != nil {
					panic(fmt.Sprintln("Kafka CA failed to write", err))
				}
				options.KafkaConfig.KafkaSSlConfig.KafkaCA = caPath
			}
		}
	} else {
		options.WebPort = 8000
		options.Test = false

		// Ignoring Clowder setup for now
		options.DbUser = os.Getenv("PGSQL_USER")
		options.DbPassword = os.Getenv("PGSQL_PASSWORD")
		options.DbHost = os.Getenv("PGSQL_HOSTNAME")
		port, _ := strconv.Atoi(os.Getenv("PGSQL_PORT"))
		options.DbPort = port
		options.DbName = os.Getenv("PGSQL_DATABASE")
		options.MetricsPort = 9000
		options.DbSSLMode = "disable"
		options.DbSSLRootCert = ""
		options.KafkaConfig = KafkaCfg{
			KafkaTopics:  []string{"platform.chrome"},
			KafkaBrokers: []string{"localhost:9092"},
		}

		options.FeatureFlagConfig.ClientAccessToken = os.Getenv("UNLEASH_API_TOKEN")
		// Only for local use to seed the database, does not work in Clowder.
		options.FeatureFlagConfig.AdminToken = os.Getenv("UNLEASH_ADMIN_TOKEN")
		options.FeatureFlagConfig.Hostname = "0.0.0.0"
		options.FeatureFlagConfig.Scheme = "http"
		options.FeatureFlagConfig.Port = 4242
		options.FeatureFlagConfig.FullURL = fmt.Sprintf("%s://%s:%d/api/", options.FeatureFlagConfig.Scheme, options.FeatureFlagConfig.Hostname, options.FeatureFlagConfig.Port)

		// Attempt parsing the maximum number of recently used workspaces specified via the environment variable.
		number, numConvErr := strconv.Atoi(os.Getenv("RECENTLY_USED_WORKSPACES_MAX_SAVED"))
		if numConvErr != nil {
			options.MaximumNumberRecentlyUsedWorkspaces = DefaultMaximumNumberRecentlyUsedWorkspaces
		} else {
			options.MaximumNumberRecentlyUsedWorkspaces = number
		}
	}

	// env variables from .env or pod env variables
	options.IntercomConfig = IntercomConfig{
		fallback:                os.Getenv("INTERCOM_DEFAULT"),
		acs:                     os.Getenv("INTERCOM_ACS"),
		acs_dev:                 os.Getenv("INTERCOM_ACS_DEV"),
		ansible:                 os.Getenv("INTERCOM_ANSIBLE"),
		ansible_dev:             os.Getenv("INTERCOM_ANSIBLE_DEV"),
		ansibleDashboard:        os.Getenv("INTERCOM_ANSIBLE"),
		ansibleDashboard_dev:    os.Getenv("INTERCOM_ANSIBLE_DEV"),
		automationHub:           os.Getenv("INTERCOM_ANSIBLE"),
		automationHub_dev:       os.Getenv("INTERCOM_ANSIBLE_DEV"),
		automationAnalytics:     os.Getenv("INTERCOM_ANSIBLE"),
		automationAnalytics_dev: os.Getenv("INTERCOM_ANSIBLE_DEV"),
		openshift:               os.Getenv("INTERCOM_OPENSHIFT"),
		openshift_dev:           os.Getenv("INTERCOM_OPENSHIFT_DEV"),
		dbaas:                   os.Getenv("INTERCOM_DBAAS"),
		dbaas_dev:               os.Getenv("INTERCOM_DBAAS_DEV"),
		activationKeys:          os.Getenv("INTERCOM_INSIGHTS"),
		activationKeys_dev:      os.Getenv("INTERCOM_INSIGHTS_DEV"),
		advisor:                 os.Getenv("INTERCOM_INSIGHTS"),
		advisor_dev:             os.Getenv("INTERCOM_INSIGHTS_DEV"),
		compliance:              os.Getenv("INTERCOM_INSIGHTS"),
		compliance_dev:          os.Getenv("INTERCOM_INSIGHTS_DEV"),
		connector:               os.Getenv("INTERCOM_INSIGHTS"),
		connector_dev:           os.Getenv("INTERCOM_INSIGHTS_DEV"),
		contentSources:          os.Getenv("INTERCOM_INSIGHTS"),
		contentSources_dev:      os.Getenv("INTERCOM_INSIGHTS_DEV"),
		dashboard:               os.Getenv("INTERCOM_INSIGHTS"),
		dashboard_dev:           os.Getenv("INTERCOM_INSIGHTS_DEV"),
		imageBuilder:            os.Getenv("INTERCOM_INSIGHTS"),
		imageBuilder_dev:        os.Getenv("INTERCOM_INSIGHTS_DEV"),
		inventory:               os.Getenv("INTERCOM_INSIGHTS"),
		inventory_dev:           os.Getenv("INTERCOM_INSIGHTS_DEV"),
		malware:                 os.Getenv("INTERCOM_INSIGHTS"),
		malware_dev:             os.Getenv("INTERCOM_INSIGHTS_DEV"),
		patch:                   os.Getenv("INTERCOM_INSIGHTS"),
		patch_dev:               os.Getenv("INTERCOM_INSIGHTS_DEV"),
		policies:                os.Getenv("INTERCOM_INSIGHTS"),
		policies_dev:            os.Getenv("INTERCOM_INSIGHTS_DEV"),
		registration:            os.Getenv("INTERCOM_INSIGHTS"),
		registration_dev:        os.Getenv("INTERCOM_INSIGHTS_DEV"),
		remediations:            os.Getenv("INTERCOM_INSIGHTS"),
		remediations_dev:        os.Getenv("INTERCOM_INSIGHTS_DEV"),
		ros:                     os.Getenv("INTERCOM_INSIGHTS"),
		ros_dev:                 os.Getenv("INTERCOM_INSIGHTS_DEV"),
		tasks:                   os.Getenv("INTERCOM_INSIGHTS"),
		tasks_dev:               os.Getenv("INTERCOM_INSIGHTS_DEV"),
		vulnerability:           os.Getenv("INTERCOM_INSIGHTS"),
		vulnerability_dev:       os.Getenv("INTERCOM_INSIGHTS_DEV"),
	}

	options.DebugConfig = DebugConfig{
		DebugFavoriteIds: []string{"", os.Getenv("DEBUG_FAVORITES_ACCOUNT_1")},
	}

	options.DashboardConfig = WidgetDashboardConfig{
		TemplatesWD: os.Getenv("TEMPLATES_WD"),
	}
	if options.DashboardConfig.TemplatesWD == "" {
		options.DashboardConfig.TemplatesWD = "/"
	}

	config = options
}

// Returning chrome-service configuration
func Get() *ChromeServiceConfig {
	return config
}
