package parser

import (
	"backend/models"
	"backend/utils"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"gorm.io/gorm"
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

// DownloadMikanPoster 下载并保存海报
func DownloadMikanPoster(homepage string, db *gorm.DB) (string, error) {
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

	// 下载并保存海报
	imgURL := fmt.Sprintf("https://%s%s", rootPath, posterPath)
	return downloadAndSaveImage(imgURL, db)
}

// saveImage 保存图片到本地
// 参数：imgData - 图片数据，suffix - 图片后缀，db - 数据库连接
// 返回：保存的图片路径
func saveImage(imgData []byte, suffix string, db *gorm.DB) (string, error) {
	// 计算图片数据的hash
	hash := fmt.Sprintf("%x", md5.Sum(imgData))

	// 查询是否存在相同hash的记录
	var existingBangumi models.Bangumi
	if result := db.Where("poster_hash = ?", hash).First(&existingBangumi); result.Error == nil {
		if existingBangumi.PosterLink != nil {
			utils.LogInfo(fmt.Sprintf("发现相同hash的图片[%s]，直接使用已有文件", *existingBangumi.PosterLink))
			return *existingBangumi.PosterLink, nil
		}
	}

	// 确保uploads目录存在
	uploadsDir := "./uploads/posters"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", err
	}

	// 生成唯一文件名
	timestamp := time.Now().UnixNano()
	fileName := fmt.Sprintf("poster_%d%s", timestamp, suffix)
	filePath := filepath.Join(uploadsDir, fileName)

	// 写入文件
	if err := os.WriteFile(filePath, imgData, 0644); err != nil {
		return "", err
	}

	utils.LogInfo(fmt.Sprintf("已保存新图片: %s", filePath))
	return filePath, nil
}

// downloadAndSaveImage 下载并保存图片
func downloadAndSaveImage(imgURL string, db *gorm.DB) (string, error) {
	// 下载图片
	imgResp, err := http.Get(imgURL)
	if err != nil {
		utils.LogError("下载图片失败", err)
		return "", err
	}
	defer imgResp.Body.Close()

	if imgResp.StatusCode != http.StatusOK {
		err := errors.New(fmt.Sprintf("下载图片失败，状态码：%d", imgResp.StatusCode))
		utils.LogError("下载图片失败", err)
		return "", err
	}

	// 读取图片内容
	imgData, err := io.ReadAll(imgResp.Body)
	if err != nil {
		utils.LogError("读取图片内容失败", err)
		return "", err
	}

	// 获取图片后缀
	suffix := filepath.Ext(imgURL)
	if suffix == "" {
		suffix = ".jpg" // 默认后缀
	}

	// 保存图片
	return saveImage(imgData, suffix, db)
}
