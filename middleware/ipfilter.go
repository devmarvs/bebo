package middleware

import (
	"net"
	"strings"

	"github.com/devmarvs/bebo"
	"github.com/devmarvs/bebo/apperr"
)

type ipRule struct {
	net *net.IPNet
}

// IPFilterOptions configures IP allow/deny rules.
type IPFilterOptions struct {
	Allow        []string
	Deny         []string
	UseForwarded bool
}

// IPFilter blocks or allows requests based on IP allow/deny lists.
func IPFilter(options IPFilterOptions) (bebo.Middleware, error) {
	allow, err := parseIPRules(options.Allow)
	if err != nil {
		return nil, err
	}
	deny, err := parseIPRules(options.Deny)
	if err != nil {
		return nil, err
	}

	return func(next bebo.Handler) bebo.Handler {
		return func(ctx *bebo.Context) error {
			ip := clientIP(ctx, options.UseForwarded)
			if ip == nil {
				return apperr.Forbidden("ip not allowed", nil)
			}
			if matchIP(deny, ip) {
				return apperr.Forbidden("ip not allowed", nil)
			}
			if len(allow) > 0 && !matchIP(allow, ip) {
				return apperr.Forbidden("ip not allowed", nil)
			}
			return next(ctx)
		}
	}, nil
}

func parseIPRules(entries []string) ([]ipRule, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	rules := make([]ipRule, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			_, cidr, err := net.ParseCIDR(entry)
			if err != nil {
				return nil, err
			}
			rules = append(rules, ipRule{net: cidr})
			continue
		}
		ip := net.ParseIP(entry)
		if ip == nil {
			return nil, net.InvalidAddrError(entry)
		}
		maskBits := 32
		if ip.To4() == nil {
			maskBits = 128
		}
		netmask := net.CIDRMask(maskBits, maskBits)
		rules = append(rules, ipRule{net: &net.IPNet{IP: ip, Mask: netmask}})
	}

	return rules, nil
}

func matchIP(rules []ipRule, ip net.IP) bool {
	for _, rule := range rules {
		if rule.net.Contains(ip) {
			return true
		}
	}
	return false
}

func clientIP(ctx *bebo.Context, useForwarded bool) net.IP {
	if useForwarded {
		if forwarded := ctx.Request.Header.Get("X-Forwarded-For"); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			if len(parts) > 0 {
				ip := net.ParseIP(strings.TrimSpace(parts[0]))
				if ip != nil {
					return ip
				}
			}
		}
	}

	host, _, err := net.SplitHostPort(ctx.Request.RemoteAddr)
	if err != nil {
		return net.ParseIP(ctx.Request.RemoteAddr)
	}
	return net.ParseIP(host)
}
