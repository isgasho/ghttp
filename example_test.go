package ghttp_test

import (
	"context"
	"errors"
	"fmt"
	neturl "net/url"
	"regexp"
	"sync"
	"time"

	"github.com/winterssy/ghttp"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/time/rate"
)

type (
	response = ghttp.PostmanResponse
)

func Example_goStyleAPI() {
	client := ghttp.New()

	req, err := ghttp.NewRequest(ghttp.MethodPost, "https://httpbin.org/post")
	if err != nil {
		panic(err)
	}

	req.
		SetQuery(ghttp.Params{
			"k1": "v1",
			"k2": "v2",
		}).
		SetHeaders(ghttp.Headers{
			"k3": "v3",
			"k4": "v4",
		}).
		SetForm(ghttp.Form{
			"k5": "v5",
			"k6": "v6",
		})
	client.Do(req)
}

func Example_requestsStyleAPI() {
	client := ghttp.New()

	client.Post("https://httpbin.org/post",
		ghttp.WithQuery(ghttp.Params{
			"k1": "v1",
			"k2": "v2",
		}),
		ghttp.WithHeaders(ghttp.Headers{
			"k3": "v3",
			"k4": "v4",
		}),
		ghttp.WithForm(ghttp.Form{
			"k5": "v5",
			"k6": "v6",
		}),
	)
}

func ExampleClient_SetProxyFromURL() {
	client := ghttp.New().SetProxyFromURL("socks5://127.0.0.1:1080")
	client.Get("https://www.google.com")
}

func ExampleClient_SetCookies() {
	cookies := ghttp.Cookies{
		"n1": "v1",
		"n2": "v2",
	}

	client := ghttp.
		New().
		EnableSession().
		SetCookies("https://httpbin.org", cookies.Decode()...)

	resp := new(response)
	err := client.
		Get("https://httpbin.org/cookies").
		JSON(resp)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Cookies.GetString("n1"))
	fmt.Println(resp.Cookies.GetString("n2"))
	// Output:
	// v1
	// v2
}

func ExampleClient_FilterCookie() {
	client := ghttp.
		New().
		EnableSession()

	client.Get("https://httpbin.org/cookies/set/uid/10086")
	c, err := client.FilterCookie("https://httpbin.org", "uid")
	if err != nil {
		panic(err)
	}

	fmt.Println(c.Name)
	fmt.Println(c.Value)
	// Output:
	// uid
	// 10086
}

func ExampleClient_DisableVerify() {
	client := ghttp.New().DisableVerify()

	client.Get("https://self-signed.badssl.com")
}

func ExampleClient_UseRateLimiter() {
	pattern := regexp.MustCompile("post|delete")
	limiter := ghttp.NewRegexpLimiter(rate.NewLimiter(1, 10), pattern)

	client := ghttp.New().UseRateLimiter(limiter)
	wg := new(sync.WaitGroup)
	for i := 0; i < 100; i += 2 {
		wg.Add(2)
		go func() {
			client.Get("https://api.example.com/get")
			wg.Done()
		}()
		go func() {
			client.Post("https://api.example.com/post")
			wg.Done()
		}()
	}
	wg.Wait()
}

func ExampleClient_OnBeforeRequest() {
	withReverseProxy := func(target string) ghttp.BeforeRequestHook {
		return func(req *ghttp.Request) error {
			u, err := neturl.Parse(target)
			if err != nil {
				return err
			}

			req.URL.Scheme = u.Scheme
			req.URL.Host = u.Host
			req.Host = u.Host
			req.SetOrigin(u.Host)
			return nil
		}
	}

	client := ghttp.New().OnBeforeRequest(withReverseProxy("https://httpbin.org"))
	client.Get("/get")
	client.Post("/post")
	client.Put("/put")
	client.Patch("/patch")
	client.Delete("/delete")
}

func ExampleClient_OnAfterResponse() {
	client := ghttp.New().OnAfterResponse(func(resp *ghttp.Response) error {
		data, err := resp.
			Prefetch().
			H()
		if err != nil {
			return err
		}

		if data.GetNumber("code").Int() != 200 {
			return errors.New(data.GetStringDefault("msg", "fail"))
		}

		return nil
	})

	client.Post("https://api.example.com/login",
		ghttp.WithBasicAuth("user", "p@ssw$"),
	)
}

func ExampleWithQuery() {
	client := ghttp.New()

	resp := new(response)
	err := client.
		Post("https://httpbin.org/post",
			ghttp.WithQuery(ghttp.Params{
				"k1": "v1",
				"k2": "v2",
			}),
		).
		JSON(resp)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Args.GetString("k1"))
	fmt.Println(resp.Args.GetString("k2"))
	// Output:
	// v1
	// v2
}

func ExampleWithHeaders() {
	client := ghttp.New()

	resp := new(response)
	err := client.
		Post("https://httpbin.org/post",
			ghttp.WithHeaders(ghttp.Headers{
				"k1": "v1",
				"k2": "v2",
			}),
		).
		JSON(resp)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Headers.GetString("K1"))
	fmt.Println(resp.Headers.GetString("K2"))
	// Output:
	// v1
	// v2
}

