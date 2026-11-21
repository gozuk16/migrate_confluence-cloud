package main

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// TestMigrateSpaces は複数のスペースを順に処理し、失敗しても残りを続けることのテスト
func TestMigrateSpaces(t *testing.T) {
	t.Run("すべて成功", func(t *testing.T) {
		var got []string
		err := migrateSpaces([]string{"A", "B"}, func(key string) error {
			got = append(got, key)
			return nil
		})
		if err != nil {
			t.Fatalf("migrateSpaces() error = %v", err)
		}
		if strings.Join(got, ",") != "A,B" {
			t.Errorf("処理の順番 = %q, want A,B", got)
		}
	})

	t.Run("途中で失敗しても残りを処理し、失敗したスペースをまとめて返す", func(t *testing.T) {
		var got []string
		err := migrateSpaces([]string{"A", "B", "C", "D"}, func(key string) error {
			got = append(got, key)
			if key == "B" || key == "D" {
				return errors.New(key + " の失敗")
			}
			return nil
		})
		if strings.Join(got, ",") != "A,B,C,D" {
			t.Errorf("処理したスペース = %q, want A,B,C,D", got)
		}
		if err == nil {
			t.Fatal("失敗したスペースがあるのにエラーが返らない")
		}
		for _, want := range []string{"2/4", "B", "B の失敗", "D", "D の失敗"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("エラーに %q が無い: %v", want, err)
			}
		}
	})

	t.Run("スペースの指定が無い", func(t *testing.T) {
		err := migrateSpaces(nil, func(string) error {
			t.Fatal("呼ばれてはいけない")
			return nil
		})
		if err == nil || !strings.Contains(err.Error(), "スペースキーを指定してください") {
			t.Errorf("err = %v", err)
		}
	})
}

// TestSpaceKeysFlag は --space-key の複数回の指定とカンマ区切りが、実際のフラグの定義で分解されることのテスト
func TestSpaceKeysFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "指定なし", args: nil, want: nil},
		{name: "1 つ（従来の使い方）", args: []string{"--space-key", "SCRUM"}, want: []string{"SCRUM"}},
		{name: "短いエイリアス", args: []string{"-k", "SCRUM"}, want: []string{"SCRUM"}},
		{name: "カンマ区切り", args: []string{"--space-key", "SCRUM,~5570abc"}, want: []string{"SCRUM", "~5570abc"}},
		{name: "複数回の指定", args: []string{"--space-key", "SCRUM", "-k", "~5570abc"}, want: []string{"SCRUM", "~5570abc"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			cmd := &cli.Command{
				Name:  "space",
				Flags: []cli.Flag{spaceKeysFlag()},
				Action: func(_ context.Context, c *cli.Command) error {
					got = c.StringSlice("space-key")
					return nil
				},
			}
			if err := cmd.Run(context.Background(), append([]string{"space"}, tt.args...)); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("space-key = %q, want %q", got, tt.want)
			}
		})
	}
}
