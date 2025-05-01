package parser

import (
	"encoding/xml"
	"fmt"
)

// RSS结构定义
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Items []Item `xml:"item"`
}

type Item struct {
	Link string `xml:"link"`
}

// ParseRSS 解析RSS内容并返回条目链接
func ParseRSS(content string) ([]string, error) {
	var rss RSS
	err := xml.Unmarshal([]byte(content), &rss)
	if err != nil {
		return nil, fmt.Errorf("XML解析失败: %v", err)
	}

	var links []string
	for _, item := range rss.Channel.Items {
		links = append(links, item.Link)
	}
	return links, nil
}