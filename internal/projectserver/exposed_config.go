package projectserver

type IndividualServerConfig struct {
	MaxWebSocketPerIp      int  `json:"maxWebsocketPerIp,omitempty"`
	IgnoreInstalledBrowser bool `json:"ignoreInstalledBrowser,omitempty"`

	ProjectsDir string `json:"projectsDir,omitempty"` //if not set, defaults to filepath.Join(config.USER_HOME, "inox-projects")
	ProdDir     string `json:"prodDir,omitempty"`     //if not set deployment in production is not allowed

	BehindCloudProxy       bool `json:"behindCloudProxy,omitempty"`
	Port                   int  `json:"port,omitempty"`
	BindToAllInterfaces    bool `json:"bindToAllInterfaces,omitempty"`
	ExposeWebServers       bool `json:"exposeWebServers,omitempty"`
	AllowBrowserAutomation bool `json:"allowBrowserAutomation,omitempty"`

	//If not empty the HTTP permissions granted to the server are read|write|delete on http(s)://<domain> (for each domain).
	//If empty the project server is allowed to make any HTTP request.
	DomainAllowList []string `json:"domainAllowList"`
}
