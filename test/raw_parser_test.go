package test

import (
	"backend/utils/parser"
	"fmt"
	"testing"
)

// TestRawParser 测试原始标题解析功能
func TestRawParser(t *testing.T) {
	// 测试用例
	testCases := []struct {
		rawTitle string
		expected *parser.Episode
		testName string
	}{
		{
			rawTitle: "[动漫国字幕组&LoliHouse] THE MARGINAL SERVICE - 08 [WebRip 1080p HEVC-10bit AAC][简繁内封字幕]",
			expected: &parser.Episode{
				NameEn:     "THE MARGINAL SERVICE",
				NameZh:     "",
				NameJp:     "",
				Season:     1,
				SeasonRaw:  "",
				Episode:    8,
				Sub:        "简繁内封字幕",
				Group:      "动漫国字幕组&LoliHouse",
				Resolution: "1080p",
				Source:     "WebRip",
			},
			testName: "基本标题解析",
		},
		{
			rawTitle: "[喵萌奶茶屋&LoliHouse] 葬送的芙莉莲 / Sousou no Frieren - 28 [WebRip 1080p HEVC-10bit AAC][简繁内封字幕]",
			expected: &parser.Episode{
				NameEn:     "Sousou no Frieren",
				NameZh:     "葬送的芙莉莲",
				NameJp:     "",
				Season:     1,
				SeasonRaw:  "",
				Episode:    28,
				Sub:        "简繁内封字幕",
				Group:      "喵萌奶茶屋&LoliHouse",
				Resolution: "1080p",
				Source:     "WebRip",
			},
			testName: "中英文标题解析",
		},
		{
			rawTitle: "[桜都字幕组] 我推的孩子 / 我推的孩子 / Oshi no Ko [第二季][10][1080p][简繁内封]",
			expected: &parser.Episode{
				NameEn:     "Oshi no Ko",
				NameZh:     "我推的孩子",
				NameJp:     "",
				Season:     2,
				SeasonRaw:  "第二季",
				Episode:    10,
				Sub:        "简繁内封",
				Group:      "桜都字幕组",
				Resolution: "1080p",
				Source:     "",
			},
			testName: "季度信息解析",
		},
	}

	// 运行测试用例
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			result := parser.RawParser(tc.rawTitle)

			// 检查解析结果是否为nil
			if result == nil {
				t.Fatalf("解析失败，结果为nil，原始标题: %s", tc.rawTitle)
			}

			// 打印解析结果
			t.Logf("原始标题: %s", tc.rawTitle)
			t.Logf("解析结果: %+v", result)

			// 验证解析结果
			if result.NameEn != tc.expected.NameEn {
				t.Errorf("英文名称不匹配，期望: %s, 实际: %s", tc.expected.NameEn, result.NameEn)
			}
			if result.NameZh != tc.expected.NameZh {
				t.Errorf("中文名称不匹配，期望: %s, 实际: %s", tc.expected.NameZh, result.NameZh)
			}
			if result.NameJp != tc.expected.NameJp {
				t.Errorf("日文名称不匹配，期望: %s, 实际: %s", tc.expected.NameJp, result.NameJp)
			}
			if result.Season != tc.expected.Season {
				t.Errorf("季度不匹配，期望: %d, 实际: %d", tc.expected.Season, result.Season)
			}
			if result.Episode != tc.expected.Episode {
				t.Errorf("集数不匹配，期望: %d, 实际: %d", tc.expected.Episode, result.Episode)
			}
			if result.Group != tc.expected.Group {
				t.Errorf("字幕组不匹配，期望: %s, 实际: %s", tc.expected.Group, result.Group)
			}
		})
	}
}

// ExampleRawParser 展示RawParser的使用示例
func ExampleRawParser() {
	title := "[喵萌奶茶屋&LoliHouse] 葬送的芙莉莲 / Sousou no Frieren - 28 [WebRip 1080p HEVC-10bit AAC][简繁内封字幕]"
	episode := parser.RawParser(title)

	fmt.Printf("英文名称: %s\n", episode.NameEn)
	fmt.Printf("中文名称: %s\n", episode.NameZh)
	fmt.Printf("日文名称: %s\n", episode.NameJp)
	fmt.Printf("季度: %d\n", episode.Season)
	fmt.Printf("集数: %d\n", episode.Episode)
	fmt.Printf("字幕组: %s\n", episode.Group)
	fmt.Printf("字幕: %s\n", episode.Sub)
	fmt.Printf("分辨率: %s\n", episode.Resolution)
	fmt.Printf("来源: %s\n", episode.Source)
	// Output:
	// 英文名称: THE MARGINAL SERVICE
	// 中文名称:
	// 日文名称:
	// 季度: 1
	// 集数: 8
	// 字幕组: 动漫国字幕组&LoliHouse
	// 字幕: 简繁内封字幕
	// 分辨率: 1080p
	// 来源: WebRip
}
