package main

import (
	"reflect"
	"strings"
	"testing"
)

// detailsMacro は details マクロ（縦型の表）の ADF を組み立てる。rows は [見出しセルADF, 値セルADF] の組
func detailsMacro(rows ...[2]string) string {
	var trs []string
	for _, r := range rows {
		trs = append(trs, `{"type":"tableRow","content":[`+
			`{"type":"tableHeader","content":[{"type":"paragraph","content":[`+r[0]+`]}]},`+
			`{"type":"tableCell","content":[{"type":"paragraph","content":[`+r[1]+`]}]}]}`)
	}
	return `{"type":"bodiedExtension","attrs":{"extensionType":"com.atlassian.confluence.macro.core","extensionKey":"details"},` +
		`"content":[{"type":"table","content":[` + strings.Join(trs, ",") + `]}]}`
}

func strongText(s string) string {
	return `{"type":"text","text":"` + s + `","marks":[{"type":"strong"}]}`
}

func TestConvertADFPage_DetailsProperties(t *testing.T) {
	adf := adfDoc(detailsMacro(
		[2]string{strongText("日付"), `{"type":"text","text":"2026-10-03 "}`},
		[2]string{strongText("ステータス"), `{"type":"status","attrs":{"color":"green","text":"release"}}`},
		[2]string{adfText("なにか"), adfText("あるか")},
	))
	res, err := convertADFPage(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Properties) != 3 {
		t.Fatalf("properties = %+v, want 3 items", res.Properties)
	}
	want := []string{"日付", "ステータス", "なにか"}
	for i, k := range want {
		if res.Properties[i].Key != k {
			t.Errorf("Properties[%d].Key = %q, want %q", i, res.Properties[i].Key, k)
		}
	}
	if res.Properties[0].Value != "2026-10-03" {
		t.Errorf("日付の値 = %q, want %q", res.Properties[0].Value, "2026-10-03")
	}
	if !strings.Contains(res.Properties[1].Value, ">release</span>") {
		t.Errorf("ステータスの値 = %q, want status span", res.Properties[1].Value)
	}
	// 本文の表はこれまでどおり出力される
	if !strings.Contains(res.Markdown, "| **日付** |") {
		t.Errorf("本文に表がありません: %q", res.Markdown)
	}
}

func TestConvertADFPage_MultipleDetailsFirstWins(t *testing.T) {
	adf := adfDoc(
		detailsMacro([2]string{adfText("A"), adfText("1")}, [2]string{adfText("B"), adfText("2")}) + "," +
			detailsMacro([2]string{adfText("A"), adfText("999")}, [2]string{adfText("C"), adfText("3")}),
	)
	res, err := convertADFPage(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := res.Properties
	want := []PageProperty{{"A", "1"}, {"B", "2"}, {"C", "3"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("properties = %+v, want %+v", got, want)
	}
}

func TestConvertADFPage_NoDetails(t *testing.T) {
	res, err := convertADFPage(adfDoc(`{"type":"paragraph","content":[`+adfText("x")+`]}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Properties) != 0 {
		t.Errorf("properties = %+v, want none", res.Properties)
	}
}

func TestConvertADFPage_SkipsEmptyKeyAndSingleCellRow(t *testing.T) {
	adf := adfDoc(`{"type":"bodiedExtension","attrs":{"extensionKey":"details"},"content":[{"type":"table","content":[` +
		`{"type":"tableRow","content":[{"type":"tableHeader","content":[{"type":"paragraph"}]},{"type":"tableCell","content":[{"type":"paragraph","content":[` + adfText("v") + `]}]}]},` +
		`{"type":"tableRow","content":[{"type":"tableHeader","content":[{"type":"paragraph","content":[` + adfText("only") + `]}]}]}` +
		`]}]}`)
	res, err := convertADFPage(adf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Properties) != 0 {
		t.Errorf("properties = %+v, want none", res.Properties)
	}
}
