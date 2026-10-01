package constants

const LoftChartRepo = "https://charts.loft.sh"

// Platform connection variables. Here rather than in vcluster-pro because the OSS CLI
// writes them too, from `vcluster platform add standalone`.
const (
	PlatformHostEnv           = "LOFT_PLATFORM_HOST"
	PlatformInstanceNameEnv   = "LOFT_PLATFORM_INSTANCE_NAME"
	PlatformProjectNameEnv    = "LOFT_PLATFORM_PROJECT_NAME"
	PlatformInsecureEnv       = "LOFT_PLATFORM_INSECURE"
	PlatformAccessKeyEnv      = "LOFT_PLATFORM_ACCESS_KEY"
	PlatformSkipConfigSyncEnv = "LOFT_PLATFORM_SKIP_CONFIG_SYNC"
)

const (
	VClusterFolder  = ".vcluster"
	ConfigFileName  = "config.json"
	ManagerFileName = "manager.json"
)
