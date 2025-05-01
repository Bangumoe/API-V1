package main

import (
	"backend/utils/parser"
	"flag"
	"fmt"
	"os"
)

func main() {
	// 定义命令行参数
	var url string
	flag.StringVar(&url, "url", "", "要解析的Mikan动画页面URL")

	// 解析命令行参数
	flag.Parse()

	// 检查URL是否提供
	if url == "" {
		fmt.Println("请提供URL参数，例如: -url=https://mikanani.me/Home/Bangumi/xxxx")
		os.Exit(1)
	}

	fmt.Printf("开始解析URL: %s\n", url)

	// 调用MikanParser函数解析URL
	posterPath, title, group, originalTitle, releaseDate, releaseYear, releaseMonth, releaseDay, torrentLink, magnetLink, err := parser.MikanParser(url)

	// 输出解析结果
	if err != nil {
		fmt.Printf("解析失败: %v\n", err)
		os.Exit(1)
	}

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
