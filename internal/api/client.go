package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client 租房仿真 API 客户端，房源相关请求自动带 X-User-ID
type Client struct {
	baseURL string
	userID  string
	http    *http.Client
}

// NewClient 创建客户端
func NewClient(baseURL, userID string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		userID:  userID,
		http:    &http.Client{},
	}
}

// needUserID 房源相关路径需要 X-User-ID
func (c *Client) needUserID(path string) bool {
	return strings.HasPrefix(path, "/api/houses")
}

func (c *Client) do(method, path string, query url.Values, body []byte) ([]byte, int, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var reqBody io.Reader
	if len(body) > 0 {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, u, reqBody)
	if err != nil {
		return nil, 0, err
	}
	if c.needUserID(path) && c.userID != "" {
		req.Header.Set("X-User-ID", c.userID)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return data, resp.StatusCode, nil
}

// InitHouses 房源数据重置，新 Session 时调用
func (c *Client) InitHouses() ([]byte, int, error) {
	return c.do("POST", "/api/houses/init", nil, nil)
}

// GetLandmarks 获取地标列表
func (c *Client) GetLandmarks(query url.Values) ([]byte, int, error) {
	return c.do("GET", "/api/landmarks", query, nil)
}

// GetLandmarkByName 按名称精确查询地标
func (c *Client) GetLandmarkByName(name string) ([]byte, int, error) {
	path := "/api/landmarks/name/" + url.PathEscape(name)
	return c.do("GET", path, nil, nil)
}

// SearchLandmarks 关键词模糊搜索地标
func (c *Client) SearchLandmarks(query url.Values) ([]byte, int, error) {
	return c.do("GET", "/api/landmarks/search", query, nil)
}

// GetLandmarkByID 按地标 id 查询
func (c *Client) GetLandmarkByID(id string) ([]byte, int, error) {
	path := "/api/landmarks/" + url.PathEscape(id)
	return c.do("GET", path, nil, nil)
}

// GetLandmarkStats 地标统计
func (c *Client) GetLandmarkStats() ([]byte, int, error) {
	return c.do("GET", "/api/landmarks/stats", nil, nil)
}

// GetHouseByID 根据房源 ID 获取详情
func (c *Client) GetHouseByID(houseID string) ([]byte, int, error) {
	path := "/api/houses/" + url.PathEscape(houseID)
	return c.do("GET", path, nil, nil)
}

// GetHouseListings 根据房源 ID 获取各平台挂牌记录
func (c *Client) GetHouseListings(houseID string) ([]byte, int, error) {
	path := "/api/houses/listings/" + url.PathEscape(houseID)
	return c.do("GET", path, nil, nil)
}

// GetHousesByCommunity 按小区名查询可租房源
func (c *Client) GetHousesByCommunity(query url.Values) ([]byte, int, error) {
	return c.do("GET", "/api/houses/by_community", query, nil)
}

// GetHousesByPlatform 按条件查询可租房源
func (c *Client) GetHousesByPlatform(query url.Values) ([]byte, int, error) {
	return c.do("GET", "/api/houses/by_platform", query, nil)
}

// GetHousesNearby 以地标为圆心查附近房源
func (c *Client) GetHousesNearby(query url.Values) ([]byte, int, error) {
	return c.do("GET", "/api/houses/nearby", query, nil)
}

// GetNearbyLandmarks 查询小区周边地标（商超/公园）
func (c *Client) GetNearbyLandmarks(query url.Values) ([]byte, int, error) {
	return c.do("GET", "/api/houses/nearby_landmarks", query, nil)
}

// GetHouseStats 房源统计
func (c *Client) GetHouseStats() ([]byte, int, error) {
	return c.do("GET", "/api/houses/stats", nil, nil)
}

// RentHouse 租房
func (c *Client) RentHouse(houseID string, listingPlatform string) ([]byte, int, error) {
	path := "/api/houses/" + url.PathEscape(houseID) + "/rent"
	q := url.Values{}
	q.Set("listing_platform", listingPlatform)
	return c.do("POST", path, q, nil)
}

// TerminateRental 退租
func (c *Client) TerminateRental(houseID string, listingPlatform string) ([]byte, int, error) {
	path := "/api/houses/" + url.PathEscape(houseID) + "/terminate"
	q := url.Values{}
	q.Set("listing_platform", listingPlatform)
	return c.do("POST", path, q, nil)
}

// TakeOffline 下架
func (c *Client) TakeOffline(houseID string, listingPlatform string) ([]byte, int, error) {
	path := "/api/houses/" + url.PathEscape(houseID) + "/offline"
	q := url.Values{}
	q.Set("listing_platform", listingPlatform)
	return c.do("POST", path, q, nil)
}

// BuildQuery 从 map 构建 url.Values
func BuildQuery(m map[string]interface{}) url.Values {
	q := url.Values{}
	for k, v := range m {
		if v == nil {
			continue
		}
		switch x := v.(type) {
		case string:
			if x != "" {
				q.Set(k, x)
			}
		case int:
			q.Set(k, fmt.Sprintf("%d", x))
		case float64:
			q.Set(k, fmt.Sprintf("%v", x))
		case bool:
			q.Set(k, fmt.Sprintf("%t", x))
		}
	}
	return q
}

// JSONMap 用于解析工具参数
func JSONMap(data []byte) (map[string]interface{}, error) {
	var m map[string]interface{}
	err := json.Unmarshal(data, &m)
	return m, err
}
