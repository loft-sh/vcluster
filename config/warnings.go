package config

import (
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"golang.org/x/mod/semver"
)

const v32 = "v0.32.0-alpha.0"

func WarningPlatform() string {
	if semver.Compare("v"+upgrade.GetVersion(), v32) == -1 {
		return ""
	}

	return `Platform configuration is no longer under "external". Please update your values and specify them with --values.
For example:

| external:                       | platform:
|   platform:                     |   project: default
|     project: default     ---->  |   apiKey:
|     apiKey:                     |     secretName: foo
|       secretName: foo           |     namespace: bar
|       namespace: bar            |     createRBAC: true
|       createRBAC: true          |

`
}

func WarningAutoDelete() string {
	if semver.Compare("v"+upgrade.GetVersion(), v32) == -1 {
		return ""
	}

	return `AutoDelete configuration is no longer under "external.platform". Please update your values and specify them with --values.
For example:

| external:                       | deletion:
|   platform:                     |   prevent: false
|     autoDelete:          ---->  |   auto:
|       afterInactivity: 3600     |     afterInactivity: 1h

`
}

func WarningAutoSleep() string {
	if semver.Compare("v"+upgrade.GetVersion(), v32) == -1 {
		return ""
	}

	return `AutoSleep configuration is no longer under "external.platform". Please update your values and specify them with --values.
For example:

| external:                       | sleep:
|   platform:                     |   auto:
|     autoSleep:           ---->  |     afterInactivity: 1h
|       afterInactivity: 3600     |     schedule: "0 3 * * *"
|       schedule: "0 3 * * *"     |     timezone: UTC
|       timezone: UTC             |     wakeup:
|       autoWakeup:               |       schedule: "0 4 * * *"
|         schedule: "0 4 * * *"   |

`
}

func WarningAutoSnapshot() string {
	if semver.Compare("v"+upgrade.GetVersion(), v32) == -1 {
		return ""
	}

	return `AutoSnapshot configuration is no longer under "external.platform" and "enabled" has been removed. Please update your values and specify them with --values.
For example:

| external:                       | snapshots:
|   platform:                     |   auto:
|     autoSnapshot:        ---->  |     schedule: "0 */12 * * *"
|       enabled: true             |     timezone: America/New_York
|       schedule: "0 */12 * * *"  |     retention:
|       timezone: America/New_Y.  |       period: 30
|       retention:                |       maxSnapshots: 14
|         period: 30              |     storage:
|         maxSnapshots: 14        |       type: s3
|       storage:                  |       s3:
|         type: s3                |         url: s3://my-bucket/path
|         s3:                     |     volumes:
|           url: s3://my-bucket/  |       enabled: true
|       volumes:                  |
|         enabled: true           |

`
}
