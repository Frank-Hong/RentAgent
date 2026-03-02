package agent

import (
	"encoding/json"
	"fmt"
	"net/url"

	"rentagent/internal/api"
)

// ToolDef OpenAI 兼容的 function 定义
type ToolDef struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function,omitempty"`
}

// FunctionDef 函数名、描述、参数 JSON schema
type FunctionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ToolDefinitions 返回所有工具定义，供 LLM 使用
func ToolDefinitions() []ToolDef {
	return []ToolDef{
		{Type: "function", Function: FunctionDef{
			Name:        "houses_init",
			Description: "房源数据重置。每个新会话开始时必须调用一次，确保用例使用初始化的数据。",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "get_landmarks",
			Description: "获取地标列表，支持按类别(category)、行政区(district)筛选。类别: subway/company/landmark。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{"type": "string", "description": "地标类别: subway/company/landmark"},
					"district": map[string]interface{}{"type": "string", "description": "行政区，如 海淀、朝阳"},
				},
			},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "get_landmark_by_name",
			Description: "按名称精确查询地标，如西二旗站、百度。返回地标 id 等，用于后续 nearby 查房。",
			Parameters: map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string", "description": "地标名称，如 西二旗站、国贸"}},
				"required": []string{"name"},
			},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "search_landmarks",
			Description: "关键词模糊搜索地标，q 必填。可加 category、district 限定。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"q": "string", "category": "string", "district": "string",
				},
				"required": []string{"q"},
			},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "get_landmark_by_id",
			Description: "按地标 id 查询地标详情。",
			Parameters: map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{"id": map[string]interface{}{"type": "string"}},
				"required": []string{"id"},
			},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "get_landmark_stats",
			Description: "获取地标统计信息（总数、按类别分布等）。",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "get_house_by_id",
			Description: "根据房源 ID 获取单套房源详情。",
			Parameters: map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{"house_id": map[string]interface{}{"type": "string"}},
				"required": []string{"house_id"},
			},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "get_house_listings",
			Description: "根据房源 ID 获取该房源在链家/安居客/58同城等各平台的全部挂牌记录。",
			Parameters: map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{"house_id": map[string]interface{}{"type": "string"}},
				"required": []string{"house_id"},
			},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "get_houses_by_community",
			Description: "按小区名查询该小区下可租房源。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"community": "string", "listing_platform": "string", "page": "integer", "page_size": "integer",
				},
				"required": []string{"community"},
			},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "get_houses_by_platform",
			Description: "查询可租房源，支持行政区、商圈、价格、户型、整租/合租、地铁距离、通勤等筛选。近地铁用 max_subway_dist=800。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"listing_platform": "string", "district": "string", "area": "string",
					"min_price": "integer", "max_price": "integer", "bedrooms": "string", "rental_type": "string",
					"decoration": "string", "orientation": "string", "elevator": "string",
					"min_area": "integer", "max_area": "integer", "subway_line": "string", "max_subway_dist": "integer",
					"subway_station": "string", "available_from_before": "string", "commute_to_xierqi_max": "integer",
					"sort_by": "string", "sort_order": "string", "page": "integer", "page_size": "integer",
				},
			},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "get_houses_nearby",
			Description: "以地标为圆心查附近可租房源。需先用地标接口获得 landmark_id。max_distance 默认 2000 米。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"landmark_id": "string", "max_distance": "number", "listing_platform": "string", "page": "integer", "page_size": "integer",
				},
				"required": []string{"landmark_id"},
			},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "get_nearby_landmarks",
			Description: "查询某小区周边地标（商超/公园）。type: shopping 或 park。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{"community": "string", "type": "string", "max_distance_m": "number"},
				"required": []string{"community"},
			},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "get_house_stats",
			Description: "获取房源统计信息。",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "rent_house",
			Description: "租房：将房源设为已租。必须调用此接口才算完成。listing_platform 必填：链家/安居客/58同城。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"house_id": "string", "listing_platform": map[string]interface{}{"type": "string", "enum": []string{"链家", "安居客", "58同城"}},
				},
				"required": []string{"house_id", "listing_platform"},
			},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "terminate_rental",
			Description: "退租。listing_platform 必填。",
			Parameters: map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{"house_id": "string", "listing_platform": "string"},
				"required": []string{"house_id", "listing_platform"},
			},
		}},
		{Type: "function", Function: FunctionDef{
			Name:        "take_offline",
			Description: "下架。listing_platform 必填。",
			Parameters: map[string]interface{}{
				"type": "object", "properties": map[string]interface{}{"house_id": "string", "listing_platform": "string"},
				"required": []string{"house_id", "listing_platform"},
			},
		}},
	}
}

