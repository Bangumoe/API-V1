package test

import (
	"backend/utils/parser"
	"fmt"
	"testing"
)

func TestMikanAndRawParserIntegration(t *testing.T) {
	// 示例URL，可替换为实际测试用的Mikan番剧页面
	url := "https://mikanani.me/Home/Episode/64d3baea9d1aa93a21caf1d802639be4437b01d0"

	posterPath, officialTitle, subGroup, originalTitle, releaseDate, releaseYear, _, _, torrentLink, _, err := parser.MikanParser(url)
	if err != nil {
		t.Fatalf("MikanParser 解析失败: %v", err)
	}
	if originalTitle == "" {
		t.Fatalf("未获取到原始标题")
	}

	episode := parser.RawParser(originalTitle)
	if episode == nil {
		t.Fatalf("RawParser 解析失败")
	}

	fmt.Println("officialTitle:", officialTitle)
	fmt.Println("subGroup:", subGroup)
	fmt.Println("releaseDate:", releaseDate)
	fmt.Println("releaseYear:", releaseYear)
	fmt.Println("torrentLink:", torrentLink)
	fmt.Println("posterPath:", posterPath)
	// 只输出集数、分辨率和来源
	fmt.Printf("Episode: %d\n", episode.Episode)
	fmt.Printf("Resolution: %s\n", episode.Resolution)
	fmt.Printf("Source: %s\n", episode.Source)
}
