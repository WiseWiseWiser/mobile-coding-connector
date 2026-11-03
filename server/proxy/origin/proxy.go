package origin

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/xhd2015/ai-critic/server/auth"
)

// NewReverseProxy is the shared streaming forwarding core for persistent
// mappings, optional authentication gates, and temporary visits. It intentionally
// retains the public Host and Origin; the application's trust policy is separate.
func NewReverseProxy(target *url.URL, transport http.RoundTripper, stripCookies ...string) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	director := proxy.Director
	proxy.Director = func(r *http.Request) {
		auth.StripManagementCredentials(r)
		cookies := r.Cookies()
		r.Header.Del("Cookie")
		for _, cookie := range cookies {
			blocked := cookie.Name == auth.ServiceCookieName
			for _, name := range stripCookies {
				if name != "" && cookie.Name == name {
					blocked = true
				}
			}
			if !blocked {
				r.AddCookie(cookie)
			}
		}
		director(r)
	}
	proxy.Transport = transport
	proxy.FlushInterval = -1
	return proxy
}
