package protocol

import (
	"fmt"
	"regexp"
	"strconv"
)

// ProtocolVersion represents a Guacamole protocol version (e.g., 1.5.0).
type ProtocolVersion struct {
	Major, Minor, Patch int
}

var (
	Version1_0_0  = ProtocolVersion{1, 0, 0}
	Version1_1_0  = ProtocolVersion{1, 1, 0}
	Version1_3_0  = ProtocolVersion{1, 3, 0}
	Version1_5_0  = ProtocolVersion{1, 5, 0}
	VersionLatest = Version1_5_0

	VERSION_1_0_0 = Version1_0_0
	VERSION_1_1_0 = Version1_1_0
	VERSION_1_3_0 = Version1_3_0
	VERSION_1_5_0 = Version1_5_0
	LATEST        = VersionLatest
)

var versionPattern = regexp.MustCompile(`^VERSION_([0-9]+)_([0-9]+)_([0-9]+)$`)

// NewProtocolVersion returns a Guacamole protocol version with the given
// major, minor, and patch components.
func NewProtocolVersion(major, minor, patch int) ProtocolVersion {
	return ProtocolVersion{major, minor, patch}
}

// ParseVersion parses a version string of the form "VERSION_1_5_0".
func ParseVersion(s string) (ProtocolVersion, bool) {
	matches := versionPattern.FindStringSubmatch(s)
	if matches == nil {
		return ProtocolVersion{}, false
	}
	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])
	return ProtocolVersion{major, minor, patch}, true
}

// GetMajor returns the major version component.
func (v ProtocolVersion) GetMajor() int {
	return v.Major
}

// GetMinor returns the minor version component.
func (v ProtocolVersion) GetMinor() int {
	return v.Minor
}

// GetPatch returns the patch version component.
func (v ProtocolVersion) GetPatch() int {
	return v.Patch
}

// AtLeast reports whether v is greater than or equal to other.
func (v ProtocolVersion) AtLeast(other ProtocolVersion) bool {
	if v.Major != other.Major {
		return v.Major > other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor > other.Minor
	}
	return v.Patch >= other.Patch
}

// String returns the canonical string representation, e.g. "VERSION_1_5_0".
func (v ProtocolVersion) String() string {
	return fmt.Sprintf("VERSION_%d_%d_%d", v.Major, v.Minor, v.Patch)
}
