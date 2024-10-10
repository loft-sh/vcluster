# email-verifier

✉️ A Go library for email verification without sending any emails.

[![Build Status](https://github.com/AfterShip/email-verifier/workflows/CI%20Actions/badge.svg)](https://github.com/AfterShip/email-verifier/actions)
[![Godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/AfterShip/email-verifier)
[![Coverage Status](https://coveralls.io/repos/github/AfterShip/email-verifier/badge.svg?t=VTgVfL)](https://coveralls.io/github/AfterShip/email-verifier)
[![Go Report Card](https://goreportcard.com/badge/github.com/AfterShip/email-verifier)](https://goreportcard.com/report/github.com/AfterShip/email-verifier)
[![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://github.com/AfterShip/email-verifier/blob/main/LICENSE)

## Features

- Email Address Validation: validates if a string contains a valid email.
- Email Verification Lookup via SMTP: performs an email verification on the passed email (catchAll detection enabled by default)
- MX Validation: checks the DNS MX records for the given domain name
- Misc Validation: including Free email provider check, Role account validation, Disposable emails address (DEA) validation
- Email Reachability: checks how confident in sending an email to the address

## Install

Use `go get` to install this package.

```shell script
go get -u github.com/AfterShip/email-verifier
```

## Usage

### Basic usage

Use `Verify` method to verify an email address with different dimensions

```go
package main

import (
	"fmt"
	
	emailverifier "github.com/AfterShip/email-verifier"
)

var (
	verifier = emailverifier.NewVerifier()
)


func main() {
	email := "example@exampledomain.org"

	ret, err := verifier.Verify(email)
	if err != nil {
		fmt.Println("verify email address failed, error is: ", err)
		return
	}
	if !ret.Syntax.Valid {
		fmt.Println("email address syntax is invalid")
		return
	}

	fmt.Println("email validation result", ret)
	/*
		result is:
		{
			"email":"example@exampledomain.org",
			"disposable":false,
			"reachable":"unknown",
			"role_account":false,
			"free":false,
			"syntax":{
			"username":"example",
				"domain":"exampledomain.org",
				"valid":true
			},
			"has_mx_records":true,
			"smtp":null,
			"gravatar":null
		}
	*/
}
```

### Email verification Lookup

Use `CheckSMTP` to performs an email verification lookup via SMTP.

```go
var (
    verifier = emailverifier.
        NewVerifier().
        EnableSMTPCheck()
)

func main() {

    domain := "domain.org"
    username := "username"
    ret, err := verifier.CheckSMTP(domain, username)
    if err != nil {
        fmt.Println("check smtp failed: ", err)
        return
    }

    fmt.Println("smtp validation result: ", ret)

}
```

If you want to disable catchAll checking, use the `DisableCatchAllCheck()` switch (in effect only when SMTP verification is enabled).

```go
 verifier = emailverifier.
        NewVerifier().
        EnableSMTPCheck().
        DisableCatchAllCheck()
```

> Note: because most of the ISPs block outgoing SMTP requests through port 25 to prevent email spamming, the module will not perform SMTP checking by default. You can initialize the verifier with  `EnableSMTPCheck()`  to enable such capability if port 25 is usable, 
> or use a socks proxy to connect over SMTP

### Use a SOCKS5 proxy to verify email 

Support setting a SOCKS5 proxy to verify the email, proxyURI should be in the format: `socks5://user:password@127.0.0.1:1080?timeout=5s`

The protocol could be socks5, socks4 and socks4a.

```go
var (
    verifier = emailverifier.
        NewVerifier().
        EnableSMTPCheck().
    	Proxy("socks5://user:password@127.0.0.1:1080?timeout=5s")
)

func main() {

    domain := "domain.org"
    username := "username"
    ret, err := verifier.CheckSMTP(domain, username)
    if err != nil {
        fmt.Println("check smtp failed: ", err)
        return
    }

    fmt.Println("smtp validation result: ", ret)

}
```

### Misc Validation

To check if an email domain is disposable via `IsDisposable`

```go
var (
    verifier = emailverifier.
        NewVerifier().
        EnableAutoUpdateDisposable()
)

func main() {
    domain := "domain.org"
    if verifier.IsDisposable(domain) {
        fmt.Printf("%s is a disposable domain\n", domain)
        return
    }
    fmt.Printf("%s is not a disposable domain\n", domain)
}
```

> Note: It is possible to automatically update the disposable domains daily by initializing verifier with `EnableAutoUpdateDisposable()`

### Suggestions for domain typo

Will check for typos in an email domain in addition to evaluating its validity. 
If we detect a possible typo, you will find a non-empty "suggestion" field in the validation result containing what we believe to be the correct domain.
Also, you can use the `SuggestDomain()` method alone to check the domain for possible misspellings

```go
func main() {
    domain := "gmai.com"
    suggestion := verifier.SuggestDomain(domain) 
    // suggestion should be `gmail.com`
    if suggestion != "" {
        fmt.Printf("domain %s is misspelled, right domain is %s. \n", domain, suggestion)
        return 
    }
    fmt.Printf("domain %s has no possible misspellings. \n", domain)
}

```

> Note: When using the `Verify()` method, domain typo checking is not enabled by default, you can enable it in a verifier with `EnableDomainSuggest()`
 
For more detailed documentation, please check on godoc.org 👉 [email-verifier](https://godoc.org/github.com/AfterShip/email-verifier)

## API 

We provide a simple **self-hosted** [API server](https://github.com/AfterShip/email-verifier/tree/main/cmd/apiserver) script for reference.

The API interface is very simple. All you need to do is to send a GET request with the following URL.

The `email` parameter would be the target email you want to verify.

`https://{your_host}/v1/{email}/verification`

## Similar Libraries Comparison

|                                     | [email-verifier](https://github.com/AfterShip/email-verifier) | [trumail](https://github.com/trumail/trumail) | [check-if-email-exists](https://reacher.email/) | [freemail](https://github.com/willwhite/freemail) |
| ----------------------------------- | :----------------------------------------------------------: | :-------------------------------------------: | :---------------------------------------------: | :-----------------------------------------------: |
| **Features**                        |                              〰️                              |                      〰️                       |                       〰️                        |                        〰️                         |
| Disposable email address validation |                              ✅                               |       ✅, but not available in free lib        |                        ✅                        |                         ✅                         |
| Disposable address autoupdate       |                              ✅                               |                       🤔                       |                        ❌                        |                         ❌                         |
| Free email provider check           |                              ✅                               |       ✅, but not available in free lib        |                        ❌                        |                         ✅                         |
| Role account validation             |                              ✅                               |                       ❌                       |                        ✅                        |                         ❌                         |
| Syntax validation                   |                              ✅                               |                       ✅                       |                        ✅                        |                         ❌                         |
| Email reachability                  |                              ✅                               |                       ✅                       |                        ✅                        |                         ❌                         |
| DNS records validation              |                              ✅                               |                       ✅                       |                        ✅                        |                         ❌                         |
| Email deliverability                |                              ✅                               |                       ✅                       |                        ✅                        |                         ❌                         |
| Mailbox disabled                    |                              ✅                               |                       ✅                       |                        ✅                        |                         ❌                         |
| Full inbox                          |                              ✅                               |                       ✅                       |                        ✅                        |                         ❌                         |
| Host exists                         |                              ✅                               |                       ✅                       |                        ✅                        |                         ❌                         |
| Catch-all                           |                              ✅                               |                       ✅                       |                        ✅                        |                         ❌                         |
| Gravatar                            |                              ✅                               |       ✅, but not available in free lib        |                        ❌                        |                         ❌                         |
| Typo check                          |                              ✅                              |       ✅, but not available in free lib        |                        ❌                        |                         ❌                         |
| Use proxy to connect over SMTP      |                              ✅                              |                        ❌                       |                        ✅                        |                         ❌                         |
| Honeyport dection                   |                              🔜                               |                       ❌                       |                        ❌                        |                         ❌                         |
| Bounce email check                  |                              🔜                               |                       ❌                       |                        ❌                        |                         ❌                         |
| **Tech**                            |                              〰️                              |                      〰️                       |                       〰️                        |                        〰️                         |
| Provide API                         |                              ✅                               |                       ✅                       |                        ✅                        |                         ❌                         |
| Free API                            |                              ✅                               |                       ❌                       |                        ❌                        |                         ❌                         |
| Language                            |                              Go                              |                      Go                       |                      Rust                       |                       JavaScript                        |
| Active maintain                     |                              ✅                               |                       ❌                       |                        ✅                        |                         ✅                         |
| High Performance                    |                              ✅                               |                       ❌                       |                        ✅                        |                         ✅                         |



## FAQ

#### The library hangs/takes a long time after 30 seconds when performing email verification lookup via SMTP

Most ISPs block outgoing SMTP requests through port 25 to prevent email spamming. `email-verifier` needs to have this port open to make a connection to the email's SMTP server. With the port being blocked, it is not possible to perform such checking, and it will instead hang until timeout error. Unfortunately, there is no easy workaround for this issue.

For more information, you may also visit [this StackOverflow thread](https://stackoverflow.com/questions/18139102/how-to-get-around-an-isp-block-on-port-25-for-smtp).

#### The output shows `"connection refused"` in the `smtp.error` field.

This error can also be due to SMTP ports being blocked by the ISP, see the above answer.

#### What does reachable: "unknown" means

This means that the server does not allow real-time verification of an email right now, or the email provider is a catch-all email server.

## Credits

- [trumail](https://github.com/trumail/trumail)
- [check-if-email-exists](https://github.com/amaurymartiny/check-if-email-exists)
- [mailcheck](https://github.com/mailcheck/mailcheck)
- disposable domains from [ivolo/disposable-email-domains](https://github.com/ivolo/disposable-email-domains)
- free provider data from [willwhite/freemail](https://github.com/willwhite/freemail)

## Contributing

For details on contributing to this repository, see the [contributing guide](https://github.com/AfterShip/email-verifier/blob/main/CONTRIBUTING.md).

## License

This package is licensed under MIT license. See [LICENSE](https://github.com/AfterShip/email-verifier/blob/main/LICENSE) for details.
