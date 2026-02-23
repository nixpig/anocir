// Package platform provides functionality for performing the low-level
// operations needed for container isolation and resource management.
// Since anocir is Linux-specific, `unix` stdlib functions are used preferentially
// over their `os` equivalent for consistency.
package platform
