package emailverifier

import "net"

// Mx is detail about the Mx host
type Mx struct {
	HasMXRecord bool      // whether has 1 or more MX record
	Records     []*net.MX // represent DNS MX records
}

// CheckMX will return the DNS MX records for the given domain name sorted by preference.
func (v *Verifier) CheckMX(domain string) (*Mx, error) {
	domain = domainToASCII(domain)
	mx, err := net.LookupMX(domain)
	if err != nil && len(mx) == 0 {
		return nil, err
	}
	return &Mx{
		HasMXRecord: len(mx) > 0,
		Records:     mx,
	}, nil
}
