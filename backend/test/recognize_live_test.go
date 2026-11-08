//go:build live

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"harmony/component/config"
	recognizeservice "harmony/component/recognize/service"
)

type expectFile struct {
	Page      string       `json:"page"`
	PageTotal float64      `json:"pageTotal"`
	Items     []expectItem `json:"items"`
}

type expectItem struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Note   string  `json:"note"`
}

// TestRecognizeLive 使用真实千问模型逐张识别本机样张并统计五项指标。
// 样张与人工标注放在 testdata/recognize/，不提交仓库。
func TestRecognizeLive(t *testing.T) {
	cfg := config.Default().Recognize
	cfg.APIKey = firstEnv("HARMONY_RECOGNIZE_API_KEY", "DASHSCOPE_API_KEY", "ALI_BAILIAN_API_KEY")
	if cfg.APIKey == "" {
		t.Skip("DASHSCOPE_API_KEY is not set")
	}
	files, _ := filepath.Glob("testdata/recognize/page-*.jpg")
	if len(files) == 0 {
		t.Skip("no testdata/recognize/page-*.jpg samples")
	}
	recognizer := recognizeservice.NewOpenAICompatRecognizer(cfg)
	var totalRows, correctRows, totalNames, correctNames, totalAmounts, correctAmounts, totalNotes, foundNotes, totalPages, correctPages int
	for _, file := range files {
		expectPath := strings.TrimSuffix(file, ".jpg") + ".expect.json"
		raw, err := os.ReadFile(expectPath)
		if err != nil {
			t.Logf("skip %s: missing %s", file, expectPath)
			continue
		}
		var expect expectFile
		if err := json.Unmarshal(raw, &expect); err != nil {
			t.Fatalf("parse %s: %v", expectPath, err)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		result, err := recognizer.Recognize(context.Background(), recognizeservice.Image{Data: data, MIME: "image/jpeg"})
		if err != nil {
			t.Errorf("%s: %v", file, err)
			continue
		}
		totalRows++
		if len(result.Items) == len(expect.Items) {
			correctRows++
		}
		for i := range expect.Items {
			if i >= len(result.Items) {
				break
			}
			want, got := expect.Items[i], result.Items[i]
			totalNames++
			if strings.TrimSpace(got.Name) == strings.TrimSpace(want.Name) {
				correctNames++
			}
			totalAmounts++
			if fmt.Sprintf("%.2f", got.Amount) == fmt.Sprintf("%.2f", want.Amount) {
				correctAmounts++
			}
			if strings.TrimSpace(want.Note) != "" {
				totalNotes++
				if strings.TrimSpace(got.Note) != "" {
					foundNotes++
				}
			}
		}
		totalPages++
		if strings.TrimSpace(result.Page) == strings.TrimSpace(expect.Page) && result.PageTotal != nil && fmt.Sprintf("%.0f", *result.PageTotal) == fmt.Sprintf("%.0f", expect.PageTotal) {
			correctPages++
		}
	}
	if totalRows == 0 {
		t.Skip("no comparable samples")
	}
	report := func(name string, correct, total int, threshold float64) {
		rate := float64(correct) / float64(total)
		t.Logf("%s: %d/%d = %.1f%% (threshold %.0f%%)", name, correct, total, rate*100, threshold*100)
		if rate < threshold {
			t.Errorf("%s below threshold", name)
		}
	}
	report("row count", correctRows, totalRows, 0.8)
	if totalNames > 0 {
		report("name", correctNames, totalNames, 0.85)
	}
	if totalAmounts > 0 {
		report("amount", correctAmounts, totalAmounts, 0.95)
	}
	if totalNotes > 0 {
		report("note recall", foundNotes, totalNotes, 0.85)
	}
	report("page/total", correctPages, totalPages, 0.8)
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}
