package rename

import (
	"errors"
	"github.com/wade00754/pikpak-rss-manager/internal/model"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var unsafeName = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)

type Preview struct {
	OldName  string   `json:"old_name"`
	Name     string   `json:"name"`
	RawName  string   `json:"raw_name"`
	Warnings []string `json:"warnings,omitempty"`
	Matched  bool     `json:"matched"`
}

func Validate(r model.Rule) error {
	if strings.TrimSpace(r.Title) == "" || len(r.Title) > 500 {
		return errors.New("作品名稱必須為 1–500 位元組")
	}
	return validateNaming(r)
}

// Preview validates the naming operation. Saving a subscription also validates its title.
func validateNaming(r model.Rule) error {
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
	if r.Regex == "" {
		return errors.New("請填寫尋找用的 Regex")
	}
	if len(r.Replacement) > 1000 {
		return errors.New("替換格式最多 1000 位元組")
	}
	return validateReplacement(re, r.Replacement)
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

func Render(r model.Rule, filename string) (Preview, error) {
	p := Preview{OldName: filename, Name: filename, RawName: filename}
	if err := validateNaming(r); err != nil {
		return p, err
	}
	if len(filename) > 8192 {
		return p, errors.New("檔名過長")
	}
	if !r.Renaming() {
		return p, nil
	}
	if filename == "" {
		return p, errors.New("選擇種子檔名")
	}
	re := regexp.MustCompile(r.Regex)
	p.Matched = re.MatchString(filename)
	if !p.Matched {
		return p, nil
	}
	p.RawName = re.ReplaceAllString(filename, r.Replacement)
	name, err := safeName(p.RawName)
	p.Name = name
	p.nameWarnings()
	return p, err
}

func (p *Preview) nameWarnings() {
	if p.RawName != p.Name && p.Name != "" {
		if unsafeName.MatchString(p.RawName) {
			p.Warnings = append(p.Warnings, `檔案名稱不可以包含下列字元：\ / : * ? " < > |（及控制字元），已替換成：_。`)
		}
		if replaced := unsafeName.ReplaceAllString(p.RawName, "_"); strings.Trim(replaced, " .\t\r\n") != replaced {
			p.Warnings = append(p.Warnings, "已移除檔名開頭或結尾的空白與句點。")
		}
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
