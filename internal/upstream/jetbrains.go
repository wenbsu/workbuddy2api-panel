package upstream

import (
	"encoding/json"
	"github.com/wenbsu/workbuddy2api-panel/internal/auth"
	"github.com/wenbsu/workbuddy2api-panel/internal/session"
	"net/http"
	"strings"
)

// JetBrainsAdapter is the default server chat profile. Values describe the actual host
// installation; no device identity or device proof is synthesized here.
// Evidence: plugin 4.12.42888899, UserAgentHttpInterceptor.buildUserAgent
// (server.js.map) and CodewikiServiceImpl.buildAuthHeaders (out/extension/index.js).
// Other endpoints retain their existing implementation.
type JetBrainsAdapter struct {
	IDEType         string `json:"ide_type"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platform_version"`
	ProductName     string `json:"product_name"`
	ProductVersion  string `json:"product_version"`
	DeploymentType  string `json:"deployment_type"`
	AgentIntent     string `json:"agent_intent,omitempty"`
	AgentType       string `json:"agent_type,omitempty"`
}

func (j *JetBrainsAdapter) Headers(c *Client, req *http.Request, a *auth.Auth, meta ChatMeta) {
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if token := a.AccessTokenValue(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	set := func(key, value string) {
		if value != "" {
			req.Header.Set(key, value)
		}
	}
	set("X-Agent-Intent", j.AgentIntent)
	set("X-Agent-Type", j.AgentType)
	set("X-User-Id", a.UID)
	if a.EnterpriseID != "" {
		set("X-Enterprise-Id", a.EnterpriseID)
		set("X-Tenant-Id", a.EnterpriseID)
	} else {
		c.injectDeviceToken(req, a)
	}
	set("X-Domain", a.DomainValue())
	if j.Platform != "" {
		set("X-IDE-Type", j.IDEType)
		set("X-IDE-Name", j.Platform)
	}
	set("X-IDE-Version", j.PlatformVersion)
	set("X-Product-Version", j.ProductVersion)
	set("X-Product", j.DeploymentType)
	ua := ""
	appendUA := func(name, version string) {
		if name != "" {
			if version == "" {
				version = "unknown"
			}
			if ua != "" {
				ua += " "
			}
			ua += name + "/" + version
		}
	}
	appendUA(j.Platform, j.PlatformVersion)
	appendUA(j.ProductName, j.ProductVersion)
	if c.UserAgent != "" {
		ua = c.UserAgent
	}
	if ua == "" {
		ua = "workbuddy2api-panel"
	}
	set("User-Agent", ua)
	c.injectConversationHeaders(req, meta)
	// Root request IDs are conditional in the plugin; the gateway has no parent request here.
	req.Header.Del("X-Root-Request-ID")
	// Main model requests use the request ID, not the message ID.
	req.Header.Set("X-Request-ID", req.Header.Get("X-Conversation-Request-ID"))
	trace := session.NewMessageID()
	req.Header.Set("X-Request-Trace-Id", trace[:8]+"-"+trace[8:12]+"-"+trace[12:16]+"-"+trace[16:20]+"-"+trace[20:])
}

// modelHeaders derives routing metadata from the prepared request, never from a
// captured model value. Direct/custom providers need separate routing support.
func (j *JetBrainsAdapter) modelHeaders(req *http.Request, body []byte) {
	var payload struct {
		Model string `json:"model"`
	}
	if json.Unmarshal(body, &payload) == nil && payload.Model != "" {
		req.Header.Set("X-Model-ID", strings.TrimPrefix(payload.Model, "custom-local:"))
	}
}
