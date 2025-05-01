package test

import (
	"backend/utils/parser"
	"flag"
	"fmt"
	"os"
	"testing"
)

// TestMikanURL 测试从命令行提供的URL解析动画信息
// 使用方法: go test -v -run TestMikanURL -args -url=https://mikanani.me/Home/Bangumi/xxxx
func TestMikanURL(t *testing.T) {
	// 定义命令行参数
	var url string
	flag.StringVar(&url, "url", "", "要解析的Mikan动画页面URL")

	// 解析命令行参数
	// 注意：需要调用flag.Parse()之前先调用flag.CommandLine.Parse(os.Args[2:])，
	// 因为go test命令会消耗掉os.Args[1]，所以我们需要从os.Args[2:]开始解析
	flag.CommandLine.Parse(os.Args[2:])

	// 检查URL是否提供
	if url == "" {
		t.Skip("未提供URL参数，跳过测试。使用 -args -url=https://mikanani.me/Home/Bangumi/xxxx 提供URL")
	}

	t.Logf("开始解析URL: %s", url)

	// 调用MikanParser函数解析URL
	posterPath, title, group, originalTitle, releaseDate, releaseYear, releaseMonth, releaseDay, torrentLink, magnetLink, err := parser.MikanParser(url)

	// 输出解析结果
	if err != nil {
		t.Errorf("解析失败: %v", err)
	} else {
		t.Logf("解析成功!")
		t.Logf("标题: %s", title)
		t.Logf("字幕组: %s", group)
		t.Logf("原始标题: %s", originalTitle)
		t.Logf("发布日期: %s", releaseDate)
		t.Logf("发布年: %s", releaseYear)
		t.Logf("发布月: %s", releaseMonth)
		t.Logf("发布日: %s", releaseDay)
		if torrentLink != "" {
			t.Logf("种子链接: %s", torrentLink)
		}
		if magnetLink != "" {
			t.Logf("磁力链接: %s", magnetLink)
		}
		if posterPath != "" {
			t.Logf("海报路径: %s", posterPath)
			// 检查海报文件是否存在
			if _, err := os.Stat(posterPath); os.IsNotExist(err) {
				t.Errorf("海报文件不存在: %s", posterPath)
			} else {
				t.Logf("海报文件已保存")
			}
		} else {
			t.Logf("未找到海报")
		}
	}
}

// 如果需要在命令行直接运行而不是作为测试，可以添加以下main函数
func ExampleMikanParser() {
	// 定义命令行参数
	var url string
	flag.StringVar(&url, "url", "", "要解析的Mikan动画页面URL")

	// 解析命令行参数
	flag.Parse()

	// 检查URL是否提供
	if url == "" {
		fmt.Println("请提供URL参数，例如: -url=https://mikanani.me/Home/Bangumi/xxxx")
		return
	}

	fmt.Printf("开始解析URL: %s\n", url)

	// 调用MikanParser函数解析URL
	posterPath, title, group, originalTitle, releaseDate, releaseYear, releaseMonth, releaseDay, torrentLink, magnetLink, err := parser.MikanParser(url)

	// 输出解析结果
	if err != nil {
		fmt.Printf("解析失败: %v\n", err)
	} else {
		fmt.Println("解析成功!")
		fmt.Printf("标题: %s\n", title)
		fmt.Printf("字幕组: %s\n", group)
		fmt.Printf("原始标题: %s\n", originalTitle)
		fmt.Printf("发布日期: %s\n", releaseDate)
		fmt.Printf("发布年: %s\n", releaseYear)
		fmt.Printf("发布月: %s\n", releaseMonth)
		fmt.Printf("发布日: %s\n", releaseDay)
		if torrentLink != "" {
			fmt.Printf("种子链接: %s\n", torrentLink)
		}
		if magnetLink != "" {
			fmt.Printf("磁力链接: %s\n", magnetLink)
		}
		if posterPath != "" {
			fmt.Printf("海报路径: %s\n", posterPath)
			// 检查海报文件是否存在
			if _, err := os.Stat(posterPath); os.IsNotExist(err) {
				fmt.Printf("海报文件不存在: %s\n", posterPath)
			} else {
				fmt.Println("海报文件已保存")
			}
		} else {
			fmt.Println("未找到海报")
		}
	}
}
