package netutil

import (
	"fmt"
	"net"
	"strings"
)

func ParseCIDRs(cidrs []string) ([]*net.IPNet, error) {
	if len(cidrs) == 0 {
		return nil, nil
	}

	out := make([]*net.IPNet, 0, len(cidrs))
	for _, raw := range cidrs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		if !strings.Contains(raw, "/") {
			ip := net.ParseIP(raw)
			if ip == nil {
				return nil, fmt.Errorf("invalid IP %q", raw)
			}

			if ip.To4() != nil {
				raw = ip.String() + "/32"
			} else {
				raw = ip.String() + "/128"
			}
		}

		_, network, err := net.ParseCIDR(raw)
		if err != nil {
			return nil, err
		}

		if ones, _ := network.Mask.Size(); ones == 0 {
			return nil, fmt.Errorf("%q covers all addresses", raw)
		}

		out = append(out, network)
	}

	return out, nil
}
