package email

import (
	"archive/zip"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

//go:generate ../../../hack/email/disposable_domains.sh
//go:embed disposable_domains.zip
var domainsZip embed.FS

type (
	optConfig struct {
		checkMX        bool
		checkMXTimeout time.Duration
	}

	Option func(config *optConfig)
)

const (
	PortSMTP           = 25
	PortSMTPSubmission = 587
)

var (
	// rfc5322: https://stackoverflow.com/a/201378/5405453.
	rfc5322            = "(?i)(?:[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*|\"(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21\\x23-\\x5b\\x5d-\\x7f]|\\\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])*\")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\\[(?:(?:(2(5[0-5]|[0-4][0-9])|1[0-9][0-9]|[1-9]?[0-9]))\\.){3}(?:(2(5[0-5]|[0-4][0-9])|1[0-9][0-9]|[1-9]?[0-9])|[a-z0-9-]*[a-z0-9]:(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21-\\x5a\\x53-\\x7f]|\\\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])+)\\])"
	validRfc5322Regexp = regexp.MustCompile(fmt.Sprintf("^%s*$", rfc5322))
	smtpPorts          = []int{PortSMTPSubmission, PortSMTP}
)

// Validate validates the email address is RFC5322
// Optionally check if we can dial the MX record domains
func Validate(emailAddress string, options ...Option) error {
	cfg := &optConfig{}
	for _, applyOption := range options {
		applyOption(cfg)
	}

	found := validRfc5322Regexp.Find([]byte(emailAddress))
	if string(found) != emailAddress {
		return errors.New("not RFC522 compliant")
	}

	// Split the email into two parts: local and domain, separated by '@'
	parts := strings.Split(emailAddress, "@")
	if len(parts) != 2 {
		return errors.New("missing @")
	}

	domain := parts[1]
	if err := checkDisposableDomains(domain); err != nil {
		return err
	}

	if cfg.checkMX {
		return checkMXRecords(domain, cfg.checkMXTimeout)
	}

	return nil
}

func checkDisposableDomains(domain string) error {
	fb, err := domainsZip.ReadFile("disposable_domains.zip")
	if err != nil {
		return err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(fb), int64(len(fb)))
	if err != nil {
		return fmt.Errorf("failed to read disposable domains list for verification: %w", err)
	}

	for _, file := range zipReader.File {
		if file.Name == "disposable_domains.json" {
			f, err := file.Open()
			if err != nil {
				return err
			}
			defer f.Close()

			var domains []string
			if err := json.NewDecoder(f).Decode(&domains); err != nil {
				return fmt.Errorf("disposable domains list improperly formatted: %w", err)
			}

			if slices.Contains(domains, domain) {
				return errors.New("disposable domains are not allowed")
			}
		}
	}

	return nil
}

func WithCheckMX() Option {
	return func(c *optConfig) {
		c.checkMX = true
		c.checkMXTimeout = time.Second
	}
}

func WithCheckMXTimeout(duration time.Duration) Option {
	return func(c *optConfig) {
		c.checkMX = true
		c.checkMXTimeout = duration
	}
}

// checkMXRecords tries to make a connection with one of the MX records for the given domain
// times out after the given duration.
func checkMXRecords(domain string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	records, err := net.DefaultResolver.LookupMX(ctx, domain)
	if err != nil {
		return err
	}

	checkCount := len(records) * len(smtpPorts)
	errc := make(chan error, checkCount)
	wg := sync.WaitGroup{}
	wg.Add(checkCount)

	// Fire off a go routine for each record and SMTP port
	// The first to return a nil error will return success
	go func() {
		defer close(errc)
		for _, record := range records {
			for _, port := range smtpPorts {
				go func() {
					defer wg.Done()
					errc <- tryDial(ctx, fmt.Sprintf("%s:%d", record.Host, port))
				}()
			}
		}
		wg.Wait()
	}()

	for {
		select {
		case <-ctx.Done():
			return errors.New("timed out trying to verify email address server")
		case err, ok := <-errc:
			if ok && err == nil {
				cancel()
				return nil
			}
		}
	}
}

func tryDial(ctx context.Context, addr string) error {
	// Using a net.Dialer instead of smtp.Dial to pass our context
	var d net.Dialer
	client, connectErr := d.DialContext(ctx, "tcp", addr)

	if connectErr != nil {
		return connectErr
	}
	defer client.Close()

	return nil
}
