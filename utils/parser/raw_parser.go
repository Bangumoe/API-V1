package parser

import (
	"backend/utils"
	"regexp"
	"strconv"
	"strings"
)

// Episode 表示一个动画剧集的信息
type Episode struct {
	NameEn     string // 英文名称
	NameZh     string // 中文名称
	NameJp     string // 日文名称
	Season     int    // 季度数字
	SeasonRaw  string // 原始季度信息
	Episode    int    // 集数
	Sub        string // 字幕信息
	Group      string // 字幕组
	Resolution string // 分辨率
	Source     string // 来源
}

// 定义正则表达式
var (
	episodeRE    = regexp.MustCompile(`\d+`)
	titleRE      = regexp.MustCompile(`(.*|\[.*])( -? \d+|\[\d+]|\[\d+.?[vV]\d]|第\d+[话話集]|\[第?\d+[话話集]]|\[\d+.?END]|[Ee][Pp]?\d+)(.*)`)
	resolutionRE = regexp.MustCompile(`1080|720|2160|4K`)
	sourceRE     = regexp.MustCompile(`B-Global|[Bb]aha|[Bb]ilibili|AT-X|Web`)
	subRE        = regexp.MustCompile(`[简繁日字幕]|CH|BIG5|GB|CHS|CHT|JP|ENG|简中|繁中|中字`) // Expanded for general fallback
	prefixRE     = regexp.MustCompile(`[^\w\s\p{Han}\p{Hiragana}\p{Katakana}-]`)
)

// selectionPrioritySubKeywords defines the order of preference for selecting Chinese subtitle tags.
// Higher priority (lower index) keywords are preferred.
var selectionPrioritySubKeywords = []string{"简体", "简日", "简", "CHS", "GB", "简日繁", "简中", "bibili", "Bilibili"}

// 中文数字映射
var chineseNumberMap = map[string]int{
	"一": 1,
	"二": 2,
	"三": 3,
	"四": 4,
	"五": 5,
	"六": 6,
	"七": 7,
	"八": 8,
	"九": 9,
	"十": 10,
}

