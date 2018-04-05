package cache

import (
	"api-management/authentication"
	"api-management/http"
	"api-management/servicediscovery"
	"encoding/json"
	"fmt"

	routing "github.com/qiangxue/fasthttp-routing"
)

type CachedRequest struct {
	Service               servicediscovery.Service
	UpdateServiceCache    bool
	Protection            authentication.ProtectionInfo
	UpdateProtectionCache bool
	Response              http.ResponseInfo
	UpdateResponseCache   bool
}

func sdCacheKey(c *routing.Context) string {
	return string(c.Request.RequestURI())
}

func oauthCacheKey(c *routing.Context) string {
	return string(c.Request.Header.Peek("Authorization"))
}

func apiResponseCacheKey(c *routing.Context) string {
	var apiKey = sdCacheKey(c) + "?"

	c.QueryArgs().VisitAll(func(key []byte, val []byte) {
		apiKey = apiKey + string(key) + "=" + string(val)
	})

	apiKey = apiKey + string(c.Request.Header.Peek("Authorization"))

	return apiKey
}

func GetCacheForRequest(c *routing.Context) CachedRequest {
	var serviceCache servicediscovery.Service
	var protectionCacheObj authentication.ProtectionInfo
	var respObj http.ResponseInfo

	sdCache, sdCacheErr := ServiceDiscoveryCacheGet(sdCacheKey(c))
	protectionCache, protectionCacheErr := OAuthCacheGet(oauthCacheKey(c))
	apiRespCache, apiRespCacheErr := ApisCacheGet(apiResponseCacheKey(c))

	if sdCacheErr == nil {
		json.Unmarshal(sdCache, &serviceCache)
	}
	if protectionCacheErr == nil {
		json.Unmarshal(protectionCache, &protectionCacheObj)
	}
	if apiRespCacheErr == nil {
		json.Unmarshal(apiRespCache, &respObj)
	}

	return CachedRequest{
		Service: serviceCache, Protection: protectionCacheObj, Response: respObj,
		UpdateProtectionCache: false, UpdateResponseCache: false, UpdateServiceCache: false}
}

func StoreRequestInfoToCache(c *routing.Context, requestInfo CachedRequest) {
	if requestInfo.UpdateServiceCache {
		fmt.Println("SET SD CACHE")
		serviceDiscoveryJson, _ := json.Marshal(requestInfo.Service)
		ServiceDiscoveryCacheStore(sdCacheKey(c), serviceDiscoveryJson)
	}

	if !requestInfo.Service.IsCachingActive {
		return
	}

	if requestInfo.UpdateProtectionCache {
		fmt.Println("SET OAUTH CACHE")
		protectionJson, _ := json.Marshal(requestInfo.Protection)
		OAuthCacheStore(oauthCacheKey(c), protectionJson)
	}

	if requestInfo.UpdateResponseCache && string(c.Method()) == "GET" {
		fmt.Println("SET RESPONSE CACHE")
		apiResponseJson, _ := json.Marshal(requestInfo.Response)
		ApisCacheStore(apiResponseCacheKey(c), apiResponseJson)
	}
}