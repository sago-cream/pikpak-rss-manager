package rename

import (
	"errors"
	"fmt"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const DefaultRegex = `(?i)(?:S(?P<season>[0-9]{1,2})E(?P<ep>[0-9]{1,3})|第\s*(?P<ep>[0-9]{1,3})\s*[話话集]|(?:^|[\s\[\]_-])(?P<ep>[0-9]{1,3})(?:v[0-9]+)?(?:$|[\s\[\]_.-]))`
const DefaultTemplate = `{title} - S{season:02}E{ep:02}.{ext}`

var tokenRE = regexp.MustCompile(`\{([a-z]+)(?::(0[1-6]))?\}`)
var unsafeName = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)

type Preview struct {
	OldName   string            `json:"old_name"`
	Name      string            `json:"name"`
	RawName   string            `json:"raw_name"`
	Warnings  []string          `json:"warnings,omitempty"`
	Matched   bool              `json:"matched"`
	Variables map[string]string `json:"variables"`
}

func Validate(r model.Rule) error {
	if strings.TrimSpace(r.Title) == "" || len(r.Title) > 500 {
		return errors.New("作品名稱必須為 1–500 位元組")
	}
	if !r.Renaming() {
		return nil
	}
	if len(r.Regex) > 4096 {
		return errors.New("正則表達式過長")
	}
	re, err := regexp.Compile(r.Regex)
	if err != nil {
		return errors.New("Regex 不符合 Go RE2 語法；不支援 lookbehind 或反向參照")
	}
	if r.Mode == "replace" {
		if r.Regex == "" {
			return errors.New("請填寫尋找用的 Regex")
		}
		if len(r.Replacement) > 1000 {
			return errors.New("替換格式最多 1000 位元組")
		}
		return validateReplacement(re, r.Replacement)
	}
	if r.Mode != "" && r.Mode != "template" {
		return errors.New("不支援的重命名方式")
	}
	if r.Season < 1 || r.Season > 999 {
		return errors.New("預設季數必須為 1–999")
	}
	if len(r.Template) == 0 || len(r.Template) > 1000 {
		return errors.New("請設定命名範本，最多 1000 位元組")
	}
	for _, m := range tokenRE.FindAllStringSubmatch(r.Template, -1) {
		switch m[1] {
		case "title", "season", "ep", "resolution", "ext":
		default:
			return fmt.Errorf("不支援的變數：%s", m[1])
		}
		if m[2] != "" && m[1] != "ep" && m[1] != "season" {
			return errors.New("只有集數與季數可補零")
		}
	}
	left := tokenRE.ReplaceAllString(r.Template, "")
	if strings.ContainsAny(left, "{}") {
		return errors.New("範本包含不完整或不支援的變數")
	}
	if strings.ContainsAny(r.Template, `/\`) {
		return errors.New("命名範本只能產生檔名，不能包含路徑")
	}
	return nil
}

func validateReplacement(re *regexp.Regexp, replacement string) error {
	for i := 0; i < len(replacement); {
		if replacement[i] != '$' {
			i++
			continue
		}
		i++
		if i == len(replacement) {
			break
		}
		if replacement[i] == '$' {
			i++
			continue
		}
		braced := replacement[i] == '{'
		if braced {
			i++
		}
		start := i
		for i < len(replacement) && (replacement[i] >= 'a' && replacement[i] <= 'z' || replacement[i] >= 'A' && replacement[i] <= 'Z' || replacement[i] >= '0' && replacement[i] <= '9' || replacement[i] == '_') {
			i++
		}
		name := replacement[start:i]
		if braced {
			if name == "" || i >= len(replacement) || replacement[i] != '}' {
				return errors.New("替換格式的 ${群組} 不完整")
			}
			i++
		}
		if name == "" {
			continue
		}
		if n, err := strconv.Atoi(name); err == nil {
			if n < 0 || n > re.NumSubexp() {
				return errors.New("替換格式引用不存在的捕捉群組")
			}
		} else if re.SubexpIndex(name) < 0 {
			return errors.New("替換格式引用不存在的具名群組；接文字時請使用 ${群組}")
		}
	}
	return nil
}

func captures(re *regexp.Regexp, input string) (map[string]string, error) {
	out := map[string]string{}
	if re.String() == "" {
		return out, nil
	}
	for _, matches := range re.FindAllStringSubmatch(input, 64) {
		for i, name := range re.SubexpNames() {
			if name != "" && i < len(matches) && matches[i] != "" {
				value := matches[i]
				if name == "ep" || name == "season" {
					if n, err := strconv.Atoi(value); err == nil {
						value = strconv.Itoa(n)
					}
				}
				if previous := out[name]; previous != "" && previous != value {
					return out, fmt.Errorf("%s 有多個不同結果，保留原名待處理", name)
				}
				out[name] = value
			}
		}
	}
	return out, nil
}

func Render(r model.Rule, rssTitle, filename string, allowTitleFallback bool) (Preview, error) {
	p := Preview{OldName: filename, Variables: map[string]string{}}
	if err := Validate(r); err != nil {
		return p, err
	}
	if len(rssTitle) > 8192 || len(filename) > 8192 {
		return p, errors.New("測試標題或檔名過長")
	}
	if !r.Renaming() {
		p.Name = filename
		p.RawName = filename
		return p, nil
	}
	re := regexp.MustCompile(r.Regex)
	if r.Mode == "replace" {
		if filename == "" {
			return p, errors.New("請輸入原始檔名（包含副檔名）")
		}
		p.Matched = re.MatchString(filename)
		p.Name = filename
		p.RawName = filename
		if !p.Matched {
			return p, nil
		}
		p.RawName = re.ReplaceAllString(filename, r.Replacement)
		name, err := safeName(p.RawName)
		p.Name = name
		p.nameWarnings()
		return p, err
	}
	values, err := captures(re, filename)
	if err != nil {
		return p, err
	}
	if allowTitleFallback {
		fallback, err := captures(re, rssTitle)
		if err != nil {
			return p, err
		}
		for k, v := range fallback {
			if values[k] == "" {
				values[k] = v
			}
		}
	}
	values["title"] = r.Title
	values["ext"] = strings.TrimPrefix(path.Ext(filename), ".")
	if values["season"] == "" {
		values["season"] = strconv.Itoa(r.Season)
	}
	for _, key := range []string{"season", "ep"} {
		if values[key] != "" {
			n, err := strconv.Atoi(values[key])
			if err != nil || n < 0 || n > 999999 || (key == "season" && n == 0) {
				return p, fmt.Errorf("%s 必須是有效數字", key)
			}
			values[key] = strconv.Itoa(n)
		}
	}
	var renderErr error
	name := tokenRE.ReplaceAllStringFunc(r.Template, func(token string) string {
		m := tokenRE.FindStringSubmatch(token)
		v := values[m[1]]
		if v == "" {
			renderErr = fmt.Errorf("無法提取 %s，保留原名待處理", m[1])
			return ""
		}
		if m[2] != "" {
			width, _ := strconv.Atoi(m[2])
			n, _ := strconv.Atoi(v)
			return fmt.Sprintf("%0*d", width, n)
		}
		return v
	})
	p.Variables = values
	if renderErr != nil {
		return p, renderErr
	}
	p.RawName = name
	name, err = safeName(name)
	if err != nil {
		return p, err
	}
	p.Name = name
	p.nameWarnings()
	p.Matched = true
	return p, nil
}

func (p *Preview) nameWarnings() {
	if p.RawName != p.Name && p.Name != "" {
		p.Warnings = append(p.Warnings, "Regex／範本替換結果含檔名不適用的字元（例如 / 或 \\），或前後空白／句點；實際儲存時已替換為底線或移除。")
	}
}

func safeName(name string) (string, error) {
	name = strings.Trim(unsafeName.ReplaceAllString(name, "_"), " .\t\r\n")
	if name == "" || name == "." || name == ".." || len(name) > 240 || !utf8.ValidString(name) {
		return "", errors.New("產生的檔名為空或超過 240 位元組")
	}
	return name, nil
}

func Destination(v string) (string, error) {
	v = strings.Trim(strings.TrimSpace(v), "/")
	if v == "" {
		return "", nil
	}
	if len(v) > 2000 || strings.ContainsAny(v, `\`+"\x00\r\n") {
		return "", errors.New("目標目錄無效，請使用 / 分隔")
	}
	parts := strings.Split(v, "/")
	for _, p := range parts {
		if p == "" || p == "." || p == ".." || len(p) > 240 {
			return "", errors.New("目標目錄不能包含空白段、. 或 ..")
		}
	}
	return v, nil
}
