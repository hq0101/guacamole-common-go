package protocol

// Config holds the connection configuration for a Guacamole session.
// Set ConnectionID to join an existing connection; leave ConnectionID unset to
// start a new one.
type Config struct {
	Protocol        string
	ConnectionID    string
	ConnectionIDSet bool
	Params          map[string]string
}

// NewConfig returns a blank connection configuration with an initialized
// parameter map, matching Java's default GuacamoleConfiguration constructor.
func NewConfig() *Config {
	return &Config{Params: map[string]string{}}
}

// NewConfigFrom copies an existing configuration, creating an independent
// configuration with the same protocol, connection ID, and parameters.
func NewConfigFrom(config *Config) *Config {
	clone := &Config{
		Protocol:        config.Protocol,
		ConnectionID:    config.ConnectionID,
		ConnectionIDSet: config.ConnectionIDSet,
	}
	clone.SetParameters(config.Params)
	return clone
}

// GetConnectionID returns the ID of the existing connection being joined, if
// any.
func (c *Config) GetConnectionID() string {
	return c.ConnectionID
}

// SetConnectionID sets the ID of an existing connection to join. Passing an
// empty string still marks the value as explicitly configured, matching Java's
// distinction between null and an empty string.
func (c *Config) SetConnectionID(connectionID string) {
	c.ConnectionID = connectionID
	c.ConnectionIDSet = true
}

// GetProtocol returns the name of the protocol to be used.
func (c *Config) GetProtocol() string {
	return c.Protocol
}

// SetProtocol sets the name of the protocol to be used.
func (c *Config) SetProtocol(protocol string) {
	c.Protocol = protocol
}

// SetParameter sets the value for the parameter with the given name.
func (c *Config) SetParameter(name, value string) {
	c.ensureParams()[name] = value
}

// GetParameter returns the value set for the parameter with the given name. If
// no such parameter is present, the empty string is returned.
func (c *Config) GetParameter(name string) string {
	return c.ensureParams()[name]
}

// Parameter returns the value set for the parameter with the given name and
// whether that parameter is present.
func (c *Config) Parameter(name string) (string, bool) {
	value, ok := c.ensureParams()[name]
	return value, ok
}

// UnsetParameter removes the value set for the parameter with the given name.
func (c *Config) UnsetParameter(name string) {
	delete(c.ensureParams(), name)
}

// GetParameterNames returns the names of all currently defined parameters.
func (c *Config) GetParameterNames() []string {
	params := c.ensureParams()
	names := make([]string, 0, len(params))
	for name := range params {
		names = append(names, name)
	}
	return names
}

// GetParameters returns the live map of parameter name/value pairs. Changes to
// the returned map affect this configuration.
func (c *Config) GetParameters() map[string]string {
	return c.ensureParams()
}

// SetParameters replaces all current parameters with the given parameters.
func (c *Config) SetParameters(params map[string]string) {
	c.Params = make(map[string]string, len(params))
	for name, value := range params {
		c.Params[name] = value
	}
}

func (c *Config) ensureParams() map[string]string {
	if c.Params == nil {
		c.Params = map[string]string{}
	}
	return c.Params
}
