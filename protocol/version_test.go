package protocol_test

import (
	"testing"

	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestParseVersion(t *testing.T) {
	cases := []struct {
		in   string
		want protocol.ProtocolVersion
		ok   bool
	}{
		{"VERSION_1_0_0", protocol.Version1_0_0, true},
		{"VERSION_1_1_0", protocol.Version1_1_0, true},
		{"VERSION_1_3_0", protocol.Version1_3_0, true},
		{"VERSION_1_5_0", protocol.Version1_5_0, true},
		{"VERSION_012_102_398", protocol.ProtocolVersion{Major: 12, Minor: 102, Patch: 398}, true},
		{"bad", protocol.ProtocolVersion{}, false},
		{"VERSION_2_0", protocol.ProtocolVersion{}, false},
		{"VERSION_", protocol.ProtocolVersion{}, false},
		{"VERSION___", protocol.ProtocolVersion{}, false},
		{"VERSION__2_3", protocol.ProtocolVersion{}, false},
		{"VERSION_1__3", protocol.ProtocolVersion{}, false},
		{"VERSION_1_2_", protocol.ProtocolVersion{}, false},
		{"VERSION_A_2_3", protocol.ProtocolVersion{}, false},
		{"VERSION_1_B_3", protocol.ProtocolVersion{}, false},
		{"VERSION_1_2_C", protocol.ProtocolVersion{}, false},
		{"_1_2_3", protocol.ProtocolVersion{}, false},
		{"version_1_2_3", protocol.ProtocolVersion{}, false},
		{"VERSION_1_2_3_extra", protocol.ProtocolVersion{}, false},
	}
	for _, c := range cases {
		got, ok := protocol.ParseVersion(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ParseVersion(%q) = %v, %v; want %v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestNewProtocolVersionAndJavaStyleGetters(t *testing.T) {
	version := protocol.NewProtocolVersion(2, 3, 4)

	if got := version.GetMajor(); got != 2 {
		t.Errorf("GetMajor() = %d, want 2", got)
	}
	if got := version.GetMinor(); got != 3 {
		t.Errorf("GetMinor() = %d, want 3", got)
	}
	if got := version.GetPatch(); got != 4 {
		t.Errorf("GetPatch() = %d, want 4", got)
	}
	if got := version.String(); got != "VERSION_2_3_4" {
		t.Errorf("String() = %q, want VERSION_2_3_4", got)
	}
}

func TestProtocolVersion_JavaStyleConstantAliases(t *testing.T) {
	if protocol.VERSION_1_0_0 != protocol.Version1_0_0 {
		t.Errorf("VERSION_1_0_0 = %v, want %v", protocol.VERSION_1_0_0, protocol.Version1_0_0)
	}
	if protocol.VERSION_1_1_0 != protocol.Version1_1_0 {
		t.Errorf("VERSION_1_1_0 = %v, want %v", protocol.VERSION_1_1_0, protocol.Version1_1_0)
	}
	if protocol.VERSION_1_3_0 != protocol.Version1_3_0 {
		t.Errorf("VERSION_1_3_0 = %v, want %v", protocol.VERSION_1_3_0, protocol.Version1_3_0)
	}
	if protocol.VERSION_1_5_0 != protocol.Version1_5_0 {
		t.Errorf("VERSION_1_5_0 = %v, want %v", protocol.VERSION_1_5_0, protocol.Version1_5_0)
	}
	if protocol.LATEST != protocol.VersionLatest {
		t.Errorf("LATEST = %v, want %v", protocol.LATEST, protocol.VersionLatest)
	}
}

func TestProtocolVersion_String(t *testing.T) {
	if got := protocol.Version1_5_0.String(); got != "VERSION_1_5_0" {
		t.Errorf("String() = %q, want VERSION_1_5_0", got)
	}
}

func TestProtocolVersion_AtLeast(t *testing.T) {
	if !protocol.Version1_5_0.AtLeast(protocol.Version1_3_0) {
		t.Error("1.5.0 should be AtLeast 1.3.0")
	}
	if !protocol.Version1_5_0.AtLeast(protocol.Version1_5_0) {
		t.Error("1.5.0 should be AtLeast 1.5.0")
	}
	if protocol.Version1_3_0.AtLeast(protocol.Version1_5_0) {
		t.Error("1.3.0 should NOT be AtLeast 1.5.0")
	}
}

func TestVersionLatest(t *testing.T) {
	if protocol.VersionLatest != protocol.Version1_5_0 {
		t.Errorf("VersionLatest = %v, want 1.5.0", protocol.VersionLatest)
	}
}
