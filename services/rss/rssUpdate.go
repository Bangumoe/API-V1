package rss

import (
	"backend/models"
	"backend/services/activity"
	"backend/utils"
	"backend/utils/parser"
	"fmt"
	"net/url"
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

	// 使用工作池并发处理
	numWorkers := 50 // 设置并发工作协程数量
	jobs := make(chan models.RSSFeed, len(rssFeeds))
	results := make(chan error, len(rssFeeds))

	// 启动工作协程
	for w := 0; w < numWorkers; w++ {
		go func(workerID int, jobs <-chan models.RSSFeed, results chan<- error) {
			for feed := range jobs {
				// 检查是否需要更新
				if !shouldUpdate(db, feed, force) {
					results <- nil // No error, just skip
					continue
				}

				utils.LogInfo(fmt.Sprintf("工作协程 %d 开始处理订阅源[ID:%d] 配置: UpdateInterval=%d小时 ParserType=%s", workerID, feed.ID, feed.UpdateInterval, feed.ParserType))

				// 根据解析器类型处理RSS源
				switch feed.ParserType {
				case "mikanani":
					utils.LogInfo(fmt.Sprintf("工作协程 %d 开始处理Mikan RSS源 %s", workerID, feed.Name))
					err := processMikanFeed(db, feed)
					utils.LogInfo(fmt.Sprintf("工作协程 %d 处理Mikan RSS源 %s 完成", workerID, feed.Name))
					if err != nil {
						utils.LogError(fmt.Sprintf("工作协程 %d 处理Mikan RSS源 %s 失败", workerID, feed.Name), err)
						results <- fmt.Errorf("处理Mikan RSS源 %s 失败: %v", feed.Name, err)
					} else {
						results <- nil
					}
				case "generic_rss":
					utils.LogInfo(fmt.Sprintf("工作协程 %d 暂不支持的解析器类型: %s", workerID, feed.ParserType))
					results <- fmt.Errorf("暂不支持的解析器类型: %s", feed.ParserType)
				default:
					utils.LogInfo(fmt.Sprintf("工作协程 %d 未知的解析器类型: %s", workerID, feed.ParserType))
					results <- fmt.Errorf("未知的解析器类型: %s", feed.ParserType)
				}
			}
		}(w, jobs, results)
	}

	// 发送任务
	for _, feed := range rssFeeds {
		jobs <- feed
	}
	close(jobs)

	// 收集结果
	var updateErrors []error
	for a := 0; a < len(rssFeeds); a++ {
		err := <-results
		if err != nil {
			updateErrors = append(updateErrors, err)
		}
	}

	utils.LogInfo("所有RSS订阅源更新任务完成")

	// 如果有错误，返回第一个错误
	if len(updateErrors) > 0 {
		return fmt.Errorf("部分RSS订阅源更新失败: %v", updateErrors)
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

	// 处理分页逻辑
	pageStart := 1
	pageEnd := 1
	if feed.PageStart != nil && feed.PageEnd != nil && *feed.PageEnd >= *feed.PageStart {
		pageStart = *feed.PageStart
		pageEnd = *feed.PageEnd
	}

	// 使用工作池并发处理分页
	numPageWorkers := 250 // 设置并发处理分页的协程数量，可以根据实际情况调整
	pageJobs := make(chan string, pageEnd-pageStart+1)
	pageResults := make(chan error, pageEnd-pageStart+1)

	// 启动分页工作协程
	for w := 0; w < numPageWorkers; w++ {
		go func(workerID int, pageJobs <-chan string, pageResults chan<- error) {
			for pageURL := range pageJobs {
				utils.LogInfo(fmt.Sprintf("分页工作协程 %d 抓取分页URL: %s", workerID, pageURL))
				// 获取RSS内容，增加重试和延迟
				rssContent, err := utils.FetchURLContentWithRetry(pageURL, 3, 2*time.Second)
				if err != nil {
					utils.LogError(fmt.Sprintf("分页工作协程 %d 获取RSS内容失败: %s", workerID, pageURL), err)
					pageResults <- fmt.Errorf("获取RSS内容失败: %s, %v", pageURL, err)
					continue
				}

				// 解析RSS条目链接
				itemLinks, err := parser.ParseRSS(string(rssContent))
				if err != nil {
					utils.LogError(fmt.Sprintf("分页工作协程 %d 解析RSS内容失败: %s", workerID, pageURL), err)
					pageResults <- fmt.Errorf("解析RSS内容失败: %s, %v", pageURL, err)
					continue
				}

				// 获取全局设置
				settings, err := models.GetGlobalSettings()
				if err != nil {
					utils.LogError("获取全局设置失败", err)
					continue
				}

				// 合并全局关键词和订阅源关键词
				var allKeywords []string
				if settings.GlobalKeywords != "" {
					for _, k := range strings.Split(settings.GlobalKeywords, ",") {
						allKeywords = append(allKeywords, strings.TrimSpace(k))
					}
				}
				if feed.Keywords != "" {
					for _, k := range strings.Split(feed.Keywords, ",") {
						allKeywords = append(allKeywords, strings.TrimSpace(k))
					}
				}

				// 合并全局排除关键词和订阅源排除关键词
				var allExcludeKeywords []string
				if settings.ExcludeKeywords != "" {
					for _, k := range strings.Split(settings.ExcludeKeywords, ",") {
						allExcludeKeywords = append(allExcludeKeywords, strings.TrimSpace(k))
					}
				}
				if feed.ExcludeKeywords != "" {
					for _, k := range strings.Split(feed.ExcludeKeywords, ",") {
						allExcludeKeywords = append(allExcludeKeywords, strings.TrimSpace(k))
					}
				}

				// 遍历所有条目链接
				for _, itemURL := range itemLinks {
					defer func() {
						if err := recover(); err != nil {
							utils.LogError(fmt.Sprintf("分页工作协程 %d 处理Mikan条目 %s 失败: %v", workerID, itemURL, err), nil)
						}
					}()

					// 先获取基本信息，不包含海报
					officialTitle, subGroup, originalTitle, releaseDate, releaseYear, _, _, torrentLink, _, _, err := parser.GetMikanBasicInfo(itemURL)
					if err != nil {
						utils.LogError(fmt.Sprintf("分页工作协程 %d 解析Mikan条目基本信息失败 %s", workerID, itemURL), err)
						continue
					}

					// 跳过无title的条目，防止污染bangumi_id=1
					if officialTitle == "" {
						utils.LogError(fmt.Sprintf("跳过无效条目：officialTitle为空，itemURL=%s", itemURL), nil)
						continue
					}

					// 全局排除关键词优先级最高
					for _, ex := range allExcludeKeywords {
						if ex != "" && (strings.Contains(originalTitle, ex) || strings.Contains(officialTitle, ex)) {
							utils.LogInfo(fmt.Sprintf("分页工作协程 %d 命中排除关键词[%s]，跳过", workerID, ex))
							continue // 跳过该item
						}
					}

					// 解析原始标题
					episodeInfo := parser.RawParser(originalTitle, settings.SubGroupBlacklist)
					if episodeInfo == nil {
						utils.LogError(fmt.Sprintf("分页工作协程 %d 解析原始标题失败: %s", workerID, originalTitle), nil)
						continue
					}

					// 检查字幕信息是否符合要求
					validChineseSubs := []string{"简体", "简日", "简", "CHS", "GB", "简日繁", "简中", "bibili", "Bilibili"}
					isChineseSub := false
					for _, validSub := range validChineseSubs {
						if episodeInfo.Sub == validSub {
							isChineseSub = true
							break
						}
					}

					if isChineseSub {
						utils.LogInfo(fmt.Sprintf("分页工作协程 %d 标题 '%s' (字幕 '%s') 符合字幕要求，优先处理", workerID, originalTitle, episodeInfo.Sub))
						// 字幕符合要求，直接进入后续处理流程
					} else {
						// 字幕不符合要求，需要检查关键词（合并后的allKeywords）
						if len(allKeywords) > 0 {
							matched := false
							for _, keyword := range allKeywords {
								if keyword != "" && (strings.Contains(originalTitle, keyword) || strings.Contains(officialTitle, keyword)) {
									matched = true
									utils.LogInfo(fmt.Sprintf("分页工作协程 %d (字幕 '%s' 不符后) 标题匹配关键词[%s]: %s (官方: %s)", workerID, episodeInfo.Sub, keyword, originalTitle, officialTitle))
									break
								}
							}
							if !matched {
								utils.LogInfo(fmt.Sprintf("分页工作协程 %d 标题 '%s' (字幕 '%s') 不符合字幕要求，且不匹配任何关键词，跳过", workerID, originalTitle, episodeInfo.Sub))
								continue
							}
							// 字幕不符但关键词匹配，继续处理
							utils.LogInfo(fmt.Sprintf("分页工作协程 %d 标题 '%s' (字幕 '%s' 不符) 但匹配关键词，继续处理", workerID, originalTitle, episodeInfo.Sub))
						} else {
							// 字幕不符合要求，且没有设置关键词，则跳过
							utils.LogInfo(fmt.Sprintf("分页工作协程 %d 标题 '%s' (字幕 '%s') 不符合字幕要求 (无关键词)，跳过", workerID, originalTitle, episodeInfo.Sub))
							continue
						}
					}

					// 关键词匹配成功后，再获取海报URL
					posterURL := ""
					if shouldGetPosterURL := true; shouldGetPosterURL {
						posterURL, err = parser.GetMikanPosterURL(itemURL)
						if err != nil {
							utils.LogError(fmt.Sprintf("分页工作协程 %d 获取海报URL失败: %s", workerID, itemURL), err)
							// 海报URL获取失败不影响主流程
						}
					}

					// 处理番剧信息
					bangumiID, err := processOrCreateBangumi(db, officialTitle, releaseYear, episodeInfo.Season, isMikan, posterURL)
					if err != nil {
						utils.LogError(fmt.Sprintf("分页工作协程 %d 处理番剧信息失败: %v", workerID, err), nil)
						continue
					}

					if bangumiID == 0 {
						utils.LogError(fmt.Sprintf("分页工作协程 %d 无效的bangumiID[0] 来自番剧:%s", workerID, officialTitle), nil)
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
						Sub:         episodeInfo.Sub,
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
						utils.LogError("分页工作协程 %d 查询RSS条目失败", result.Error)
						continue
					}

					if len(existingItems) > 0 {
						utils.LogInfo(fmt.Sprintf("分页工作协程 %d 发现重复条目[BangumiID:%d URL:%s]，跳过创建", workerID, bangumiID, torrentLink))
						continue
					} else {
						utils.LogInfo(fmt.Sprintf("分页工作协程 %d 确认无重复条目[BangumiID:%d URL:%s]，开始创建新条目", workerID, bangumiID, torrentLink))
					}

					// 保存RSS条目
					result = db.Create(&rssItem)
					if result.Error != nil {
						utils.LogError("分页工作协程 %d 保存RSS条目失败", result.Error)
						continue
					}

					utils.LogInfo(fmt.Sprintf("分页工作协程 %d 成功添加RSS条目: %s (第%d集)", workerID, officialTitle, episodeInfo.Episode))
				}
			}
		}(w, pageJobs, pageResults)
	}

	// 发送分页任务
	for page := pageStart; page <= pageEnd; page++ {
		var pageURL string
		if page == 1 {
			pageURL = feed.URL
		} else {
			if strings.HasSuffix(feed.URL, "/") {
				pageURL = fmt.Sprintf("%s%d", feed.URL, page)
			} else {
				pageURL = fmt.Sprintf("%s/%d", feed.URL, page)
			}
		}
		pageJobs <- pageURL
	}
	close(pageJobs)

	// 收集分页结果
	var pageErrors []error
	for i := 0; i < (pageEnd - pageStart + 1); i++ {
		err := <-pageResults
		if err != nil {
			pageErrors = append(pageErrors, err)
		}
	}

	// 如果有分页处理错误，记录下来但不中断整个feed的处理
	if len(pageErrors) > 0 {
		utils.LogError(fmt.Sprintf("RSS源[ID:%d] 分页处理完成，发现 %d 个分页处理错误", feed.ID, len(pageErrors)), fmt.Errorf("%v", pageErrors))
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
		activityService.RecordActivity("rss", fmt.Sprintf("更新RSS源 \"%s\"，分页范围[%d-%d]，每页获取新条目", feed.Name, pageStart, pageEnd))
	}

	return nil
}

// processOrCreateBangumi 处理或创建番剧信息
func processOrCreateBangumi(db *gorm.DB, officialTitle string, releaseYear string, season int, isMikan bool, posterURL string) (uint, error) {
	utils.LogInfo(fmt.Sprintf("处理番剧信息: 标题=%s, 年份=%s, 季度=%d, 海报URL: %s", officialTitle, releaseYear, season, posterURL))

	if season <= 0 {
		season = 1
		utils.LogInfo(fmt.Sprintf("季度已修正为: %d", season))
	}

	var bangumi models.Bangumi

	err := db.Transaction(func(tx *gorm.DB) error {
		// Define the record to find or create based on primary identifiers
		findCondition := models.Bangumi{OfficialTitle: officialTitle, Season: season}

		// Use FirstOrInit to load or initialize the struct, then decide on updates
		if err := tx.Where(findCondition).FirstOrInit(&bangumi).Error; err != nil {
			utils.LogError(fmt.Sprintf("FirstOrInit 失败 for Title '%s' Season %d", officialTitle, season), err)
			return fmt.Errorf("FirstOrInit 番剧信息失败: %v", err)
		}

		isNewRecord := bangumi.ID == 0
		needsSave := false

		// Update fields based on new data
		if releaseYear != "" && (bangumi.Year == nil || *bangumi.Year != releaseYear) {
			bangumi.Year = &releaseYear
			needsSave = true
			utils.LogInfo(fmt.Sprintf("番剧[ID:%d Title:%s S:%d] 年份更新为: %s", bangumi.ID, officialTitle, season, releaseYear))
		}

		if isMikan {
			sourceMikan := "mikan"
			if bangumi.Source == nil || *bangumi.Source != sourceMikan {
				bangumi.Source = &sourceMikan
				needsSave = true
				utils.LogInfo(fmt.Sprintf("番剧[ID:%d Title:%s S:%d] 来源更新为: mikan", bangumi.ID, officialTitle, season))
			}
		}

		// 更新海报链接，不再处理海报哈希和本地文件
		if posterURL != "" {
			if bangumi.PosterLink == nil || *bangumi.PosterLink != posterURL {
				bangumi.PosterLink = &posterURL
				needsSave = true
				utils.LogInfo(fmt.Sprintf("番剧[ID:%d Title:%s S:%d] 海报链接更新为: %s", bangumi.ID, officialTitle, season, posterURL))
			}
		} else if bangumi.PosterLink != nil { // 如果传入的posterURL为空，且数据库中存在海报链接，则清空
			bangumi.PosterLink = nil
			needsSave = true
			utils.LogInfo(fmt.Sprintf("番剧[ID:%d Title:%s S:%d] 海报链接被清空", bangumi.ID, officialTitle, season))
		}

		// PosterHash字段不再主动管理，如果模型定义中它依赖于本地文件，则应在此处设为nil或根据新逻辑处理
		if bangumi.PosterHash != nil { // 如果之前有 PosterHash，现在不再使用，则清空
			bangumi.PosterHash = nil
			needsSave = true // 确保更改被保存
			utils.LogInfo(fmt.Sprintf("番剧[ID:%d Title:%s S:%d] PosterHash 已清空", bangumi.ID, officialTitle, season))
		}

		if isNewRecord {
			utils.LogInfo(fmt.Sprintf("准备创建新番剧: Title '%s' Season %d", officialTitle, season))
			if err := tx.Create(&bangumi).Error; err != nil {
				utils.LogError(fmt.Sprintf("创建新番剧失败: Title '%s' Season %d", officialTitle, season), err)
				return fmt.Errorf("创建新番剧失败: %v", err)
			}
			utils.LogInfo(fmt.Sprintf("成功创建新番剧[ID:%d]: Title '%s' Season %d", bangumi.ID, officialTitle, season))
		} else if needsSave {
			utils.LogInfo(fmt.Sprintf("准备更新已存在番剧[ID:%d]: Title '%s' Season %d", bangumi.ID, officialTitle, season))
			if err := tx.Save(&bangumi).Error; err != nil {
				utils.LogError(fmt.Sprintf("保存番剧[ID:%d]更新失败", bangumi.ID), err)
				return fmt.Errorf("保存番剧更新失败: %v", err)
			}
			utils.LogInfo(fmt.Sprintf("成功更新番剧[ID:%d]", bangumi.ID))
		} else {
			utils.LogInfo(fmt.Sprintf("番剧[ID:%d Title:%s S:%d] 无需更新", bangumi.ID, officialTitle, season))
		}

		// 不再需要删除旧海报文件的逻辑
		return nil // Transaction success
	})

	if err != nil {
		utils.LogError(fmt.Sprintf("处理番剧信息事务失败 for Title '%s' Season %d", officialTitle, season), err)
		// Fallback read
		var fallbackBangumi models.Bangumi
		// 查询时不再依赖 PosterHash
		if db.Where("official_title = ? AND season = ?", officialTitle, season).First(&fallbackBangumi).Error == nil {
			utils.LogInfo(fmt.Sprintf("事务失败后, 成功通过查询找到番剧记录 [ID:%d]", fallbackBangumi.ID))
			return fallbackBangumi.ID, nil
		}
		return 0, fmt.Errorf("处理番剧信息失败 (事务后最终错误): %v", err)
	}

	utils.LogInfo(fmt.Sprintf("成功处理番剧信息[ID:%d] for Title '%s' Season %d", bangumi.ID, officialTitle, season))
	return bangumi.ID, nil
}
