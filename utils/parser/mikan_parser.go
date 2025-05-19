package parser

import (
	"backend/utils"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// parseRelativeDate 处理相对日期
func parseRelativeDate(text string) (string, string, string, string) {
	now := time.Now()
	var targetDate time.Time

	switch {
	case strings.Contains(text, "昨天"):
		targetDate = now.AddDate(0, 0, -1)
	case strings.Contains(text, "前天"):
		targetDate = now.AddDate(0, 0, -2)
	case strings.Contains(text, "今天"):
		targetDate = now
	default:
		return "", "", "", ""
	}

	return formatDate(targetDate)
}

// formatDate 格式化日期
func formatDate(t time.Time) (string, string, string, string) {
	releaseDate := t.Format("2006/01/02")
	releaseYear := t.Format("2006")
	releaseMonth := t.Format("01")
	releaseDay := t.Format("02")
	return releaseDate, releaseYear, releaseMonth, releaseDay
}

// GetMikanBasicInfo 获取Mikan页面的基本信息（不包含海报）
func GetMikanBasicInfo(homepage string) (string, string, string, string, string, string, string, string, string, string, error) {
	// 解析URL获取主机名
	parsedURL, err := url.Parse(homepage)
	if err != nil {
		utils.LogError("解析URL失败", err)
		return "", "", "", "", "", "", "", "", "", "", err
	}
	rootPath := parsedURL.Host

	// 发送HTTP请求获取页面内容
	resp, err := http.Get(homepage)
	if err != nil {
		utils.LogError("请求页面失败", err)
		return "", "", "", "", "", "", "", "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := errors.New(fmt.Sprintf("请求失败，状态码：%d", resp.StatusCode))
		utils.LogError("请求页面失败", err)
		return "", "", "", "", "", "", "", "", "", "", err
	}

	// 使用goquery解析HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		utils.LogError("解析HTML失败", err)
		return "", "", "", "", "", "", "", "", "", "", err
	}

	// 提取官方标题
	officialTitle := ""
	doc.Find("p.bangumi-title a[href^='/Home/Bangumi/']").Each(func(i int, s *goquery.Selection) {
		officialTitle = strings.TrimSpace(s.Text())
	})

	// 使用正则表达式移除"第X季"的文本
	re := regexp.MustCompile(`第.*季`)
	officialTitle = strings.TrimSpace(re.ReplaceAllString(officialTitle, ""))

	// 提取字幕组信息
	subGroup := ""
	doc.Find("p.bangumi-info a.magnet-link-wrap[href^='/Home/PublishGroup/']").Each(func(i int, s *goquery.Selection) {
		subGroup = strings.TrimSpace(s.Text())
	})

	// 提取原始标题
	originalTitle := ""
	doc.Find("div.central-container div.episode-header p.episode-title").Each(func(i int, s *goquery.Selection) {
		originalTitle = strings.TrimSpace(s.Text())
	})

	// 提取发布时间
	releaseDate := ""
	releaseYear := ""
	releaseMonth := ""
	releaseDay := ""
	doc.Find("p.bangumi-info").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if strings.Contains(text, "发布日期：") {
			dateText := strings.TrimPrefix(text, "发布日期：")
			dateText = strings.TrimSpace(dateText)
			utils.LogInfo(fmt.Sprintf("原始发布日期文本: %s", dateText))

			// 处理相对日期（昨天、今天等）
			if strings.Contains(dateText, "天") {
				releaseDate, releaseYear, releaseMonth, releaseDay = parseRelativeDate(dateText)
				utils.LogInfo(fmt.Sprintf("解析相对日期: %s -> %s/%s/%s", dateText, releaseYear, releaseMonth, releaseDay))
				return
			}

			// 处理标准日期格式
			parts := strings.Split(dateText, "/")
			if len(parts) >= 3 {
				releaseDate = dateText
				releaseYear = parts[0]
				releaseMonth = parts[1]
				dayTime := strings.Split(parts[2], " ")
				releaseDay = dayTime[0]
				utils.LogInfo(fmt.Sprintf("解析标准日期: %s -> %s/%s/%s", dateText, releaseYear, releaseMonth, releaseDay))
				return
			}

			utils.LogError(fmt.Sprintf("无法解析的日期格式: %s", dateText), nil)
		}
	})

	// 提取种子链接
	torrentLink := ""
	doc.Find("div.leftbar-nav a.episode-btn[href$='.torrent']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			torrentLink = fmt.Sprintf("https://%s%s", rootPath, href)
		}
	})

	// 提取磁力链接
	magnetLink := ""
	doc.Find("div.leftbar-nav a.episode-btn[href^='magnet:']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			magnetLink = href
		}
	})

	// 返回值顺序应该是：
	// officialTitle, subGroup, originalTitle, releaseDate, releaseYear, releaseMonth, releaseDay, torrentLink, magnetLink, nil
	return officialTitle, subGroup, originalTitle, releaseDate, releaseYear, releaseMonth, releaseDay, torrentLink, magnetLink, "", nil
}

// GetMikanPosterURL 获取Mikan页面的海报URL
func GetMikanPosterURL(homepage string) (string, error) {
	// 解析URL获取主机名
	parsedURL, err := url.Parse(homepage)
	if err != nil {
		return "", err
	}
	rootPath := parsedURL.Host

	// 获取海报URL
	resp, err := http.Get(homepage)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	// 提取海报链接
	var posterPath string
	posterDiv := doc.Find("div.bangumi-poster").AttrOr("style", "")
	if posterDiv == "" {
		return "", nil
	}

	re := regexp.MustCompile(`url\('([^']+)'\)`)
	matches := re.FindStringSubmatch(posterDiv)
	if len(matches) <= 1 {
		return "", nil
	}

	posterPath = matches[1]
	if strings.Contains(posterPath, "?") {
		posterPath = strings.Split(posterPath, "?")[0]
	}

	// 返回海报的完整URL
	posterFullURL := fmt.Sprintf("https://%s%s", rootPath, posterPath)
	return posterFullURL, nil
}
