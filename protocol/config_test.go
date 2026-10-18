package protocol_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/hq0101/guacamole-common-go/protocol"
)

func TestConfig_Fields(t *testing.T) {
	cfg := protocol.Config{
		Protocol:     "rdp",
		ConnectionID: "",
		Params:       map[string]string{"hostname": "127.0.0.1"},
	}
	if cfg.Protocol != "rdp" {
		t.Errorf("Protocol = %q, want rdp", cfg.Protocol)
	}
}

func TestConfig_SetConnectionIDMarksEmptyValueConfigured(t *testing.T) {
	cfg := &protocol.Config{}
	cfg.SetConnectionID("")

	if !cfg.ConnectionIDSet {
		t.Error("ConnectionIDSet = false, want true after SetConnectionID")
	}
	if cfg.ConnectionID != "" {
		t.Errorf("ConnectionID = %q, want empty string", cfg.ConnectionID)
	}
}

func TestConfig_AccessorsMatchJavaConfigurationNames(t *testing.T) {
	cfg := &protocol.Config{}
	cfg.SetProtocol("vnc")
	cfg.SetConnectionID("$connection")

	if got := cfg.GetProtocol(); got != "vnc" {
		t.Errorf("GetProtocol() = %q, want vnc", got)
	}
	if got := cfg.GetConnectionID(); got != "$connection" {
		t.Errorf("GetConnectionID() = %q, want $connection", got)
	}
	if !cfg.ConnectionIDSet {
		t.Error("ConnectionIDSet = false, want true after SetConnectionID")
	}
}

func TestNewConfigInitializesMutableParameterMap(t *testing.T) {
	cfg := protocol.NewConfig()
	params := cfg.GetParameters()
	params["hostname"] = "guacd.example"

	if got := cfg.GetParameter("hostname"); got != "guacd.example" {
		t.Errorf("GetParameter(hostname) = %q, want guacd.example", got)
	}
	if _, ok := cfg.Parameter("missing"); ok {
		t.Error("Parameter(missing) ok = true, want false")
	}
}

func TestNewConfigFromCopiesIndependentConfiguration(t *testing.T) {
	original := protocol.NewConfig()
	original.Protocol = "rdp"
	original.SetConnectionID("")
	original.SetParameter("hostname", "server.example")

	clone := protocol.NewConfigFrom(original)
	original.Protocol = "ssh"
	original.SetConnectionID("$changed")
	original.SetParameter("hostname", "changed.example")

	if clone.Protocol != "rdp" {
		t.Errorf("Protocol = %q, want rdp", clone.Protocol)
	}
	if !clone.ConnectionIDSet {
		t.Error("ConnectionIDSet = false, want true")
	}
	if clone.ConnectionID != "" {
		t.Errorf("ConnectionID = %q, want empty string", clone.ConnectionID)
	}
	if got := clone.GetParameter("hostname"); got != "server.example" {
		t.Errorf("GetParameter(hostname) = %q, want server.example", got)
	}
}

func TestConfig_ParameterMethodsMatchJavaSemantics(t *testing.T) {
	cfg := &protocol.Config{}
	cfg.SetParameter("hostname", "server.example")
	cfg.SetParameter("empty", "")

	if got := cfg.GetParameter("hostname"); got != "server.example" {
		t.Errorf("GetParameter(hostname) = %q, want server.example", got)
	}
	if got := cfg.GetParameter("missing"); got != "" {
		t.Errorf("GetParameter(missing) = %q, want empty string", got)
	}
	if got, ok := cfg.Parameter("empty"); !ok || got != "" {
		t.Errorf("Parameter(empty) = %q, %v; want empty string, true", got, ok)
	}

	names := cfg.GetParameterNames()
	sort.Strings(names)
	if want := []string{"empty", "hostname"}; !reflect.DeepEqual(names, want) {
		t.Errorf("GetParameterNames() = %v, want %v", names, want)
	}

	cfg.UnsetParameter("hostname")
	if _, ok := cfg.Parameter("hostname"); ok {
		t.Error("Parameter(hostname) ok = true after UnsetParameter, want false")
	}

	source := map[string]string{"port": "3389"}
	cfg.SetParameters(source)
	source["port"] = "3390"
	if got := cfg.GetParameter("port"); got != "3389" {
		t.Errorf("GetParameter(port) = %q, want copied value 3389", got)
	}
	if _, ok := cfg.Parameter("empty"); ok {
		t.Error("Parameter(empty) ok = true after SetParameters replacement, want false")
	}

	live := cfg.GetParameters()
	live["username"] = "guac"
	if got := cfg.GetParameter("username"); got != "guac" {
		t.Errorf("GetParameter(username) = %q, want guac", got)
	}
}

