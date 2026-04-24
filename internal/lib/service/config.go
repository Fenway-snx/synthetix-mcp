package service

import (
	"github.com/spf13/viper"

	snx_lib_config "github.com/Fenway-snx/synthetix-mcp/internal/lib/config"
)

// Defines operations required to obtain logging configuration.
type HasServiceConfigCommon interface {
	DeploymentMode() DeploymentMode // Obtains the deployment mode
	LogLevel() string               // Obtains the log level, in string form (e.g. "info").
	LogOutputJSON() bool            // Obtains the flag controlling whether log output should be in JSON.
	LogTags() string                // Obtains the raw log tags string (comma-separated key:value pairs).
}

// Logging configuration type, to be composed into larger structures as
// required.
type ServiceConfigCommon struct {
	deploymentMode DeploymentMode // computed: parsed from "deployment_mode" (or "environment")
	logLevel       string         // config: "log_level"
	logOutputJSON  bool           // config: "log_output_json"
	logTags        string         // config: "log_tags" (comma-separated key:value pairs, e.g. "env:prod,region:us-east-1")
}

var _ HasServiceConfigCommon = (*ServiceConfigCommon)(nil)

func LoadServiceConfigCommon(
	v *viper.Viper,
) (r ServiceConfigCommon, err error) {

	var deploymentMode DeploymentMode
	deploymentMode, _, _ = snx_lib_config.LoadDeploymentMode(v)

	r = ServiceConfigCommon{
		deploymentMode: deploymentMode,
		logLevel:       v.GetString("log_level"),
		logOutputJSON:  v.GetBool("log_output_json"),
		logTags:        v.GetString("log_tags"),
	}

	return
}

// Builds a [ServiceConfigCommon] for tests without Viper.
func NewServiceConfigCommonForTest(
	deploymentMode DeploymentMode,
	logLevel string,
	logOutputJSON bool,
	logTags string,
) ServiceConfigCommon {
	return ServiceConfigCommon{
		deploymentMode: deploymentMode,
		logLevel:       logLevel,
		logOutputJSON:  logOutputJSON,
		logTags:        logTags,
	}
}

func (c ServiceConfigCommon) DeploymentMode() DeploymentMode {
	return c.deploymentMode
}

func (c ServiceConfigCommon) LogLevel() string {
	return c.logLevel
}

func (c ServiceConfigCommon) LogOutputJSON() bool {
	return c.logOutputJSON
}

func (c ServiceConfigCommon) LogTags() string {
	return c.logTags
}
