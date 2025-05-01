package rss

import (
	"backend/models"
	"backend/services/activity"
	"backend/utils"
	"backend/utils/parser"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
)

// UpdateRSSFeeds 更新所有RSS订阅源
func UpdateRSSFeeds(db *gorm.DB, force bool) error {
	var rssFeeds []models.RSSFeed

	// 启用GORM调试模式，显示SQL语句
	//db = db.Debug()
	//utils.LogInfo("启用GORM调试模式，显示SQL语句")

	// 获取所有RSS订阅源
	result := db.Find(&rssFeeds)
	if result.Error != nil {
		return fmt.Errorf("获取RSS订阅源失败: %v", result.Error)
	}

	utils.LogInfo(fmt.Sprintf("开始更新%d个RSS订阅源", len(rssFeeds)))

	// 遍历所有RSS订阅源
	for _, feed := range rssFeeds {
		// 检查是否需要更新
		if !shouldUpdate(db, feed, force) {
			continue
		}

		utils.LogInfo(fmt.Sprintf("当前订阅源[ID:%d]配置: UpdateInterval=%d小时 ParserType=%s", feed.ID, feed.UpdateInterval, feed.ParserType))

		// 根据解析器类型处理RSS源
		switch feed.ParserType {
		case "mikanani":
			utils.LogInfo(fmt.Sprintf("开始处理Mikan RSS源 %s", feed.Name))
			err := processMikanFeed(db, feed)
			utils.LogInfo(fmt.Sprintf("处理Mikan RSS源 %s 完成", feed.Name))
			if err != nil {
				utils.LogError(fmt.Sprintf("处理Mikan RSS源 %s 失败", feed.Name), err)
			}
		case "generic_rss":
			// 可以在这里添加其他类型的RSS解析器
			utils.LogInfo(fmt.Sprintf("暂不支持的解析器类型: %s", feed.ParserType))
		default:
			utils.LogInfo(fmt.Sprintf("未知的解析器类型: %s", feed.ParserType))
		}
	}

	return nil
}

// UpdateSingleRSSFeed 更新单个RSS订阅源
func UpdateSingleRSSFeed(db *gorm.DB, feedID uint) error {
	var rssFeed models.RSSFeed

	// 获取指定RSS订阅源
	result := db.First(&rssFeed, feedID)
	if result.Error != nil {
		return fmt.Errorf("获取RSS订阅源失败: %v", result.Error)
	}

	utils.LogInfo(fmt.Sprintf("开始更新单个RSS订阅源 ID:%d", feedID))

	// 根据解析器类型处理RSS源
	switch rssFeed.ParserType {
	case "mikanani":
		utils.LogInfo(fmt.Sprintf("开始处理Mikan RSS源 %s", rssFeed.Name))
		err := processMikanFeed(db, rssFeed)
		if err != nil {
			utils.LogError(fmt.Sprintf("处理Mikan RSS源 %s 失败", rssFeed.Name), err)
			return err
		}
		utils.LogInfo(fmt.Sprintf("处理Mikan RSS源 %s 完成", rssFeed.Name))
	case "generic_rss":
		return fmt.Errorf("暂不支持的解析器类型: %s", rssFeed.ParserType)
	default:
		return fmt.Errorf("未知的解析器类型: %s", rssFeed.ParserType)
	}

	return nil
}

// shouldUpdate 检查RSS源是否需要更新
func shouldUpdate(db *gorm.DB, feed models.RSSFeed, force bool) bool {
	if force {
		return true
	}
	// 根据最后更新时间判断是否需要更新
	result := time.Since(feed.UpdatedAt).Hours() >= float64(feed.UpdateInterval)
	utils.LogInfo(fmt.Sprintf("更新间隔计算：时间差%.1f小时 >= 间隔%d小时 -> %t", time.Since(feed.UpdatedAt).Hours(), feed.UpdateInterval, result))
	return result
}

