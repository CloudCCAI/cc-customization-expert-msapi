package httpclient

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	http *http.Client
}

func New() *Client {
	return &Client{
		http: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // match Node CLI behavior
			},
		},
	}
}

func (c *Client) PostEnvelope(url string, data map[string]any, header map[string]any, response any) error {
	if header == nil {
		header = map[string]any{"appVersion": "1.0.0"}
	}
	header["source"] = "cloudcc_cli"
	payload := map[string]any{
		"head": header,
		"body": data,
	}
	return c.postJSON(url, payload, nil, response, true)
}

func (c *Client) PostClass(url string, data any, accessToken string, response any) error {
	if accessToken == "" {
		return fmt.Errorf("OpenAPI Token is null. Please check your cloudcc-cli.config file or cache")
	}
	return c.postJSON(url, data, map[string]string{"accessToken": accessToken}, response, false)
}

func (c *Client) PostRaw(url string, data any, headers map[string]string, response any) error {
	return c.postJSON(url, data, headers, response, false)
}

func (c *Client) postJSON(url string, data any, headers map[string]string, response any, checkCode bool) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", res.StatusCode, string(resBody))
	}
	var raw map[string]any
	if checkCode {
		if err := json.Unmarshal(resBody, &raw); err == nil {
			code := numberOrString(raw["code"], "200")
			if code != "200" && code != "" {
				if msg, ok := raw["msg"].(string); ok && msg != "" {
					return errors.New(msg)
				}
				return fmt.Errorf("Unknown exception")
			}
		}
	}
	if response == nil {
		return nil
	}
	return json.Unmarshal(resBody, response)
}

func numberOrString(v any, fallback string) string {
	switch x := v.(type) {
	case nil:
		return fallback
	case string:
		return x
	case float64:
		return fmt.Sprintf("%.0f", x)
	default:
		return fmt.Sprint(x)
	}
}
