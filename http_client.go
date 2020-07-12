package nacos

import (
	"encoding/json"
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
	enableLog bool
	log       LogInterface
}

func (c *httpClient) api(method, apiURI string, params, body *paramMap) ([]byte, error) {
	headers := map[string]string{}
	headers["Client-Version"] = constant.ClientVersion
	headers["User-Agent"] = constant.ClientVersion
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

func di(method, target string, header map[string]string, body url.Values, err ...error) []interface{} {
	ps := []interface{}{}
	if len(err) > 0 {
		ps = append(ps, err[0].Error())
	}
	ps = append(ps, method, target)
	b, _ := json.Marshal(header)
	bStr := ""
	if len(body) > 0 {
		bStr = body.Encode()
	}
	ps = append(ps, "header:"+string(b), "body:"+bStr)
	return ps
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
		c.log.Error("httpClientDo(NewRequest)", di(method, target, headers, body, err)...)
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	if c.enableLog {
		c.log.Debug("httpClientDo(clientDo)", di(method, target, headers, body)...)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		c.log.Error("httpClientDo(clientDo)", di(method, target, headers, body, err)...)
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("httpClientDo(readAll)", di(method, target, headers, body, err)...)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		err := errors.New(string(b))
		c.log.Error("httpClientDo(statusCode)", di(method, target, headers, body, err)...)
		return nil, err
	}
	if c.enableLog {
		c.log.Debug("httpClientDo(resp)", string(b))
	}
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
