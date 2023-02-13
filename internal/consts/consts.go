package consts

const (
	// Schema key for the API token configuration.
	APITokenSchemaKey = "api_token"

	// Environment variable key for the API token configuration.
	APITokenEnvVarKey = "AZION_TERRAFORM_TOKEN"

	// Schema key for the API key configuration.
	APIKeySchemaKey = "api_key"

	// Environment variable key for the API key configuration.
	APIKeyEnvVarKey = "azion_API_KEY"

	// Schema key for the email configuration.
	EmailSchemaKey = "email"

	// Environment variable key for the email configuration.
	EmailEnvVarKey = "azion_EMAIL"

	// Schema key for the API user service key configuration.
	APIUserServiceKeySchemaKey = "api_user_service_key"

	// Environment variable key for the API user service key configuration.
	APIUserServiceKeyEnvVarKey = "azion_API_USER_SERVICE_KEY"

	// Schema key for the API hostname configuration.
	APIHostnameSchemaKey = "api_hostname"

	// Environment variable key for the API hostname configuration.
	APIHostnameEnvVarKey = "azion_API_HOSTNAME"

	// Default value for the API hostname.
	APIHostnameDefault = "api.azion.com"

	// Schema key for the API base path configuration.
	APIBasePathSchemaKey = "api_base_path"

	// Environment variable key for the API base path configuration.
	APIBasePathEnvVarKey = "azion_API_BASE_PATH"

	// Default value for the API base path.
	APIBasePathDefault = "/client/v4"

	// Schema key for the requests per second configuration.
	RPSSchemaKey = "rps"

	// Environment variable key for the requests per second configuration.
	RPSEnvVarKey = "azion_RPS"

	// Default value for the requests per second.
	RPSDefault = "4"

	// Schema key for the retries configuration.
	RetriesSchemaKey = "retries"

	// Environment variable key for the retries configuration.
	RetriesEnvVarKey = "azion_RETRIES"

	// Default value for the retries.
	RetriesDefault = "4"

	// Schema key for the minimum backoff configuration.
	MinimumBackoffSchemaKey = "min_backoff"

	// Environment variable key for the minimum backoff configuration.
	MinimumBackoffEnvVar = "azion_MIN_BACKOFF"

	// Default value for the minimum backoff.
	MinimumBackoffDefault = "1"

	// Schema key for the maximum configuration.
	MaximumBackoffSchemaKey = "max_backoff"

	// Environment variable key for the maximum backoff configuration.
	MaximumBackoffEnvVarKey = "azion_MAX_BACKOFF"

	// Default value for the maximum backoff.
	MaximumBackoffDefault = "30"

	APIClientLoggingSchemaKey = "api_client_logging"
	APIClientLoggingEnvVarKey = "azion_API_CLIENT_LOGGING"

	// Schema key for the account ID configuration.
	AccountIDSchemaKey = "account_id"

	// Environment variable key for the account ID configuration.
	//
	// Deprecated: Use resource specific account ID values instead.
	AccountIDEnvVarKey = "azion_ACCOUNT_ID"

	// Schema key for the zone ID configuration.
	ZoneIDSchemaKey = "zone_id"

	UserAgentDefault = "terraform/%s terraform-plugin-sdk/%s terraform-provider-azion/%s"
)
