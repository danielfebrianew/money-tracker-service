package cookie

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

const (
	AudienceUser  = "user"
	AudienceAdmin = "admin"

	UserAccessCookie   = "user_access_token"
	UserRefreshCookie  = "user_refresh_token"
	AdminAccessCookie  = "admin_access_token"
	AdminRefreshCookie = "admin_refresh_token"

	UserRefreshPath  = "/api/auth/refresh"
	AdminRefreshPath = "/api/admin/auth/refresh"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type cookieNames struct {
	access      string
	refresh     string
	refreshPath string
}

func namesFor(audience string) cookieNames {
	if audience == AudienceAdmin {
		return cookieNames{access: AdminAccessCookie, refresh: AdminRefreshCookie, refreshPath: AdminRefreshPath}
	}
	return cookieNames{access: UserAccessCookie, refresh: UserRefreshCookie, refreshPath: UserRefreshPath}
}

func SetAuthCookies(c echo.Context, pair TokenPair, audience string) {
	names := namesFor(audience)
	secure := isSecure(c)
	c.SetCookie(&http.Cookie{
		Name:     names.access,
		Value:    pair.AccessToken,
		Path:     "/",
		MaxAge:   int(pair.ExpiresIn),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	c.SetCookie(&http.Cookie{
		Name:     names.refresh,
		Value:    pair.RefreshToken,
		Path:     names.refreshPath,
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearAuthCookies(c echo.Context, audience string) {
	names := namesFor(audience)
	secure := isSecure(c)
	c.SetCookie(&http.Cookie{
		Name:     names.access,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	c.SetCookie(&http.Cookie{
		Name:     names.refresh,
		Value:    "",
		Path:     names.refreshPath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func RefreshTokenFromRequest(c echo.Context, cookieName string) string {
	if ck, err := c.Cookie(cookieName); err == nil && ck.Value != "" {
		return ck.Value
	}
	return ""
}

func isSecure(c echo.Context) bool {
	return c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https"
}
