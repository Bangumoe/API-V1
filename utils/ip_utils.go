package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// IPInfoResponse 定义了API返回的JSON结构
type IPInfoResponse struct {
	Ret  int `json:"ret"`
	Data struct {
		IP          string `json:"ip"`
		Country     string `json:"country"`
		CountryCode string `json:"country_code"`
		Prov        string `json:"prov"`
		City        string `json:"city"`
	} `json:"data"`
}

// IsChineseIP 使用外部API检查IP地址是否来自中国
func IsChineseIP(ipAddr string) bool {
	fmt.Printf("IsChineseIP called with IP: %s\n", ipAddr)

	// 针对本地开发环境，将本地回环地址视为中国IP
	if ipAddr == "::1" || ipAddr == "127.0.0.1" {
		fmt.Printf("IP %s is a loopback address, treating as Chinese IP for local development.\n", ipAddr)
		return true
	}

	// 如果是本地回环地址或私有IP，直接返回false或根据实际需求处理
	// net.ParseIP(ipAddr) 可以用来检查是否是有效的IP格式，以及是否是私有/回环地址
	// 这里为了简化，直接调用API，API本身可能会处理无效IP的情况

	apiURL := fmt.Sprintf("https://ip9.com.cn/get?ip=%s", ipAddr)

	client := http.Client{
		Timeout: 5 * time.Second, // 设置超时时间
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		fmt.Printf("Error making GET request to IP API for IP %s: %v\n", ipAddr, err)
		return false // 网络错误，保守返回false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("IP API returned non-OK status for IP %s: %d\n", ipAddr, resp.StatusCode)
		bodyBytes, _ := ioutil.ReadAll(resp.Body) // Try to read body for logging
		fmt.Printf("IP API response body for IP %s: %s\n", ipAddr, string(bodyBytes))
		return false // API返回非200状态，保守返回false
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading IP API response body for IP %s: %v\n", ipAddr, err)
		return false // 读取响应体错误，保守返回false
	}
	fmt.Printf("IP API response body for IP %s: %s\n", ipAddr, string(body))

	var ipInfo IPInfoResponse
	err = json.Unmarshal(body, &ipInfo)
	if err != nil {
		fmt.Printf("Error unmarshalling IP API response for IP %s: %v\n", ipAddr, err)
		return false // JSON解析错误，保守返回false
	}

	fmt.Printf("IP API parsed response for IP %s: Ret=%d, CountryCode='%s'\n", ipAddr, ipInfo.Ret, ipInfo.Data.CountryCode)
	result := ipInfo.Ret == 200 && ipInfo.Data.CountryCode == "cn"
	fmt.Printf("IsChineseIP for IP %s returning: %t\n", ipAddr, result)
	return result
}

// GetPrefixedURL prefixes the given relative link with the appropriate domain based on the client's IP.
func GetPrefixedURL(clientIP string, relativeLink *string) *string {
	if relativeLink == nil || *relativeLink == "" {
		return relativeLink
	}

	linkStr := *relativeLink

	// If it's already an absolute URL, return it as is.
	if strings.HasPrefix(linkStr, "http://") || strings.HasPrefix(linkStr, "https://") {
		return relativeLink
	}

	isChineseIP := IsChineseIP(clientIP)
	var domain string
	if isChineseIP {
		domain = "https://mikanime.tv" // Domain for Chinese IPs
	} else {
		domain = "https://mikanani.me" // Domain for non-Chinese IPs
	}

	// Ensure the path part starts with a slash.
	pathPart := linkStr
	if !strings.HasPrefix(pathPart, "/") {
		pathPart = "/" + pathPart
	}

	newLink := domain + pathPart
	return &newLink
}