func ExampleWithForm() {
	client := ghttp.New()

	resp := new(response)
	err := client.
		Post("https://httpbin.org/post",
			ghttp.WithForm(ghttp.Form{
				"k1": "v1",
				"k2": "v2",
			}),
		).
		JSON(resp)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Form.GetString("k1"))
	fmt.Println(resp.Form.GetString("k2"))
	// Output:
	// v1
	// v2
}

func ExampleWithJSON() {
	client := ghttp.New()

	resp := new(response)
	err := client.
		Post("https://httpbin.org/post",
			ghttp.WithJSON(map[string]interface{}{
				"msg": "hello world",
				"num": 2019,
			}, true),
		).
		JSON(resp)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.JSON.GetString("msg"))
	fmt.Println(resp.JSON.GetNumber("num").Int())
	// Output:
	// hello world
	// 2019
}

func ExampleWithMultipart() {
	client := ghttp.New()

	files := ghttp.Files{
		"file1": ghttp.MustOpen("./testdata/testfile1.txt"),
		"file2": ghttp.MustOpen("./testdata/testfile2.txt"),
	}
	form := ghttp.Form{
		"k1": "v1",
		"k2": "v2",
	}

	resp := new(response)
	err := client.
		Post("https://httpbin.org/post",
			ghttp.WithMultipart(files, form),
		).
		JSON(resp)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Files.GetString("file1"))
	fmt.Println(resp.Files.GetString("file2"))
	fmt.Println(resp.Form.GetString("k1"))
	fmt.Println(resp.Form.GetString("k2"))
	// Output:
	// testfile1.txt
	// testfile2.txt
	// v1
	// v2
}

func ExampleWithCookies() {
	client := ghttp.New()

	resp := new(response)
	err := client.
		Get("https://httpbin.org/cookies",
			ghttp.WithCookies(ghttp.Cookies{
				"n1": "v1",
				"n2": "v2",
			}),
		).
		JSON(resp)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Cookies.GetString("n1"))
	fmt.Println(resp.Cookies.GetString("n2"))
	// Output:
	// v1
	// v2
}

func ExampleWithContext() {
	client := ghttp.New()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client.Get("https://httpbin.org/delay/10",
		ghttp.WithContext(ctx),
	)
}

func ExampleWithRetry() {
	client := ghttp.New()

	backoff := ghttp.NewExponentialBackoff(1*time.Second, 30*time.Second, true)
	retrier := ghttp.NewRetrier(5, backoff)
	client.Post("https://api.example.com/login",
		ghttp.WithBasicAuth("user", "p@ssw$"),
		ghttp.WithRetry(retrier),
	)
}

func ExampleRequest_Export() {
	req, err := ghttp.NewRequest(ghttp.MethodPost, "https://httpbin.org/post",
		ghttp.WithQuery(ghttp.Params{
			"k1": "v1",
			"k2": "v2",
		}),
		ghttp.WithHeaders(ghttp.Headers{
			"k3": "v3",
			"k4": "v4",
		}),
		ghttp.WithForm(ghttp.Form{
			"k5": "v5",
			"k6": "v6",
		}),
	)
	if err != nil {
		panic(err)
	}

	cmd, _ := req.Export()
	fmt.Println(cmd)
	// Output:
	// curl -v -X 'POST' -d 'k5=v5&k6=v6' -H 'Content-Type: application/x-www-form-urlencoded' -H 'K3: v3' -H 'K4: v4' 'https://httpbin.org/post?k1=v1&k2=v2'
}

func ExampleResponse_EnsureStatusOk() {
	client := ghttp.New()

	_, err := client.
		Get("https://www.google.com/404").
		EnsureStatusOk().
		Text()

	fmt.Println(err)
	// Output:
	// ghttp: bad status (404 Not Found)
}

func ExampleResponse_Text() {
	client := ghttp.New()

	_, _ = client.
		Get("https://www.example.com").
		Text()

	_, _ = client.
		Get("https://www.example.cn").
		Text(simplifiedchinese.GBK)
}

func ExampleResponse_H() {
	client := ghttp.New()

	data, err := client.
		Post("https://httpbin.org/post",
			ghttp.WithQuery(ghttp.Params{
				"k1": "v1",
				"k2": "v2",
			}),
			ghttp.WithHeaders(ghttp.Headers{
				"k3": "v3",
				"k4": "v4",
			}),
			ghttp.WithForm(ghttp.Form{
				"k5": "v5",
				"k6": "v6",
			}),
		).
		H()
	if err != nil {
		panic(err)
	}

	fmt.Println(data.GetH("args").GetString("k1"))
	fmt.Println(data.GetH("args").GetString("k2"))
	fmt.Println(data.GetH("headers").GetString("K3"))
	fmt.Println(data.GetH("headers").GetString("K4"))
	fmt.Println(data.GetH("form").GetString("k5"))
	fmt.Println(data.GetH("form").GetString("k6"))
	// Output:
	// v1
	// v2
	// v3
	// v4
	// v5
	// v6
}
