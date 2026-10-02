package rename

import (
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"testing"
)

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
