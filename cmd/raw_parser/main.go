package main

import (
	"backend/utils/parser"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	// 定义命令行参数
	var rawTitle string
	var url string
	var jsonOutput bool
	flag.StringVar(&rawTitle, "title", "", "要解析的原始动画标题")
	flag.StringVar(&url, "url", "", "要解析的Mikan动画页面URL")
	flag.BoolVar(&jsonOutput, "json", false, "以JSON格式输出解析结果")

	// 解析命令行参数
	flag.Parse()

	// 检查参数
	if rawTitle == "" && url == "" {
		fmt.Println("请提供标题或URL参数，例如: -title=\"[字幕组] 动画名称 - 01 [1080p]\" 或 -url=https://mikanani.me/Home/Bangumi/xxxx")
		os.Exit(1)
	}

	// 如果提供了URL，先使用MikanParser获取原始标题
	if url != "" {
		fmt.Printf("开始解析URL: %s\n", url)
		_, _, _, _, _, _, _, _, _, originalTitle, err := parser.GetMikanBasicInfo(url)
		if err != nil {
			fmt.Printf("解析URL失败: %v\n", err)
			os.Exit(1)
		}

		if originalTitle == "" {
			fmt.Println("未找到原始标题，请检查URL是否正确")
			os.Exit(1)
		}

		rawTitle = originalTitle
		fmt.Printf("获取到原始标题: %s\n", rawTitle)
	}

	// 使用RawParser解析原始标题
	fmt.Printf("开始解析标题: %s\n", rawTitle)
	episode := parser.RawParser(rawTitle)

	// 输出解析结果
	if episode == nil {
		fmt.Println("解析失败，无法识别标题格式")
		os.Exit(1)
	}

	if jsonOutput {
		jsonData, err := json.MarshalIndent(episode, "", "  ")
		if err != nil {
			fmt.Printf("转换为JSON失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Println("解析成功!")
		fmt.Printf("英文名称: %s\n", episode.NameEn)
		fmt.Printf("中文名称: %s\n", episode.NameZh)
		fmt.Printf("日文名称: %s\n", episode.NameJp)
		fmt.Printf("季度: %d (原始季度信息: %s)\n", episode.Season, episode.SeasonRaw)
		fmt.Printf("集数: %d\n", episode.Episode)
		fmt.Printf("字幕组: %s\n", episode.Group)
		fmt.Printf("字幕: %s\n", episode.Sub)
		fmt.Printf("分辨率: %s\n", episode.Resolution)
		fmt.Printf("来源: %s\n", episode.Source)
	}
}
