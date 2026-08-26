package constants

import "time"

const (
	PollingInterval         = time.Second * 2
	PollingTimeoutVeryShort = time.Second * 5
	PollingTimeoutShort     = time.Second * 20
	PollingTimeout          = time.Second * 60
	PollingTimeoutLong      = time.Second * 120
	PollingTimeoutVeryLong  = time.Second * 300
)
