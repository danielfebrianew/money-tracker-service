package auth

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/pkg/httphelper"
)

func bind(c echo.Context, dest interface{}) error  { return httphelper.Bind(c, dest) }
func requireUserID(c echo.Context) (string, error) { return httphelper.RequireUserID(c) }
func respondError(c echo.Context, err error) error { return httphelper.RespondError(c, err) }

// Audience constants for cookie scoping. User and admin sessions live under
// separate cookie names so they can coexist on the same client without one
// overriding the other.
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

type cookieNames struct {
	access      string
	refresh     string
	refreshPath string
}

func cookiesFor(audience string) cookieNames {
	if audience == AudienceAdmin {
		return cookieNames{access: AdminAccessCookie, refresh: AdminRefreshCookie, refreshPath: AdminRefreshPath}
	}
	return cookieNames{access: UserAccessCookie, refresh: UserRefreshCookie, refreshPath: UserRefreshPath}
}

// SetAuthCookies is the exported variant for callers in other modules (e.g. admin).
func SetAuthCookies(c echo.Context, pair TokenPair, audience string) {
	setAuthCookies(c, pair, audience)
}

// ClearAuthCookies is the exported variant for callers in other modules.
func ClearAuthCookies(c echo.Context, audience string) {
	clearAuthCookies(c, audience)
}

// RefreshTokenFromRequest exposes the cookie/body lookup helper to other modules.
func RefreshTokenFromRequest(c echo.Context, cookieName string) string {
	return refreshTokenFromRequest(c, cookieName)
}

func setAuthCookies(c echo.Context, pair TokenPair, audience string) {
	names := cookiesFor(audience)
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

func clearAuthCookies(c echo.Context, audience string) {
	names := cookiesFor(audience)
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

func isSecure(c echo.Context) bool {
	return c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https"
}
