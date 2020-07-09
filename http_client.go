package nacos

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/google/uuid"
	"github.com/magicdvd/nacos-client/constant"
)

type httpClient struct {
	addr            string
	contextPath     string
	accessToken     string
	accessTokenTTL  int64
	lastRefreshTime time.Time
	username        string
	password        string
	*http.Client
	log LogInterface
}

func (c *httpClient) api(method, apiURI string, params, body *paramMap) ([]byte, error) {
	headers := map[string]string{}
	headers["Client-Version"] = constant.ClientVersion
	headers["User-Agent"] = constant.ClientVersion
	//headers["Accept-Encoding"] = "gzip,deflate,sdch"}
	headers["Connection"] = "Keep-Alive"
	uuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	headers["RequestId"] = uuid.String()
	headers["Request-Module"] = "Naming"
	headers["Content-Type"] = "application/x-www-form-urlencoded;charset=utf-8"
	query, bodyData := url.Values{}, url.Values{}
	if params != nil {
		query = params.Parse()
	}
	if body != nil {
		bodyData = body.Parse()
	}
	if c.accessToken != "" {
		query.Set(constant.AccessToken, c.accessToken)
	}
	return c.do(method, c.addr+c.contextPath+apiURI, headers, query, bodyData)
}

func (c *httpClient) do(method, target string, headers map[string]string, params, body url.Values) ([]byte, error) {
	if len(params) > 0 {
		target += "?" + params.Encode()
	}
	var req *http.Request
	var err error
	if len(body) > 0 {
		req, err = http.NewRequest(method, target, strings.NewReader(body.Encode()))
	} else {
		req, err = http.NewRequest(method, target, nil)
	}
	if err != nil {
		c.log.Error("do(NewRequest)", "httpClient", target, body, err)
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	c.log.Debug("do(clientDo)", method, target, headers, params, body)
	resp, err := c.Client.Do(req)
	if err != nil {
		c.log.Error("do(clientDo)", "httpClient", target, body, err)
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("do(readAll)", "httpClient", target, err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		c.log.Error("do(statusCode)", "httpClient", target, body, resp.StatusCode, string(b))
		return nil, errors.New(string(b))
	}
	c.log.Debug("do(resp)", string(b))
	return b, nil
}

func (c *httpClient) refreshLogin() {
	for {
		if c.accessTokenTTL == 0 {
			if err := c.login(); err != nil {
				return
			}
		}
		<-time.After(time.Duration(c.accessTokenTTL) * time.Second * 9 / 10)
		if err := c.login(); err != nil {
			return
		}
	}
}

func (c *httpClient) login() error {
	params := url.Values{}
	params.Set("username", c.username)
	body := url.Values{}
	body.Set("password", c.password)
	b, err := c.do(http.MethodPost, c.addr+c.contextPath+constant.APILoginPath, map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, params, body)
	if err != nil {
		return err
	}
	accessToken, err := jsonparser.GetString(b, constant.AccessToken)
	if err != nil {
		return err
	}
	accessTokenTTL, err := jsonparser.GetInt(b, constant.AccessTokenTTL)
	if err != nil {
		return err
	}
	if accessTokenTTL == 0 {
		return errors.New("accessTokenTTL is empty")
	}
	c.accessToken = accessToken
	c.accessTokenTTL = accessTokenTTL
	c.lastRefreshTime = time.Now()
	return nil
}
