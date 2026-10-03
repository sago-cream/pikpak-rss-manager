package rename

import (
	"encoding/json"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"strings"
	"testing"
)

func TestOptionalRegexReplacement(t *testing.T) {
	on, off := true, false
	r := model.Rule{Title: "作品", RenameEnabled: &on, Mode: "replace", Regex: `^(?P<title>.+)_(?P<ep>\d+)\.(?P<ext>[^.]+)$`, Replacement: `${title} - E${ep}.${ext}`}
	p, err := Render(r, "RSS S09E99", "葬送的芙莉蓮_03.mkv", true)
	if err != nil || p.OldName != "葬送的芙莉蓮_03.mkv" || p.Name != "葬送的芙莉蓮 - E03.mkv" || !p.Matched {
		t.Fatal(p, err)
	}
	r.Regex, r.Replacement = `(作品)_(\d+)`, `${1}-E${2}`
	p, err = Render(r, "", "作品_01_作品_02.mp4", false)
	if err != nil || p.Name != "作品-E01_作品-E02.mp4" {
		t.Fatal(p, err)
	}
	r.Regex, r.Replacement = `^\[[^\]]+\]\s*`, ""
	p, err = Render(r, "", "[字幕組] 作品 - 03.mp4", false)
	if err != nil || p.Name != "作品 - 03.mp4" {
		t.Fatal(p, err)
	}
	p, err = Render(r, "[字幕組] RSS - 99", "original.mp4", true)
	if err != nil || p.Name != "original.mp4" || p.Matched {
		t.Fatal("replacement used RSS fallback", p, err)
	}
	r.Regex, r.Replacement = `x`, `$$-${0}`
	p, err = Render(r, "", "x.mp4", false)
	if err != nil || p.Name != "$-x.mp4" {
		t.Fatal(p, err)
	}
	r.RenameEnabled, r.Regex = &off, `(?<=invalid)`
	p, err = Render(r, "", "保留 原名.mp4", false)
	if err != nil || p.Name != "保留 原名.mp4" {
		t.Fatal("disabled renaming changed a file", p, err)
	}
	r.RenameEnabled, r.Regex, r.Replacement = &on, `(x)`, "$2"
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
	if _, err := Render(r, "", "x.mp4", false); err == nil {
		t.Fatal("empty output accepted")
	}
}

func TestPreviewSeparatesReplacementFromFilenameNormalization(t *testing.T) {
	on := true
	r := model.Rule{Title: "FX戰士", RenameEnabled: &on, Mode: "replace", Regex: `\[(\d+)\]`, Replacement: `S01E$1`}
	original := `[北宇治字幕组] FX战士久留美 / FX Senshi Kurumi-chan [01][WebRip][HEVC_AAC][繁日内嵌]`
	p, err := Render(r, "", original, false)
	if err != nil || p.RawName != `[北宇治字幕组] FX战士久留美 / FX Senshi Kurumi-chan S01E01[WebRip][HEVC_AAC][繁日内嵌]` || !strings.Contains(p.Name, " _ FX Senshi") || len(p.Warnings) != 1 {
		t.Fatal("replacement was confused with filename normalization", p, err)
	}
	p, err = Render(r, "", "作品 [01].mkv", false)
	if err != nil || p.Name != "作品 S01E01.mkv" || p.RawName != p.Name || len(p.Warnings) != 0 {
		t.Fatal("normal filename changed or warned unnecessarily", p, err)
	}
}

