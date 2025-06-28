package httpclient

import (
	"bytes"
	"github.com/zeromicro/go-zero/core/logx"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var (
	MaxRetry    = 3
	MaxTimeOut  = 5000 //毫秒
	EmptyBuffer = &bytes.Buffer{}
)

func Get(url string, headers map[string]string, timeout int, retry int) ([]byte, error) {
	return requestWithTrys("GET", url, nil, headers, timeout, retry)
}

func Post(url string, params *bytes.Buffer, headers map[string]string, timeout int, retry int) ([]byte, error) {
	return requestWithTrys("POST", url, params, headers, timeout, retry)
}

func requestWithTrys(method, url string, params *bytes.Buffer, headers map[string]string, timeout int, retry int) ([]byte, error) {
	var result []byte
	var err error

	if retry < 0 {
		retry = MaxRetry
	}

	for i := 0; i < retry+1; i++ {
		result, err = Request(method, url, *params, headers, timeout)
		if err == nil {
			break
		}
		logx.Errorf("Request err : %+v", err)
	}

	if err != nil {
		return nil, err
	}

	return result, nil
}

func Request(method, uri string, params bytes.Buffer, headers map[string]string, timeout int) ([]byte, error) {

	logx.Info(method, " api: ", uri, " params:", params.String())

	var req *http.Request
	var err error

	if method == "GET" {
		req, err = http.NewRequest("GET", uri, nil)
	} else {
		a := strings.NewReader("")
		req, err = http.NewRequest("POST", uri, a)
	}
	if err != nil {
		return nil, err
	}

	if method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 设置header
	for k, v := range headers {
		if k == "Host" {
			req.Host = v
		} else {
			req.Header.Set(k, v)
		}
	}

	// 输出到 log
	//byts, _ := httputil.DumpRequest(req, true)
	//log.Info(string(byts))

	// 走代理
	//proxy := func(_ *http.Request) (*url.URL, error) {
	//	return url.Parse("http://localhost:15236")
	//}

	transport := &http.Transport{
		//Proxy:                  proxy,
		//TLSClientConfig:        &tls.Config{
		//	InsecureSkipVerify:          true,
		//},
	}

	if timeout <= 0 {
		timeout = MaxTimeOut
	}
	c := &http.Client{
		Timeout: time.Millisecond * time.Duration(timeout),
		Transport: transport,
	}

	var res *http.Response
	if res, err = c.Do(req); err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	var body []byte
	if body, err = ioutil.ReadAll(res.Body); err != nil {
		return nil, err
	}

	logx.Info("post api res: ", string(body))
	return body, nil
}
