package protocol

const (
	defaultOptimalWidth      = 1024
	defaultOptimalHeight     = 768
	defaultOptimalResolution = 96
)

// ClientInfo holds client capability information sent during the handshake.
type ClientInfo struct {
	OptimalWidth      int // default 1024
	OptimalHeight     int // default 768
	OptimalResolution int // default 96 DPI
	AudioMimetypes    []string
	VideoMimetypes    []string
	ImageMimetypes    []string
	Timezone          string // IANA timezone; sent if protocol >= 1.1.0
	Name              string // username; sent if protocol >= 1.5.0
	TimezoneSet       bool   // distinguishes explicit empty timezone from unset
	NameSet           bool   // distinguishes explicit empty name from unset
	OptimalWidthSet   bool   // distinguishes explicit zero width from unset
	OptimalHeightSet  bool   // distinguishes explicit zero height from unset
	OptimalResSet     bool   // distinguishes explicit zero DPI from unset
}

// NewClientInfo returns client information with Java guacamole-common defaults.
// Image mimetypes default to ["image/png","image/jpeg"] matching Java's
// GuacamoleClientInformation; without this, guacd < 1.5.0 cannot encode
// display bitmaps and the browser sees only the cursor on a black screen.
func NewClientInfo() *ClientInfo {
	return &ClientInfo{
		OptimalWidth:      defaultOptimalWidth,
		OptimalHeight:     defaultOptimalHeight,
		OptimalResolution: defaultOptimalResolution,
		AudioMimetypes:    []string{},
		VideoMimetypes:    []string{},
		ImageMimetypes:    []string{"image/png", "image/jpeg"},
	}
}

// GetOptimalScreenWidth returns the optimal screen width requested by the
// client, in pixels.
func (i *ClientInfo) GetOptimalScreenWidth() int {
	if i.OptimalWidth != 0 || i.OptimalWidthSet {
		return i.OptimalWidth
	}
	return defaultOptimalWidth
}

// SetOptimalScreenWidth sets the client's optimal screen width.
func (i *ClientInfo) SetOptimalScreenWidth(width int) {
	i.OptimalWidth = width
	i.OptimalWidthSet = true
}

// GetOptimalScreenHeight returns the optimal screen height requested by the
// client, in pixels.
func (i *ClientInfo) GetOptimalScreenHeight() int {
	if i.OptimalHeight != 0 || i.OptimalHeightSet {
		return i.OptimalHeight
	}
	return defaultOptimalHeight
}

// SetOptimalScreenHeight sets the client's optimal screen height.
func (i *ClientInfo) SetOptimalScreenHeight(height int) {
	i.OptimalHeight = height
	i.OptimalHeightSet = true
}

// GetOptimalResolution returns the optimal screen resolution, in DPI.
func (i *ClientInfo) GetOptimalResolution() int {
	if i.OptimalResolution != 0 || i.OptimalResSet {
		return i.OptimalResolution
	}
	return defaultOptimalResolution
}

// SetOptimalResolution sets the optimal screen resolution, in DPI.
func (i *ClientInfo) SetOptimalResolution(resolution int) {
	i.OptimalResolution = resolution
	i.OptimalResSet = true
}

// GetAudioMimetypes returns the live list of supported audio mimetypes.
func (i *ClientInfo) GetAudioMimetypes() []string {
	if i.AudioMimetypes == nil {
		i.AudioMimetypes = []string{}
	}
	return i.AudioMimetypes
}

// GetVideoMimetypes returns the live list of supported video mimetypes.
func (i *ClientInfo) GetVideoMimetypes() []string {
	if i.VideoMimetypes == nil {
		i.VideoMimetypes = []string{}
	}
	return i.VideoMimetypes
}

// GetImageMimetypes returns the live list of supported image mimetypes.
func (i *ClientInfo) GetImageMimetypes() []string {
	if i.ImageMimetypes == nil {
		i.ImageMimetypes = []string{}
	}
	return i.ImageMimetypes
}

// GetTimezone returns the client timezone, if configured.
func (i *ClientInfo) GetTimezone() string {
	return i.Timezone
}

// SetTimezone sets the client timezone, marking it as explicitly configured.
func (i *ClientInfo) SetTimezone(timezone string) {
	i.Timezone = timezone
	i.TimezoneSet = true
}

// TimezoneValue returns the client timezone and whether it is configured.
func (i *ClientInfo) TimezoneValue() (string, bool) {
	return i.Timezone, i.Timezone != "" || i.TimezoneSet
}

// GetName returns the name of the Guacamole user reported by the client, if
// configured.
func (i *ClientInfo) GetName() string {
	return i.Name
}

// SetName sets the client name, marking it as explicitly configured.
func (i *ClientInfo) SetName(name string) {
	i.Name = name
	i.NameSet = true
}

// NameValue returns the client name and whether it is configured.
func (i *ClientInfo) NameValue() (string, bool) {
	return i.Name, i.Name != "" || i.NameSet
}
