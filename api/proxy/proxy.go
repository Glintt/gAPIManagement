package proxy

import (
	"github.com/Glintt/gAPI/api/cache"
	"github.com/Glintt/gAPI/api/config"
	"github.com/Glintt/gAPI/api/http"
	"github.com/Glintt/gAPI/api/plugins"
	"github.com/Glintt/gAPI/api/ratelimiting"
	"github.com/Glintt/gAPI/api/servicediscovery"
	"github.com/Glintt/gAPI/api/users"
	"github.com/Glintt/gAPI/api/servicediscovery/service"
	thirdpartyauthentication "github.com/Glintt/gAPI/api/thirdpartyauthentication"
	"github.com/Glintt/gAPI/api/utils"
	"github.com/Glintt/gAPI/api/authentication"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	routing "github.com/qiangxue/fasthttp-routing"
)

var oauthserver thirdpartyauthentication.OAuthServer

var SERVICE_NAME = "/proxy"

func StartProxy(router *routing.Router) {
	oauthserver = config.GApiConfiguration.ThirdPartyOAuth

	ratelimiting.InitRateLimiting()
	router.To("GET,POST,PUT,PATCH,DELETE", "/*", ratelimiting.RateLimiting, HandleRequest)

	
}

func HandleRequest(c *routing.Context) error {
	user := authentication.GetAuthenticatedUser(c)
	if user.Username == "" {
		user = users.GetInternalAPIUser()
	}

	sd := *servicediscovery.GetServiceDiscoveryObject(user)

	utils.LogMessage("=========================================", utils.InfoLogType)
	utils.LogMessage("REQUEST =====> Method = "+string(c.Method())+"; URI = "+string(c.Request.RequestURI()), utils.InfoLogType)
	utils.LogMessage("=========================================", utils.InfoLogType)

	cachedRequest := cache.GetCacheForRequest(c)

	// Service discovery
	if cachedRequest.Service.ToURI == "" {
		utils.LogMessage("SD NOT FROM CACHE", utils.DebugLogType)

		var err error
		cachedRequest.Service, err = getServiceFromServiceDiscovery(c, sd)

		utils.LogMessage("IsExternalRequest = "+strconv.FormatBool(sd.IsExternalRequest(c)), utils.DebugLogType)
		utils.LogMessage("IsReachableFromExternal = "+strconv.FormatBool(servicediscovery.IsServiceReachableFromExternal(cachedRequest.Service, sd)), utils.DebugLogType)

		if err != nil || (sd.IsExternalRequest(c) && !servicediscovery.IsServiceReachableFromExternal(cachedRequest.Service, sd)) {
			http.Response(c, `{"error": true, "msg": "Resource not found"}`, 404, SERVICE_NAME, config.APPLICATION_JSON)
			return nil
		}

		cachedRequest.UpdateServiceCache = true
	} else {
		utils.LogMessage("SD FROM CACHE", utils.DebugLogType)
	}

	// OAuth authentication
	if !cachedRequest.Protection.Cached {
		utils.LogMessage("PROTECTION NOT FROM CACHE", utils.DebugLogType)
		cachedRequest.Protection = checkAuthorization(c, cachedRequest.Service)

		if cachedRequest.Protection.Error != nil {
			http.Response(c, `{"error":true, "msg":"Not Authorized."}`, 401, cachedRequest.Service.MatchingURI, config.APPLICATION_JSON)
			return nil
		}

		cachedRequest.UpdateProtectionCache = true
	} else {
		utils.LogMessage("PROTECTION FROM CACHE", utils.DebugLogType)
	}

	// Call plugins before making the call to the microservice
	if runtime.GOOS != "windows" {
		utils.LogMessage("CALL BEFORE REQUEST PLUGINS", utils.DebugLogType)
		pluginErr := plugins.CallBeforeRequestPlugins(c)
		if pluginErr != nil {
			return nil
		}
	}

	// Make request to microservice
	if cachedRequest.Response.StatusCode == 0 {
		utils.LogMessage("RESPONSE NOT FROM CACHE", utils.DebugLogType)
		cachedRequest.Response = getApiResponse(c, cachedRequest.Protection, cachedRequest.Service)
		if cachedRequest.Response.StatusCode < 300 {
			cachedRequest.UpdateResponseCache = true
		}
	} else {
		utils.LogMessage("RESPONSE FROM CACHE", utils.DebugLogType)
	}

	http.Response(c, string(cachedRequest.Response.Body), cachedRequest.Response.StatusCode, cachedRequest.Service.MatchingURI, string(cachedRequest.Response.ContentType))

	if cachedRequest.Response.StatusCode < 300 {
		cache.StoreRequestInfoToCache(c, cachedRequest)
	}
	return nil
}

func getApiResponse(c *routing.Context, authorization thirdpartyauthentication.ProtectionInfo, s service.Service) http.ResponseInfo {

	c.Request.Header.Set(authorization.Header, authorization.UserInfo)
	headers := http.GetHeadersFromRequest(c.Request)
	if _, ok := headers["X-Forwarded-For"]; !ok {
		headers["X-Forwarded-For"] = c.RemoteIP().String()
	}

	body := c.Request.Body()

	response := s.Call(string(c.Method()), http.GetURIWithParams(c), headers, string(body))

	return http.ResponseInfo{
		StatusCode:  response.Header.StatusCode(),
		ContentType: response.Header.ContentType(),
		Body:        response.Body()}
}

func checkAuthorization(c *routing.Context, s service.Service) thirdpartyauthentication.ProtectionInfo {
	if !s.Protected {
		return thirdpartyauthentication.ProtectionInfo{Header: "n/a", UserInfo: "n/a", Error: nil}
	}

	endpoint := strings.Replace(string(c.RequestURI()), s.MatchingURI, "", 1)
	endpoint = strings.Split(endpoint, "?")[0]

	if _, ok := s.ProtectedExclude[endpoint]; !ok {
		for key, val := range s.ProtectedExclude {
			re := regexp.MustCompile(key)

			value := strings.ToLower(val)
			method := strings.ToLower(string(c.Method()))

			if re.ReplaceAllString(endpoint, "") == "" && strings.Contains(value, method) {
				return thirdpartyauthentication.ProtectionInfo{Header: "n/a", UserInfo: "n/a", Error: nil}
			}
		}
	} else {
		return thirdpartyauthentication.ProtectionInfo{Header: "n/a", UserInfo: "n/a", Error: nil}
	}

	headerName, userInfo, notAuthorizedErr := Protect(s, c)

	if notAuthorizedErr != nil {
		return thirdpartyauthentication.ProtectionInfo{Header: "n/a", UserInfo: "n/a", Error: notAuthorizedErr}
	}

	return thirdpartyauthentication.ProtectionInfo{Header: headerName, UserInfo: userInfo, Error: nil}

}

func getServiceFromServiceDiscovery(c *routing.Context, sd servicediscovery.ServiceDiscovery) (service.Service, error) {
	return sd.GetEndpointForUri(string(c.Request.RequestURI()))
}

func Protect(s service.Service, c *routing.Context) (string, string, error) {
	if s.Protected {
		return oauthserver.Authorize(c.Request)
	}
	return "n/a", "n/a", nil
}
