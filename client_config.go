package nacos

import (
	"net/http"
	"strings"

	"github.com/magicdvd/nacos-client/constant"
)

func (c *ServiceClient) PublishConfig(dataID string, group string, content string, params ...Param) error {
	query := newParamMap()
	query.Set(
		paramConfigDataID(dataID),
		paramConfigGroup(group),
		paramConfigContent(content),
		ParamConfigTenant(c.opts.defaultTenant),
	)
	query.Set(params...)
	_, err := c.client.api(http.MethodPost, constant.APIConfig, nil, query)
	if err != nil {
		c.log.Error("PublishConfig", "api", err)
		return err
	}
	return nil
}

func (c *ServiceClient) GetConfig(dataID string, group string, params ...Param) (string, error) {
	query := newParamMap()
	query.Set(
		paramConfigDataID(dataID),
		paramConfigGroup(group),
		ParamConfigTenant(c.opts.defaultTenant),
	)
	query.Set(params...)
	b, err := c.client.api(http.MethodGet, constant.APIConfig, query, nil)
	if err != nil {
		c.log.Error("GetConfig", "api", err)
		return "", err
	}
	return string(b), nil
}

func (c *ServiceClient) RemoveConfig(dataID string, group string, params ...Param) error {
	query := newParamMap()
	query.Set(
		paramConfigDataID(dataID),
		paramConfigGroup(group),
		ParamConfigTenant(c.opts.defaultTenant),
	)
	query.Set(params...)
	_, err := c.client.api(http.MethodDelete, constant.APIConfig, query, nil)
	if err != nil {
		c.log.Error("RemoveConfig", "api", err)
		return err
	}
	return nil
}

func (c *ServiceClient) ListenConfig(dataID string, group string, callback func(string), params ...Param) <-chan error {
	ch := make(chan error)
	query := newParamMap()
	query.Set(
		paramConfigDataID(dataID),
		paramConfigGroup(group),
		ParamConfigTenant(c.opts.defaultTenant),
	)
	query.ListenConfigs("")
	go func() {
		defer close(ch)
		for {
			b, err := c.client.listen(http.MethodPost, constant.APIConfigListen, c.opts.listenInterval, nil, query)
			if err != nil {
				c.log.Error("ListenConfig", "api", err)
				ch <- err
				return
			}
			nc := string(b)
			if strings.ToLower(strings.Trim(nc, " ")) == "" {
				continue
			}
			nb, err := c.GetConfig(dataID, group, params...)
			if err != nil {
				c.log.Error("ListenConfig", "api", err)
				ch <- err
				return
			}
			callback(nb)
			query.ListenConfigs(nb)
		}
	}()
	return ch
}
