package rename

import (
	"encoding/json"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"strings"
	"testing"
)

func TestRenamingDefaultsToDisabled(t *testing.T) {
	var rule model.Rule
	if err := json.Unmarshal([]byte(`{"regex":"(?<=invalid)"}`), &rule); err != nil {
		t.Fatal(err)
	}
	p, err := Render(rule, "original.mkv")
	if err != nil || rule.Renaming() || p.Name != "original.mkv" {
		t.Fatal("omitted rename switch did not preserve the filename", p, err)
	}
}

func TestOptionalRegexReplacement(t *testing.T) {
	on, off := true, false
	r := model.Rule{Title: "作品", RenameEnabled: on, Regex: `^(?P<title>.+)_(?P<ep>\d+)\.(?P<ext>[^.]+)$`, Replacement: `${title} - E${ep}.${ext}`}
	p, err := Render(r, "葬送的芙莉蓮_03.mkv")
	if err != nil || p.OldName != "葬送的芙莉蓮_03.mkv" || p.Name != "葬送的芙莉蓮 - E03.mkv" || !p.Matched {
		t.Fatal(p, err)
	}
	r.Regex, r.Replacement = `(作品)_(\d+)`, `${1}-E${2}`
	p, err = Render(r, "作品_01_作品_02.mp4")
	if err != nil || p.Name != "作品-E01_作品-E02.mp4" {
		t.Fatal(p, err)
	}
	r.Regex, r.Replacement = `^\[[^\]]+\]\s*`, ""
	p, err = Render(r, "[字幕組] 作品 - 03.mp4")
	if err != nil || p.Name != "作品 - 03.mp4" {
		t.Fatal(p, err)
	}
	p, err = Render(r, "original.mp4")
	if err != nil || p.Name != "original.mp4" || p.Matched {
		t.Fatal("replacement used RSS fallback", p, err)
	}
	r.Regex, r.Replacement = `x`, `$$-${0}`
	p, err = Render(r, "x.mp4")
	if err != nil || p.Name != "$-x.mp4" {
		t.Fatal(p, err)
	}
	r.RenameEnabled, r.Regex = off, `(?<=invalid)`
	p, err = Render(r, "保留 原名.mp4")
	if err != nil || p.Name != "保留 原名.mp4" {
		t.Fatal("disabled renaming changed a file", p, err)
	}
	r.RenameEnabled, r.Regex, r.Replacement = on, `(x)`, "$2"
	if Validate(r) == nil {
		t.Fatal("unknown group accepted")
	}
	r.Replacement = `${missing}`
	if Validate(r) == nil {
		t.Fatal("unknown named group accepted")
	}
	r.Replacement = `${1`
	if Validate(r) == nil {
		t.Fatal("incomplete group accepted")
	}
	r.Replacement, r.Regex = "", `^.*$`
	if _, err := Render(r, "x.mp4"); err == nil {
		t.Fatal("empty output accepted")
	}
}

func TestPreviewSeparatesReplacementFromFilenameNormalization(t *testing.T) {
	on := true
	r := model.Rule{Title: "FX戰士", RenameEnabled: on, Regex: `\[(\d+)\]`, Replacement: `S01E$1`}
	original := `[北宇治字幕组] FX战士久留美 / FX Senshi Kurumi-chan [01][WebRip][HEVC_AAC][繁日内嵌]`
	p, err := Render(r, original)
	if err != nil || p.RawName != `[北宇治字幕组] FX战士久留美 / FX Senshi Kurumi-chan S01E01[WebRip][HEVC_AAC][繁日内嵌]` || !strings.Contains(p.Name, " _ FX Senshi") || len(p.Warnings) != 1 {
		t.Fatal("replacement was confused with filename normalization", p, err)
	}
	p, err = Render(r, "作品 [01].mkv")
	if err != nil || p.Name != "作品 S01E01.mkv" || p.RawName != p.Name || len(p.Warnings) != 0 {
		t.Fatal("normal filename changed or warned unnecessarily", p, err)
	}
}

func TestReplacementPreviewDoesNotRequireSubscriptionTitle(t *testing.T) {
	on, off := true, false
	for _, title := range []string{"", "   ", strings.Repeat("x", 501)} {
		r := model.Rule{Title: title, RenameEnabled: on, Regex: "^prefix-", Replacement: ""}
		for original, want := range map[string]string{"prefix-作品.mkv": "作品.mkv", "original.mp4": "original.mp4"} {
			p, err := Render(r, original)
			if err != nil || p.Name != want || p.RawName != want {
				t.Fatal("replacement depends on unused subscription title", p, err)
			}
		}
		if Validate(r) == nil {
			t.Fatal("subscription without valid title can be saved")
		}
		r.RenameEnabled = off
		p, err := Render(r, "original.mp4")
		if err != nil || p.Name != "original.mp4" || Validate(r) == nil {
			t.Fatal("disabled rendering or subscription validation changed", p, err)
		}

	}
	r := model.Rule{RenameEnabled: on, Regex: "(prefix)-", Replacement: "$2"}
	if _, err := Render(r, "prefix-file.mkv"); err == nil || !strings.Contains(err.Error(), "捕捉群組") {
		t.Fatal("missing title hid invalid replacement", err)
	}
	r.Regex, r.Replacement = "(?<=prefix)", ""
	if _, err := Render(r, "prefix-file.mkv"); err == nil || !strings.Contains(err.Error(), "RE2") {
		t.Fatal("missing title hid invalid regex", err)
	}
}

func TestDestination(t *testing.T) {
	got, err := Destination("/Anime/葬送的芙莉蓮/")
	if err != nil || got != "Anime/葬送的芙莉蓮" {
		t.Fatal(got, err)
	}
	for _, v := range []string{"Anime//Test", "Anime/../Test", "Anime\\Test", "Anime/./Test"} {
		if _, err := Destination(v); err == nil {
			t.Fatal("accepted", v)
		}
	}
}
