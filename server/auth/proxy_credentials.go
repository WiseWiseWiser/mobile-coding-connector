package auth

import (
	"net/http"
	"strings"
)

// StripManagementCredentials prevents credentials for this management server
// from reaching a different application through an origin proxy. Application
// cookies and unrelated application bearer tokens must remain intact.
func StripManagementCredentials(r *http.Request) {
	cookies := r.Cookies()
	name := CookieName()
	r.Header.Del("Cookie")
	for _, cookie := range cookies {
		if cookie.Name != name {
			r.AddCookie(cookie)
		}
	}
	fields := strings.Fields(r.Header.Get("Authorization"))
	if len(fields) == 2 && strings.EqualFold(fields[0], "Bearer") {
		if _, valid := loadAndCheckToken(fields[1]); valid {
			r.Header.Del("Authorization")
		}
	}
}
