package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zalando/skipper/filters"
	"golang.org/x/oauth2"
)

const (
	// Deprecated, use filters.OAuthGrantName instead
	OAuthGrantName = filters.OAuthGrantName

	secretsRefreshInternal = time.Minute
	tokenWasRefreshed      = "oauth-did-refresh"
)

var (
	errExpiredToken = errors.New("expired access token")
)

type grantSpec struct {
	config OAuthConfig
}

type grantFilter struct {
	config OAuthConfig
}

func (s *grantSpec) Name() string { return filters.OAuthGrantName }

func (s *grantSpec) CreateFilter([]interface{}) (filters.Filter, error) {
	return &grantFilter{
		config: s.config,
	}, nil
}

func providerContext(c OAuthConfig) context.Context {
	return context.WithValue(context.Background(), oauth2.HTTPClient, c.AuthClient)
}

func serverError(ctx filters.FilterContext) {
	ctx.Serve(&http.Response{
		StatusCode: http.StatusInternalServerError,
	})
}

func badRequest(ctx filters.FilterContext) {
	ctx.Serve(&http.Response{
		StatusCode: http.StatusBadRequest,
	})
}

func loginRedirect(ctx filters.FilterContext, config OAuthConfig) {
	loginRedirectWithOverride(ctx, config, "")
}

func loginRedirectWithOverride(ctx filters.FilterContext, config OAuthConfig, originalOverride string) {
	req := ctx.Request()
	redirect, original := config.RedirectURLs(req)

	if originalOverride != "" {
		original = originalOverride
	}

	state, err := config.flowState.createState(original)
	if err != nil {
		log.Errorf("Failed to create login redirect: %v", err)
		serverError(ctx)
		return
	}

	authConfig := config.GetConfig()
	ctx.Serve(&http.Response{
		StatusCode: http.StatusTemporaryRedirect,
		Header: http.Header{
			"Location": []string{authConfig.AuthCodeURL(state, config.GetAuthURLParameters(redirect)...)},
		},
	})
}

func (f *grantFilter) refreshToken(c cookie) (*oauth2.Token, error) {
	// Set the expiry of the token to the past to trigger oauth2.TokenSource
	// to refresh the access token.
	token := &oauth2.Token{
		AccessToken:  c.AccessToken,
		RefreshToken: c.RefreshToken,
		Expiry:       time.Now().Add(-time.Minute),
	}

	ctx := providerContext(f.config)

	// oauth2.TokenSource implements the refresh functionality,
	// we're hijacking it here.
	tokenSource := f.config.GetConfig().TokenSource(ctx, token)
	return tokenSource.Token()
}

func (f *grantFilter) refreshTokenIfRequired(c cookie, ctx filters.FilterContext) (*oauth2.Token, error) {
	canRefresh := c.RefreshToken != ""

	if c.isAccessTokenExpired() {
		if canRefresh {
			token, err := f.refreshToken(c)
			if err == nil {
				// Remember that this token was just successfully refreshed
				// so that we can send  an updated cookie in the response.
				ctx.StateBag()[tokenWasRefreshed] = true
			}
			return token, err
		} else {
			return nil, errExpiredToken
		}
	} else {
		return &oauth2.Token{
			AccessToken:  c.AccessToken,
			TokenType:    "Bearer",
			RefreshToken: c.RefreshToken,
			Expiry:       c.Expiry,
		}, nil
	}
}

func (f *grantFilter) setAccessTokenHeader(req *http.Request, token string) {
	if f.config.AccessTokenHeaderName != "" {
		req.Header.Set(f.config.AccessTokenHeaderName, authHeaderPrefix+token)
	}
}

func (f *grantFilter) createTokenContainer(token *oauth2.Token, tokeninfo map[string]interface{}) (tokenContainer, error) {
	subject := ""
	if f.config.TokeninfoSubjectKey != "" {
		if s, ok := tokeninfo[f.config.TokeninfoSubjectKey].(string); ok {
			subject = s
		} else {
			return tokenContainer{}, fmt.Errorf("tokeninfo subject key '%s' is missing", f.config.TokeninfoSubjectKey)
		}
	}

	tokeninfo["sub"] = subject

	return tokenContainer{
		OAuth2Token: token,
		Subject:     subject,
		Claims:      tokeninfo,
	}, nil
}

func (f *grantFilter) Request(ctx filters.FilterContext) {
	req := ctx.Request()

	c, err := extractCookie(req, f.config)
	if err == http.ErrNoCookie {
		loginRedirect(ctx, f.config)
		return
	}

	token, err := f.refreshTokenIfRequired(*c, ctx)
	if err != nil && c.isAccessTokenExpired() {
		// Refresh failed and we no longer have a valid access token.
		loginRedirect(ctx, f.config)
		return
	}

	tokeninfo, err := f.config.TokeninfoClient.getTokeninfo(token.AccessToken, ctx)
	if err != nil {
		if err != errInvalidToken {
			log.Errorf("Failed to call tokeninfo: %v.", err)
		}
		loginRedirect(ctx, f.config)
		return
	}

	f.setAccessTokenHeader(req, token.AccessToken)

	tokenContainer, err := f.createTokenContainer(token, tokeninfo)
	if err != nil {
		log.Errorf("Failed to create token container: %v.", err)
		loginRedirect(ctx, f.config)
		return
	}

	// Set token in state bag for response Set-Cookie. By piggy-backing
	// on the OIDC token container, we gain downstream compatibility with
	// the oidcClaimsQuery filter.
	ctx.StateBag()[oidcClaimsCacheKey] = tokenContainer

	// Set the tokeninfo also in the tokeninfoCacheKey state bag, so we
	// can reuse e.g. the forwardToken() filter.
	ctx.StateBag()[tokeninfoCacheKey] = tokeninfo
}

func (f *grantFilter) Response(ctx filters.FilterContext) {
	// If the token was refreshed in this request flow,
	// we want to send an updated cookie. If it wasn't, the
	// users will still have their old cookie and we do not
	// need to send it again and this function can exit early.
	didRefresh, ok := ctx.StateBag()[tokenWasRefreshed].(bool)
	if !didRefresh || !ok {
		return
	}

	container, ok := ctx.StateBag()[oidcClaimsCacheKey].(tokenContainer)
	if !ok {
		return
	}

	req := ctx.Request()
	c, err := createCookie(f.config, req.Host, container.OAuth2Token)
	if err != nil {
		log.Errorf("Failed to generate cookie: %v.", err)
		return
	}

	rsp := ctx.Response()
	rsp.Header.Add("Set-Cookie", c.String())
}
