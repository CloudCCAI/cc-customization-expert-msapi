// Package edition contains build-time distribution metadata. Release builds
// set these variables with -ldflags so all editions use the same Go sources.
package edition

var PackageName = "cc-customization-expert-msapi"
var VersionSuffix = "-msapi"
var DefaultExecutionMode = "msapi"
var StrictExecutionMode = "msapi"
