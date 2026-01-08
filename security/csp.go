package security

import "strings"

// CSP builds a Content-Security-Policy header value.
type CSP struct {
	directives map[string][]string
	order      []string
}

// NewCSP creates a CSP builder.
func NewCSP() *CSP {
	return &CSP{directives: make(map[string][]string)}
}

// Set replaces a directive with the provided values.
func (c *CSP) Set(directive string, values ...string) *CSP {
	if c == nil {
		return nil
	}
	directive = normalizeDirective(directive)
	if directive == "" {
		return c
	}
	if c.directives == nil {
		c.directives = make(map[string][]string)
	}
	if _, ok := c.directives[directive]; !ok {
		c.order = append(c.order, directive)
	}
	c.directives[directive] = filterValues(values)
	return c
}

// Add appends values to an existing directive.
func (c *CSP) Add(directive string, values ...string) *CSP {
	if c == nil {
		return nil
	}
	directive = normalizeDirective(directive)
	if directive == "" {
		return c
	}
	if c.directives == nil {
		c.directives = make(map[string][]string)
	}
	if _, ok := c.directives[directive]; !ok {
		c.order = append(c.order, directive)
	}
	c.directives[directive] = append(c.directives[directive], filterValues(values)...)
	return c
}

// DefaultSrc sets the default-src directive.
func (c *CSP) DefaultSrc(values ...string) *CSP {
	return c.Set("default-src", values...)
}

// ScriptSrc sets the script-src directive.
func (c *CSP) ScriptSrc(values ...string) *CSP {
	return c.Set("script-src", values...)
}

// StyleSrc sets the style-src directive.
func (c *CSP) StyleSrc(values ...string) *CSP {
	return c.Set("style-src", values...)
}

// ImgSrc sets the img-src directive.
func (c *CSP) ImgSrc(values ...string) *CSP {
	return c.Set("img-src", values...)
}

// ConnectSrc sets the connect-src directive.
func (c *CSP) ConnectSrc(values ...string) *CSP {
	return c.Set("connect-src", values...)
}

// FontSrc sets the font-src directive.
func (c *CSP) FontSrc(values ...string) *CSP {
	return c.Set("font-src", values...)
}

// FrameAncestors sets the frame-ancestors directive.
func (c *CSP) FrameAncestors(values ...string) *CSP {
	return c.Set("frame-ancestors", values...)
}

// ObjectSrc sets the object-src directive.
func (c *CSP) ObjectSrc(values ...string) *CSP {
	return c.Set("object-src", values...)
}

// BaseURI sets the base-uri directive.
func (c *CSP) BaseURI(values ...string) *CSP {
	return c.Set("base-uri", values...)
}

// FormAction sets the form-action directive.
func (c *CSP) FormAction(values ...string) *CSP {
	return c.Set("form-action", values...)
}

// UpgradeInsecureRequests adds the upgrade-insecure-requests directive.
func (c *CSP) UpgradeInsecureRequests() *CSP {
	return c.Set("upgrade-insecure-requests")
}

// String returns the policy string.
func (c *CSP) String() string {
	if c == nil {
		return ""
	}
	parts := make([]string, 0, len(c.order))
	for _, directive := range c.order {
		values := c.directives[directive]
		if len(values) == 0 {
			parts = append(parts, directive)
			continue
		}
		parts = append(parts, directive+" "+strings.Join(values, " "))
	}
	return strings.Join(parts, "; ")
}

func normalizeDirective(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.ToLower(value)
}

func filterValues(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	return filtered
}