// processMikanFeed 处理Mikan RSS源
func processMikanFeed(db *gorm.DB, feed models.RSSFeed) error {
	utils.LogInfo(fmt.Sprintf("开始处理RSS源[ID:%d] 名称:%s", feed.ID, feed.Name))

	// 解析URL获取主机名
	parsedURL, err := url.Parse(feed.URL)
	if err != nil {
		return fmt.Errorf("解析URL失败: %v", err)
	}

	// 获取主机名
	hostname := parsedURL.Host

	// 判断是否为Mikan网站
	isMikan := strings.Contains(hostname, "mikanani")

	// 获取RSS内容
	rssContent, err := utils.FetchURLContent(feed.URL)
	if err != nil {
		return fmt.Errorf("获取RSS内容失败: %v", err)
	}

	// 解析RSS条目链接
	itemLinks, err := parser.ParseRSS(rssContent)
	if err != nil {
		return fmt.Errorf("解析RSS内容失败: %v", err)
	}

	// 预处理关键词
	var keywords []string
	if feed.Keywords != "" {
		keywords = strings.Split(feed.Keywords, ",")
		// 去除空格
		for i := range keywords {
			keywords[i] = strings.TrimSpace(keywords[i])
		}
		utils.LogInfo(fmt.Sprintf("RSS源关键词: %v", keywords))
	}

	// 遍历所有条目链接
	for _, itemURL := range itemLinks {
		// 调用MikanParser解析具体条目
		defer func() {
			if err := recover(); err != nil {
				utils.LogError(fmt.Sprintf("处理Mikan条目 %s 失败: %v", itemURL, err), nil)
			}
		}()

		// 先获取基本信息，不包含海报
		officialTitle, subGroup, originalTitle, releaseDate, releaseYear, _, _, torrentLink, _, _, err := parser.GetMikanBasicInfo(itemURL)
		if err != nil {
			utils.LogError(fmt.Sprintf("解析Mikan条目基本信息失败 %s", itemURL), err)
			continue
		}

		// 先检查关键词匹配
		if len(keywords) > 0 {
			matched := false
			for _, keyword := range keywords {
				if strings.Contains(originalTitle, keyword) || strings.Contains(officialTitle, keyword) {
					matched = true
					utils.LogInfo(fmt.Sprintf("标题匹配关键词[%s]: %s", keyword, originalTitle))
					break
				}
			}
			if !matched {
				utils.LogInfo(fmt.Sprintf("标题不匹配任何关键词，跳过: %s", originalTitle))
				continue
			}
		}

		// 解析原始标题
		episodeInfo := parser.RawParser(originalTitle)
		if episodeInfo == nil {
			utils.LogError(fmt.Sprintf("解析原始标题失败: %s", originalTitle), nil)
			continue
		}

		// 关键词匹配成功后，再获取和保存海报
		posterPath := ""
		if shouldDownloadPoster := true; shouldDownloadPoster {
			posterPath, err = parser.DownloadMikanPoster(itemURL, db)
			if err != nil {
				utils.LogError("下载海报失败", err)
				// 海报下载失败不影响主流程
			}
		}

		// 处理番剧信息
		bangumiID, err := processOrCreateBangumi(db, officialTitle, releaseYear, episodeInfo.Season, isMikan, posterPath)
		if err != nil {
			utils.LogError(fmt.Sprintf("处理番剧信息失败: %v", err), nil)
			continue
		}

		if bangumiID == 0 {
			utils.LogError(fmt.Sprintf("无效的bangumiID[0] 来自番剧:%s", officialTitle), nil)
			continue
		}

		// 创建RSS条目
		episodeFloat := float64(episodeInfo.Episode)
		rssItem := models.RSSItem{
			BangumiID:   bangumiID,
			RssID:       feed.ID,
			Title:       officialTitle,
			URL:         torrentLink,
			Homepage:    itemURL,
			Downloaded:  false,
			Episode:     &episodeFloat,
			Resolution:  episodeInfo.Resolution,
			Group:       subGroup,
			ReleaseDate: releaseDate,
		}

		// 设置来源
		if isMikan {
			rssItem.Source = "mikan"
		} else if episodeInfo.Source != "" {
			rssItem.Source = episodeInfo.Source
		}

		// 检查是否已存在相同的RSS条目
		var existingItems []models.RSSItem
		result := db.Where("bangumi_id = ? AND rss_id = ? AND url = ?", bangumiID, feed.ID, torrentLink).Find(&existingItems)
		if result.Error != nil {
			utils.LogError("查询RSS条目失败", result.Error)
			continue
		}

		if len(existingItems) > 0 {
			utils.LogInfo(fmt.Sprintf("发现重复条目[BangumiID:%d URL:%s]，跳过创建", bangumiID, torrentLink))
			continue
		} else {
			utils.LogInfo(fmt.Sprintf("确认无重复条目[BangumiID:%d URL:%s]，开始创建新条目", bangumiID, torrentLink))
		}

		// 保存RSS条目
		result = db.Create(&rssItem)
		if result.Error != nil {
			utils.LogError("保存RSS条目失败", result.Error)
			continue
		}

		utils.LogInfo(fmt.Sprintf("成功添加RSS条目: %s (第%d集)", officialTitle, episodeInfo.Episode))
	}

	// 更新RSS源的更新时间
	utils.LogInfo(fmt.Sprintf("准备更新RSS源[ID:%d] 原更新时间：%s", feed.ID, feed.UpdatedAt.Format(time.RFC3339)))
	updateResult := db.Model(&feed).Select("UpdatedAt").Update("UpdatedAt", time.Now())
	if updateResult.Error != nil {
		utils.LogError("更新RSS源更新时间失败", updateResult.Error)
	} else if updateResult.RowsAffected > 0 {
		utils.LogInfo(fmt.Sprintf("成功更新RSS源[ID:%d]的更新时间", feed.ID))

		// 记录活动
		activityService := activity.NewActivityService(db)
		activityService.RecordActivity("rss", fmt.Sprintf("更新RSS源 \"%s\"，获取%d个新条目", feed.Name, len(itemLinks)))
	}

	return nil
}

