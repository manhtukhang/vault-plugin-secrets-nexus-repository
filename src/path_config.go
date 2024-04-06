package nxr

const (
	// pathConfigHelpSynopsis summarizes the help text for the configuration
	pathConfigHelpSynopsis = `Configure the Nexus Repository admin configuration.`

	// pathConfigHelpDescription describes the help text for the configuration
	pathConfigHelpDescription = `
The Nexus Repository secret backend requires credentials for managing user.

You must create a username ("username" parameter)
and password ("password" parameter)
and specify the Nexus Repository address ("url" parameter)
for the API before using this secrets backend.

An optional "insecure" parameter will enable bypassing
the TLS connection verification with Nexus Repository
(when server using self-signed certificate).

An optional "timeout" is the maximum time (in second)
to wait before the request times out.
`
	pathConfigAdmin = "config/admin"
)

// adminConfig includes the minimum configuration
// required to instantiate a new Nexus Repository client.
type adminConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url"`
	Insecure bool   `json:"insecure,omitempty"`
	Timeout  *int   `json:"timeout,omitempty"`
}
