package netutil

import "testing"

func TestParseCIDRsRejectsWorldOpen(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"0.0.0.0/0", "::/0"} {
		if _, err := ParseCIDRs([]string{raw}); err == nil {
			t.Fatalf("%s must be rejected", raw)
		}
	}

	nets, err := ParseCIDRs([]string{"127.0.0.1", "10.0.0.0/8", "::1/128"})
	if err != nil || len(nets) != 3 {
		t.Fatalf("nets=%v err=%v", nets, err)
	}
}