// processOrCreateBangumi 处理或创建番剧信息
func processOrCreateBangumi(db *gorm.DB, officialTitle string, releaseYear string, season int, isMikan bool, posterPath string) (uint, error) {
	// 增加调试日志
	utils.LogInfo(fmt.Sprintf("处理番剧信息: 标题=%s, 年份=%s, 季度=%d", officialTitle, releaseYear, season))

	// 确保 season 至少为 1
	if season <= 0 {
		season = 1
		utils.LogInfo(fmt.Sprintf("季度已修正为: %d", season))
	}

	// 计算图片hash
	posterHash := ""
	if posterPath != "" {
		posterHash = utils.CalculateFileMD5(posterPath)
		utils.LogInfo(fmt.Sprintf("海报文件Hash: %s, 路径: %s", posterHash, posterPath))
	}

	// 首先检查是否存在相同hash的海报
	if posterHash != "" {
		var existingByHash models.Bangumi
		resultByHash := db.Where("poster_hash = ?", posterHash).First(&existingByHash)
		if resultByHash.Error == nil {
			utils.LogInfo(fmt.Sprintf("发现相同hash的海报记录[ID:%d]，跳过保存", existingByHash.ID))
			// 如果找到相同hash的记录，直接使用该记录，不更新其他信息
			return existingByHash.ID, nil
		}
	}

	// 通过标题和季度查找
	var existingBangumi models.Bangumi
	result := db.Where("official_title = ? AND season = ?", officialTitle, season).First(&existingBangumi)

	// 如果找到现有记录
	if result.Error == nil {
		utils.LogInfo(fmt.Sprintf("发现已存在的番剧记录[ID:%d]", existingBangumi.ID))

		// 年份为空时更新年份
		if existingBangumi.Year == nil && releaseYear != "" {
			utils.LogInfo(fmt.Sprintf("更新番剧年份: %s", releaseYear))
			existingBangumi.Year = &releaseYear
			if err := db.Save(&existingBangumi).Error; err != nil {
				utils.LogError("更新番剧年份失败", err)
			}
		}

		// 如果有新海报且hash不同，则更新海报
		if posterPath != "" && posterHash != "" &&
			(existingBangumi.PosterHash == nil || *existingBangumi.PosterHash != posterHash) {
			utils.LogInfo("检测到新的海报，准备更新")

			// 删除旧海报文件
			if existingBangumi.PosterLink != nil {
				oldPath := *existingBangumi.PosterLink
				if err := os.Remove(oldPath); err != nil {
					utils.LogError(fmt.Sprintf("删除旧海报失败[%s]", oldPath), err)
				} else {
					utils.LogInfo(fmt.Sprintf("已删除旧海报[%s]", oldPath))
				}
			}

			// 更新海报信息
			existingBangumi.PosterLink = &posterPath
			existingBangumi.PosterHash = &posterHash
			if err := db.Save(&existingBangumi).Error; err != nil {
				return 0, fmt.Errorf("更新番剧海报信息失败: %v", err)
			}
			utils.LogInfo("海报信息更新成功")
		}

		return existingBangumi.ID, nil
	}

	// 创建新番剧记录
	utils.LogInfo("未找到现有记录，准备创建新番剧")
	bangumi := models.Bangumi{
		OfficialTitle: officialTitle,
		Season:        season,
	}

	if releaseYear != "" {
		utils.LogInfo(fmt.Sprintf("设置番剧年份: %s", releaseYear))
		bangumi.Year = &releaseYear
	}

	if isMikan {
		source := "mikan"
		bangumi.Source = &source
	}

	if posterPath != "" && posterHash != "" {
		bangumi.PosterLink = &posterPath
		bangumi.PosterHash = &posterHash
	}

	// 保存番剧信息
	err := db.Create(&bangumi).Error
	if err != nil {
		utils.LogError("创建番剧记录失败", err)
		// 如果创建失败，尝试最后一次查找
		var existingRecord models.Bangumi
		if result := db.Where(
			"(official_title = ? AND season = ?) OR (poster_hash = ? AND poster_hash IS NOT NULL)",
			officialTitle, season, posterHash,
		).First(&existingRecord); result.Error == nil {
			utils.LogInfo(fmt.Sprintf("在创建失败后找到现有记录[ID:%d]", existingRecord.ID))
			return existingRecord.ID, nil
		}
		return 0, fmt.Errorf("保存番剧信息失败: %v", err)
	}

	utils.LogInfo(fmt.Sprintf("成功创建新番剧记录[ID:%d]", bangumi.ID))
	return bangumi.ID, nil
}
