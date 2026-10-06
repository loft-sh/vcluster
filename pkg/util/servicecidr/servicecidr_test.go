package servicecidr

import (
	"net"
	"testing"
)

func TestDetectPrefix(t *testing.T) {
	testCases := []struct {
		name      string
		ip        net.IP
		isInRange func(ip net.IP) bool
		expected  string
	}{
		{
			name: "simple 1",
			ip:   net.ParseIP("192.168.194.208"),
			isInRange: func(ip net.IP) bool {
				return (&net.IPNet{IP: net.ParseIP("192.168.194.128"), Mask: net.CIDRMask(25, 32)}).Contains(ip)
			},
			expected: "192.168.194.128/25",
		},
		{
			name: "simple 2",
			ip:   net.ParseIP("192.168.1.5"),
			isInRange: func(ip net.IP) bool {
				return (&net.IPNet{IP: net.ParseIP("192.168.1.0"), Mask: net.CIDRMask(24, 32)}).Contains(ip)
			},
			expected: "192.168.1.0/24",
		},
		{
			name: "simple 3",
			ip:   net.ParseIP("10.0.0.128"),
			isInRange: func(ip net.IP) bool {
				return (&net.IPNet{IP: net.ParseIP("10.0.0.128"), Mask: net.CIDRMask(25, 32)}).Contains(ip)
			},
			expected: "10.0.0.128/25",
		},
		{
			name: "simple 4",
			ip:   net.ParseIP("172.16.5.33"),
			isInRange: func(ip net.IP) bool {
				return (&net.IPNet{IP: net.ParseIP("172.16.5.32"), Mask: net.CIDRMask(27, 32)}).Contains(ip)
			},
			expected: "172.16.5.32/27",
		},
		{
			name: "simple 5",
			ip:   net.ParseIP("192.168.194.225"),
			isInRange: func(ip net.IP) bool {
				return (&net.IPNet{IP: net.ParseIP("192.168.194.128"), Mask: net.CIDRMask(25, 32)}).Contains(ip)
			},
			expected: "192.168.194.128/25",
		},
		{
			name: "simple 1 ipv6",
			ip:   net.ParseIP("2001:db8::1"),
			isInRange: func(ip net.IP) bool {
				return (&net.IPNet{IP: net.ParseIP("2001:db8::"), Mask: net.CIDRMask(32, 128)}).Contains(ip)
			},
			expected: "2001:db8::/32",
		},
		{
			name: "simple 2 ipv6",
			ip:   net.ParseIP("2001:db8:abcd:1::"),
			isInRange: func(ip net.IP) bool {
				return (&net.IPNet{IP: net.ParseIP("2001:db8:abcd:1::"), Mask: net.CIDRMask(64, 128)}).Contains(ip)
			},
			expected: "2001:db8:abcd:1::/64",
		},
		{
			name: "simple 3 ipv6",
			ip:   net.ParseIP("fe80::1234:5678"),
			isInRange: func(ip net.IP) bool {
				return (&net.IPNet{IP: net.ParseIP("fe80::"), Mask: net.CIDRMask(10, 128)}).Contains(ip)
			},
			expected: "fe80::/10",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			prefix := DetectPrefix(testCase.ip, testCase.isInRange)
			if prefix != testCase.expected {
				t.Errorf("expected %s, got %s", testCase.expected, prefix)
			}
		})
	}
}
