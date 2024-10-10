package emailverifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	SIGNUP_PAGE = "https://login.yahoo.com/account/create?specId=yidregsimplified&lang=en-US&src=&done=https%3A%2F%2Fwww.yahoo.com&display=login"
	SIGNUP_API  = "https://login.yahoo.com/account/module/create?validateField=userId"
	// USER_AGENT Fake one to use in API requests
	USER_AGENT = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/54.0.2840.71 Safari/537.36"
)

// Check yahoo email exists by their login & registration page.
// See https://login.yahoo.com
// See https://login.yahoo.com/account/create
func newYahooAPIVerifier(client *http.Client) smtpAPIVerifier {
	if client == nil {
		client = http.DefaultClient
	}
	return yahoo{
		client: client,
	}
}

type yahoo struct {
	client *http.Client
}

type yahooValidateReq struct {
	Domain, Username, Acrumb, SessionIndex string
	Cookies                                []*http.Cookie
}

type yahooErrorResp struct {
	Errors []errItem `json:"errors"`
}

type errItem struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

func (y yahoo) isSupported(host string) bool {
	// FIXME Is this `contains` too lenient?
	return strings.Contains(host, "yahoo")
}

func (y yahoo) check(domain, username string) (*SMTP, error) {
	cookies, signUpPageRespBytes, err := y.toSignUpPage()
	if err != nil {
		return nil, err
	}
	if len(cookies) == 0 {
		return nil, errors.New("yahoo check by api, no cookies")
	}

	acrumb := getAcrumb(cookies)
	if acrumb == "" {
		return nil, errors.New("yahoo check by api, no acrumb")
	}

	sessionIndex := getSessionIndex(signUpPageRespBytes)
	if sessionIndex == "" {
		return nil, errors.New("yahoo check by api, no sessionIndex")
	}

	yahooErrResp, err := y.sendValidateRequest(yahooValidateReq{
		Domain:       domain,
		Username:     username,
		Acrumb:       acrumb,
		SessionIndex: sessionIndex,
		Cookies:      cookies,
	})
	if err != nil {
		return nil, err
	}
	usernameExists := checkUsernameExists(yahooErrResp)
	return &SMTP{
		HostExists:  true,
		Deliverable: usernameExists,
	}, nil
}

func getSessionIndex(respBytes []byte) string {
	re := regexp.MustCompile(`value="([^"]+)" name="sessionIndex"`)
	match := re.FindSubmatch(respBytes)
	if len(match) > 1 {
		return string(match[1])
	}
	return ""
}

func checkUsernameExists(resp yahooErrorResp) bool {
	for _, item := range resp.Errors {
		if item.Name == "userId" && item.Error == "IDENTIFIER_EXISTS" {
			return true
		}
	}
	return false
}

func (y yahoo) sendValidateRequest(req yahooValidateReq) (yahooErrorResp, error) {
	var res yahooErrorResp
	data, err := json.Marshal(struct {
		Acrumb       string `json:"acrumb"`
		SpecId       string `json:"specId"`
		Yid          string `json:"userId"`
		SessionIndex string `json:"sessionIndex"`
		YidDomain    string `json:"yidDomain"`
	}{
		Acrumb:       req.Acrumb,
		SpecId:       "yidregsimplified",
		Yid:          req.Username,
		SessionIndex: req.SessionIndex,
		YidDomain:    req.Domain,
	})
	if err != nil {
		return res, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, SIGNUP_API, bytes.NewReader(data))
	if err != nil {
		return res, err
	}
	for _, c := range req.Cookies {
		request.AddCookie(c)
	}
	request.Header.Add("X-Requested-With", "XMLHttpRequest")
	request.Header.Add("Content-Type", "application/json; charset=UTF-8")
	resp, err := y.client.Do(request)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return res, err
	}
	return res, json.Unmarshal(respBytes, &res)
}

func (y yahoo) toSignUpPage() ([]*http.Cookie, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, SIGNUP_PAGE, nil)
	if err != nil {
		return nil, nil, err
	}
	request.Header.Add("User-Agent", USER_AGENT)
	resp, err := y.client.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	return resp.Cookies(), respBytes, err
}

func getAcrumb(cookies []*http.Cookie) string {
	for _, c := range cookies {
		re := regexp.MustCompile(`s=(?P<acrumb>[^;^&]*)`)
		match := re.FindStringSubmatch(c.Value)
		if len(match) > 1 {
			return match[1]
		}
	}
	return ""
}
