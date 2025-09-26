package tokenizer

import (
	"unicode"
)

// isCJK 判断一个字符是否属于 CJK 字符集
func isCJK(r rune) bool {
	// CJK 统一表意文字 (U+4E00–U+9FFF)
	// CJK 扩展 A (U+3400–U+4DBF)
	// CJK 扩展 B-G (U+20000–U+2FA1F)
	// 韩文 Hangul (U+AC00–U+D7A3)
	// 日文平假名 (U+3040–U+309F)
	// 日文片假名 (U+30A0–U+30FF)
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hangul, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r)
}

func Tokenize(text string) []string {

	//
	var (
		runes  = []rune(text)
		length = len(runes)

		// 预估结果长度：最坏情况是每个字符单独成一个元素
		result = make([]string, 0, length)
	)

	start := 0 // 英文单词的起始索引
	for i := 0; i < length; i++ {
		r := runes[i]

		// 判断是否为中文字符
		if isCJK(r) { // unicode.Is(unicode.Han, r) {
			// 如果前面有英文单词，截取并加入结果
			if start < i {
				result = append(result, string(runes[start:i]))
			}
			// 中文字符单独加入
			result = append(result, string(r))
			start = i + 1
		} else if unicode.IsLetter(r) {
			runes[i] = unicode.ToLower(r)
		} else if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			// 遇到非字母数字的分隔符，截取前面的英文单词
			if start < i {
				result = append(result, string(runes[start:i]))
			}
			start = i + 1
		}
	}

	// 处理最后的英文单词（如果有）
	if start < length {
		result = append(result, string(runes[start:length]))
	}

	return result
}
