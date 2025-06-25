package useragent

import "github.com/loft-sh/image/version"

// DefaultUserAgent is a value that should be used by User-Agent headers, unless the user specifically instructs us otherwise.
var DefaultUserAgent = "containers/" + version.Version + " (github.com/containers/image)"