// getGroup 获取字幕组名称
func getGroup(name string) string {
	parts := regexp.MustCompile(`[\[\]]`).Split(name, -1)
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// preProcess 预处理标题
func preProcess(rawName string) string {
	return strings.ReplaceAll(strings.ReplaceAll(rawName, "【", "["), "】", "]")
}

// prefixProcess 处理前缀
func prefixProcess(raw, group string) string {
	// 替换字幕组信息
	raw = regexp.MustCompile("."+group+".").ReplaceAllString(raw, "")

	// 处理前缀
	rawProcess := prefixRE.ReplaceAllString(raw, "/")
	argGroup := strings.Split(rawProcess, "/")

	// 移除空字符串
	var filteredArgGroup []string
	for _, arg := range argGroup {
		if arg != "" {
			filteredArgGroup = append(filteredArgGroup, arg)
		}
	}
	argGroup = filteredArgGroup

	// 如果只有一个元素，按空格分割
	if len(argGroup) == 1 {
		argGroup = strings.Split(argGroup[0], " ")
	}

	// 处理特殊标记
	for _, arg := range argGroup {
		if regexp.MustCompile(`新番|月?番`).MatchString(arg) && len(arg) <= 5 {
			raw = regexp.MustCompile("."+arg+".").ReplaceAllString(raw, "")
		} else if regexp.MustCompile(`港澳台地区`).MatchString(arg) {
			raw = regexp.MustCompile("."+arg+".").ReplaceAllString(raw, "")
		}
	}

	return raw
}

// seasonProcess 处理季度信息
func seasonProcess(seasonInfo string) (string, string, int) {
	nameSeason := seasonInfo

	// 替换方括号为空格
	nameSeason = regexp.MustCompile(`[\[\]]`).ReplaceAllString(nameSeason, " ")

	// 查找季度信息
	seasonRule := `S\d{1,2}|Season \d{1,2}|[第].[季期]`
	seasons := regexp.MustCompile(seasonRule).FindAllString(nameSeason, -1)

	// 如果没有找到季度信息，返回原始名称和默认季度1
	if len(seasons) == 0 {
		return nameSeason, "", 1
	}

	// 移除季度信息，获取纯名称
	name := regexp.MustCompile(seasonRule).ReplaceAllString(nameSeason, "")

	// 解析季度数字
	season := 1
	seasonRaw := seasons[0]

	for _, s := range seasons {
		if regexp.MustCompile(`Season|S`).MatchString(s) {
			// 处理英文季度格式
			seasonStr := regexp.MustCompile(`Season|S`).ReplaceAllString(s, "")
			seasonNum, err := strconv.Atoi(strings.TrimSpace(seasonStr))
			if err == nil {
				season = seasonNum
				break
			}
		} else if regexp.MustCompile(`[第 ].*[季期(部分)]|部分`).MatchString(s) {
			// 处理中文季度格式
			seasonPro := regexp.MustCompile(`[第季期 ]`).ReplaceAllString(s, "")
			seasonPro = strings.TrimSpace(seasonPro)

			// 尝试转换为数字
			seasonNum, err := strconv.Atoi(seasonPro)
			if err == nil {
				season = seasonNum
				break
			}

			// 尝试从中文数字映射中获取
			if val, ok := chineseNumberMap[seasonPro]; ok {
				season = val
				break
			}
		}
	}

	return name, seasonRaw, season
}

// nameProcess 处理名称，分离英文、中文和日文名称
func nameProcess(name string) (string, string, string) {
	var nameEn, nameZh, nameJp string

	// 去除空白和特殊标记
	name = strings.TrimSpace(name)
	name = regexp.MustCompile(`[(（]仅限港澳台地区[）)]`).ReplaceAllString(name, "")

	// 尝试按不同分隔符分割
	split := regexp.MustCompile(`/|\s{2}|-\s{2}`).Split(name, -1)

	// 移除空字符串
	var filteredSplit []string
	for _, s := range split {
		if s != "" {
			filteredSplit = append(filteredSplit, s)
		}
	}
	split = filteredSplit

	// 如果只有一个元素，尝试其他分隔符
	if len(split) == 1 {
		if regexp.MustCompile(`_{1}`).MatchString(name) {
			split = strings.Split(name, "_")
		} else if regexp.MustCompile(` - {1}`).MatchString(name) {
			split = strings.Split(name, "-")
		}
	}

	// 如果仍然只有一个元素，尝试按空格分割并识别中文部分
	if len(split) == 1 {
		splitSpace := strings.Split(split[0], " ")
		if len(splitSpace) > 0 {
			// 检查首尾是否有中文
			indices := []int{0}
			if len(splitSpace) > 1 {
				indices = append(indices, len(splitSpace)-1)
			}

			for _, idx := range indices {
				if regexp.MustCompile(`^\p{Han}{2,}`).MatchString(splitSpace[idx]) {
					chs := splitSpace[idx]

					// 创建不包含中文部分的新切片
					var newSplitSpace []string
					for i, s := range splitSpace {
						if i != idx {
							newSplitSpace = append(newSplitSpace, s)
						}
					}

					split = []string{chs, strings.Join(newSplitSpace, " ")}
					break
				}
			}
		}
	}

	// 识别不同语言的名称
	for _, item := range split {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		if regexp.MustCompile(`\p{Hiragana}|\p{Katakana}{2,}`).MatchString(item) && nameJp == "" {
			nameJp = item
		} else if regexp.MustCompile(`\p{Han}{2,}`).MatchString(item) && nameZh == "" {
			nameZh = item
		} else if regexp.MustCompile(`[a-zA-Z]{3,}`).MatchString(item) && nameEn == "" {
			nameEn = item
		}
	}

	return nameEn, nameZh, nameJp
}

// findTags 查找标签信息（字幕、分辨率、来源）
func findTags(other string) (string, string, string) {
	elements := strings.Split(regexp.MustCompile(`[\[\]()（）]`).ReplaceAllString(other, " "), " ")

	var sub, resolution, source string
	var bestSubMatch string
	highestPriority := -1 // Lower index in selectionPrioritySubKeywords means higher priority

	// First pass: check for prioritized Chinese subs based on selectionPrioritySubKeywords
	for _, element := range elements {
		element = strings.TrimSpace(element)
		if element == "" {
			continue
		}
		for i, keyword := range selectionPrioritySubKeywords {
			// Check if the element (which is a potential tag) contains the keyword.
			// This allows matching for elements like "CHS字幕" with keyword "CHS".
			if strings.Contains(element, keyword) {
				if highestPriority == -1 || i < highestPriority {
					bestSubMatch = keyword // Assign the keyword itself as the sub tag
					highestPriority = i
				}
			}
		}
		// Concurrently check for resolution and source from the same element
		if resolutionRE.MatchString(element) {
			resolution = element
		} else if sourceRE.MatchString(element) {
			source = element
		}
	}

	// Second pass: if no prioritized Chinese sub was found, check for any other sub using the general subRE
	if bestSubMatch == "" {
		for _, element := range elements {
			element = strings.TrimSpace(element)
			if element == "" {
				continue
			}
			// Check if this element is a general subtitle tag according to subRE
			if subRE.MatchString(element) {
				// Ensure it's not one of the selectionPrioritySubKeywords that might have been missed
				// or to prevent complex interactions if subRE is broad.
				isAlreadyPrioritizedCandidate := false
				for _, prioKeyword := range selectionPrioritySubKeywords {
					if strings.Contains(element, prioKeyword) {
						isAlreadyPrioritizedCandidate = true
						break
					}
				}
				if !isAlreadyPrioritizedCandidate {
					// Use the element itself if it matches subRE, as subRE is designed to match whole tags.
					bestSubMatch = element
					break // Found a general sub, stop this loop
				}
			}
		}
	}

	sub = bestSubMatch
	return cleanSub(sub), resolution, source
}

// cleanSub 清理字幕信息
func cleanSub(sub string) string {
	if sub == "" {
		return sub
	}
	return regexp.MustCompile(`_MP4|_MKV`).ReplaceAllString(sub, "")
}

// process 处理原始标题
func process(rawTitle string) (string, string, string, int, string, int, string, string, string, string) {
	// 预处理标题
	rawTitle = strings.TrimSpace(rawTitle)
	rawTitle = strings.ReplaceAll(rawTitle, "\n", " ")
	contentTitle := preProcess(rawTitle)

	// 获取字幕组名称
	group := getGroup(contentTitle)

	// 匹配标题结构
	matchObj := titleRE.FindStringSubmatch(contentTitle)
	if len(matchObj) < 4 {
		utils.LogError("解析标题失败", nil)
		return "", "", "", 0, "", 0, "", "", "", group
	}

	// 提取季度信息、集数信息和其他信息
	seasonInfo := strings.TrimSpace(matchObj[1])
	episodeInfo := strings.TrimSpace(matchObj[2])
	other := strings.TrimSpace(matchObj[3])

	// 处理前缀
	processRaw := prefixProcess(seasonInfo, group)

	// 处理季度
	rawName, seasonRaw, season := seasonProcess(processRaw)

	// 处理名称
	nameEn, nameZh, nameJp := "", "", ""
	try := func() {
		nameEn, nameZh, nameJp = nameProcess(rawName)
	}
	try()

	// 处理集数
	episode := 0
	rawEpisode := episodeRE.FindString(episodeInfo)
	if rawEpisode != "" {
		episodeNum, err := strconv.Atoi(rawEpisode)
		if err == nil {
			episode = episodeNum
		}
	}

	// 处理其他标签
	sub, resolution, source := findTags(other)

	return nameEn, nameZh, nameJp, season, seasonRaw, episode, sub, resolution, source, group
}

// RawParser 解析原始标题并返回Episode对象
func RawParser(raw string) *Episode {
	nameEn, nameZh, nameJp, season, seasonRaw, episode, sub, resolution, source, group := process(raw)

	// 如果解析失败，记录错误并返回nil
	if nameEn == "" && nameZh == "" && nameJp == "" {
		utils.LogError("解析器无法解析标题", nil)
		return nil
	}

	// 创建并返回Episode对象
	return &Episode{
		NameEn:     nameEn,
		NameZh:     nameZh,
		NameJp:     nameJp,
		Season:     season,
		SeasonRaw:  seasonRaw,
		Episode:    episode,
		Sub:        sub,
		Group:      group,
		Resolution: resolution,
		Source:     source,
	}
}