func TestClientInfo_Fields(t *testing.T) {
	info := protocol.ClientInfo{
		OptimalWidth:      1920,
		OptimalHeight:     1080,
		OptimalResolution: 96,
		Timezone:          "Asia/Shanghai",
		Name:              "user1",
	}
	if info.Timezone != "Asia/Shanghai" {
		t.Errorf("Timezone = %q", info.Timezone)
	}
}

func TestNewClientInfoDefaultsMatchJava(t *testing.T) {
	info := protocol.NewClientInfo()

	if info.OptimalWidth != 1024 {
		t.Errorf("OptimalWidth = %d, want 1024", info.OptimalWidth)
	}
	if info.OptimalHeight != 768 {
		t.Errorf("OptimalHeight = %d, want 768", info.OptimalHeight)
	}
	if info.OptimalResolution != 96 {
		t.Errorf("OptimalResolution = %d, want 96", info.OptimalResolution)
	}
	if info.AudioMimetypes == nil || len(info.AudioMimetypes) != 0 {
		t.Errorf("AudioMimetypes = %v, want empty non-nil slice", info.AudioMimetypes)
	}
	if info.VideoMimetypes == nil || len(info.VideoMimetypes) != 0 {
		t.Errorf("VideoMimetypes = %v, want empty non-nil slice", info.VideoMimetypes)
	}
	wantImage := []string{"image/png", "image/jpeg"}
	if len(info.ImageMimetypes) != len(wantImage) || info.ImageMimetypes[0] != wantImage[0] || info.ImageMimetypes[1] != wantImage[1] {
		t.Errorf("ImageMimetypes = %v, want %v (Java GuacamoleClientInformation default)", info.ImageMimetypes, wantImage)
	}
}

func TestClientInfo_JavaAccessorsExposeDefaultsAndLiveMimetypeSlices(t *testing.T) {
	info := &protocol.ClientInfo{}

	if got := info.GetOptimalScreenWidth(); got != 1024 {
		t.Errorf("GetOptimalScreenWidth() = %d, want 1024", got)
	}
	if got := info.GetOptimalScreenHeight(); got != 768 {
		t.Errorf("GetOptimalScreenHeight() = %d, want 768", got)
	}
	if got := info.GetOptimalResolution(); got != 96 {
		t.Errorf("GetOptimalResolution() = %d, want 96", got)
	}

	audio := info.GetAudioMimetypes()
	video := info.GetVideoMimetypes()
	image := info.GetImageMimetypes()
	if audio == nil || video == nil || image == nil {
		t.Fatalf("mimetype getters returned nil slices: audio=%v video=%v image=%v", audio, video, image)
	}

	info.AudioMimetypes = []string{"audio/L16"}
	audio = info.GetAudioMimetypes()
	audio[0] = "audio/ogg"
	if got := info.AudioMimetypes[0]; got != "audio/ogg" {
		t.Errorf("GetAudioMimetypes() did not expose live slice element, got %q", got)
	}
}

func TestClientInfo_JavaAccessorsPreserveExplicitZeroDimensions(t *testing.T) {
	info := protocol.NewClientInfo()
	info.SetOptimalScreenWidth(0)
	info.SetOptimalScreenHeight(0)
	info.SetOptimalResolution(0)

	if got := info.GetOptimalScreenWidth(); got != 0 {
		t.Errorf("GetOptimalScreenWidth() = %d, want 0", got)
	}
	if got := info.GetOptimalScreenHeight(); got != 0 {
		t.Errorf("GetOptimalScreenHeight() = %d, want 0", got)
	}
	if got := info.GetOptimalResolution(); got != 0 {
		t.Errorf("GetOptimalResolution() = %d, want 0", got)
	}
}

func TestClientInfo_SettersMarkEmptyValuesConfigured(t *testing.T) {
	info := &protocol.ClientInfo{}
	info.SetTimezone("")
	info.SetName("")

	if !info.TimezoneSet {
		t.Error("TimezoneSet = false, want true after SetTimezone")
	}
	if !info.NameSet {
		t.Error("NameSet = false, want true after SetName")
	}
	if got, ok := info.TimezoneValue(); !ok || got != "" {
		t.Errorf("TimezoneValue() = %q, %v; want empty string, true", got, ok)
	}
	if got, ok := info.NameValue(); !ok || got != "" {
		t.Errorf("NameValue() = %q, %v; want empty string, true", got, ok)
	}
	if got := info.GetTimezone(); got != "" {
		t.Errorf("GetTimezone() = %q, want empty string", got)
	}
	if got := info.GetName(); got != "" {
		t.Errorf("GetName() = %q, want empty string", got)
	}
}

func TestFilter_ErrDropInstruction(t *testing.T) {
	var f protocol.Filter = func(instr protocol.Instruction) (protocol.Instruction, error) {
		return protocol.Instruction{}, protocol.ErrDropInstruction
	}
	_, err := f(protocol.Instruction{Opcode: "test"})
	if err != protocol.ErrDropInstruction {
		t.Errorf("expected ErrDropInstruction, got %v", err)
	}
}