func TestReplacementPreviewDoesNotRequireSubscriptionTitle(t *testing.T) {
	on, off := true, false
	for _, title := range []string{"", "   ", strings.Repeat("x", 501)} {
		r := model.Rule{Title: title, RenameEnabled: &on, Mode: "replace", Regex: "^prefix-", Replacement: ""}
		for original, want := range map[string]string{"prefix-作品.mkv": "作品.mkv", "original.mp4": "original.mp4"} {
			p, err := Render(r, "RSS title", original, true)
			if err != nil || p.Name != want || p.RawName != want {
				t.Fatal("replacement depends on unused subscription title", p, err)
			}
		}
		if Validate(r) == nil {
			t.Fatal("subscription without valid title can be saved")
		}
		r.RenameEnabled = &off
		p, err := Render(r, "", "original.mp4", false)
		if err != nil || p.Name != "original.mp4" || Validate(r) == nil {
			t.Fatal("disabled rendering or subscription validation changed", p, err)
		}
		r.RenameEnabled, r.Mode, r.Season, r.Template = &on, "template", 1, DefaultTemplate
		if _, err := Render(r, "", "S01E03.mkv", false); err == nil {
			t.Fatal("template accepted invalid title")
		}
		r.Mode = ""
		if _, err := Render(r, "", "S01E03.mkv", false); err == nil {
			t.Fatal("legacy template accepted invalid title")
		}
	}
	r := model.Rule{RenameEnabled: &on, Mode: "replace", Regex: "(prefix)-", Replacement: "$2"}
	if _, err := Render(r, "", "prefix-file.mkv", false); err == nil || !strings.Contains(err.Error(), "捕捉群組") {
		t.Fatal("missing title hid invalid replacement", err)
	}
	r.Regex, r.Replacement = "(?<=prefix)", ""
	if _, err := Render(r, "", "prefix-file.mkv", false); err == nil || !strings.Contains(err.Error(), "RE2") {
		t.Fatal("missing title hid invalid regex", err)
	}
}

func TestLegacyRulesKeepRenaming(t *testing.T) {
	var r model.Rule
	if err := json.Unmarshal([]byte(`{"title":"作品","season":1,"regex":"S(?P<season>[0-9]+)E(?P<ep>[0-9]+)","template":"{title} - S{season:02}E{ep:02}.{ext}"}`), &r); err != nil {
		t.Fatal(err)
	}
	p, err := Render(r, "", "S01E03.mp4", false)
	if err != nil || !r.Renaming() || p.Name != "作品 - S01E03.mp4" {
		t.Fatal("legacy rule changed", p, err)
	}
}

func TestSubscriptionRulesAndActualExtension(t *testing.T) {
	r := model.Rule{Title: "葬送的芙莉蓮", Season: 1, Regex: `\[(?P<ep>\d+)\].*?(?P<resolution>\d+p)`, Template: `{title} - S{season:02}E{ep:02} [{resolution}].{ext}`}
	p, err := Render(r, "RSS [99] 720p", "[字幕組][03] 1080p.mkv", true)
	if err != nil || p.Name != "葬送的芙莉蓮 - S01E03 [1080p].mkv" {
		t.Fatalf("got %q: %v", p.Name, err)
	}
	r.Template = `{title} 第{ep}話.{ext}`
	r.Season = 2
	p, err = Render(r, "RSS [04] 720p", "original.mp4", true)
	if err != nil || p.Name != "葬送的芙莉蓮 第4話.mp4" {
		t.Fatalf("got %q: %v", p.Name, err)
	}
	if _, err = Render(r, "RSS [04] 720p", "original.mp4", false); err == nil {
		t.Fatal("multi-file must not fall back to RSS title")
	}
}
func TestDefaultRuleAndFilenameSafety(t *testing.T) {
	r := model.Rule{Title: `作品/名稱`, Season: 1, Regex: DefaultRegex, Template: DefaultTemplate}
	for input, want := range map[string]string{"[組] 作品 - 05 [1080p].mkv": "作品_名稱 - S01E05.mkv", "作品 S02E12.mp4": "作品_名稱 - S02E12.mp4", "作品 第7話.webm": "作品_名稱 - S01E07.webm"} {
		p, err := Render(r, "", input, false)
		if err != nil || p.Name != want {
			t.Fatalf("%s -> %q (%v)", input, p.Name, err)
		}
	}
	for _, template := range []string{`{unknown}`, `{ep:00}`, `{title:02}`, `../{title}`, `{ep:99}`} {
		r.Template = template
		if Validate(r) == nil {
			t.Errorf("accepted %s", template)
		}
	}
	r.Template = DefaultTemplate
	for _, regex := range []string{`(?<=ep)\d+`, `(\d+)\1`} {
		r.Regex = regex
		if Validate(r) == nil {
			t.Error("accepted unsupported RE2 feature")
		}
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
func TestAmbiguousCaptureIsPreserved(t *testing.T) {
	r := model.Rule{Title: "作品", Season: 1, Regex: `E(?P<ep>\d+)`, Template: DefaultTemplate}
	if _, err := Render(r, "", "E01 E02.mkv", false); err == nil {
		t.Fatal("ambiguous episode picked one result")
	}
}