// ExecuteTool 执行指定工具，返回结果字符串与错误
func ExecuteTool(client *api.Client, name string, argsJSON string) (string, error) {
	var args map[string]interface{}
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("解析参数失败: %w", err)
		}
	}
	if args == nil {
		args = make(map[string]interface{})
	}
	str := func(k string) string {
		val, ok := args[k]
		if !ok || val == nil {
			return ""
		}
		if s, ok := val.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", val)
	}
	intVal := func(k string) int {
		val, ok := args[k]
		if !ok || val == nil {
			return 0
		}
		switch x := val.(type) {
		case float64:
			return int(x)
		case int:
			return x
		}
		return 0
	}

	switch name {
	case "houses_init":
		data, code, err := client.InitHouses()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("status=%d body=%s", code, string(data)), nil
	case "get_landmarks":
		q := url.Values{}
		if v := str("category"); v != "" {
			q.Set("category", v)
		}
		if v := str("district"); v != "" {
			q.Set("district", v)
		}
		data, _, err := client.GetLandmarks(q)
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "get_landmark_by_name":
		data, _, err := client.GetLandmarkByName(str("name"))
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "search_landmarks":
		q := url.Values{}
		q.Set("q", str("q"))
		if v := str("category"); v != "" {
			q.Set("category", v)
		}
		if v := str("district"); v != "" {
			q.Set("district", v)
		}
		data, _, err := client.SearchLandmarks(q)
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "get_landmark_by_id":
		data, _, err := client.GetLandmarkByID(str("id"))
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "get_landmark_stats":
		data, _, err := client.GetLandmarkStats()
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "get_house_by_id":
		data, _, err := client.GetHouseByID(str("house_id"))
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "get_house_listings":
		data, _, err := client.GetHouseListings(str("house_id"))
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "get_houses_by_community":
		q := url.Values{}
		q.Set("community", str("community"))
		if v := str("listing_platform"); v != "" {
			q.Set("listing_platform", v)
		}
		if p := intVal("page"); p > 0 {
			q.Set("page", fmt.Sprintf("%d", p))
		}
		if ps := intVal("page_size"); ps > 0 {
			q.Set("page_size", fmt.Sprintf("%d", ps))
		}
		data, _, err := client.GetHousesByCommunity(q)
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "get_houses_by_platform":
		q := buildPlatformQuery(args)
		data, _, err := client.GetHousesByPlatform(q)
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "get_houses_nearby":
		q := url.Values{}
		q.Set("landmark_id", str("landmark_id"))
		if v := str("max_distance"); v != "" {
			q.Set("max_distance", v)
		}
		if v := str("listing_platform"); v != "" {
			q.Set("listing_platform", v)
		}
		if p := intVal("page"); p > 0 {
			q.Set("page", fmt.Sprintf("%d", p))
		}
		if ps := intVal("page_size"); ps > 0 {
			q.Set("page_size", fmt.Sprintf("%d", ps))
		}
		data, _, err := client.GetHousesNearby(q)
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "get_nearby_landmarks":
		q := url.Values{}
		q.Set("community", str("community"))
		if v := str("type"); v != "" {
			q.Set("type", v)
		}
		if v := str("max_distance_m"); v != "" {
			q.Set("max_distance_m", v)
		}
		data, _, err := client.GetNearbyLandmarks(q)
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "get_house_stats":
		data, _, err := client.GetHouseStats()
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "rent_house":
		data, code, err := client.RentHouse(str("house_id"), str("listing_platform"))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("status=%d %s", code, string(data)), nil
	case "terminate_rental":
		data, code, err := client.TerminateRental(str("house_id"), str("listing_platform"))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("status=%d %s", code, string(data)), nil
	case "take_offline":
		data, code, err := client.TakeOffline(str("house_id"), str("listing_platform"))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("status=%d %s", code, string(data)), nil
	default:
		return "", fmt.Errorf("未知工具: %s", name)
	}
}

func buildPlatformQuery(args map[string]interface{}) url.Values {
	q := make(url.Values)
	for k, v := range args {
		if v == nil {
			continue
		}
		switch k {
		case "listing_platform", "district", "area", "bedrooms", "rental_type", "decoration", "orientation", "subway_line", "subway_station", "available_from_before", "sort_by", "sort_order":
			if s, ok := v.(string); ok && s != "" {
				q.Set(k, s)
			}
		case "elevator":
			if b, ok := v.(bool); ok {
				q.Set(k, fmt.Sprintf("%t", b))
			} else if s, ok := v.(string); ok && s != "" {
				q.Set(k, s)
			}
		case "min_price", "max_price", "min_area", "max_area", "max_subway_dist", "commute_to_xierqi_max", "page", "page_size":
			switch x := v.(type) {
			case float64:
				q.Set(k, fmt.Sprintf("%.0f", x))
			case int:
				q.Set(k, fmt.Sprintf("%d", x))
			case string:
				if x != "" {
					q.Set(k, x)
				}
			}
		}
	}
	return q
}
