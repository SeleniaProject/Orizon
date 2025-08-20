package lsp

import "unicode/utf8"

// utf16LineCharFromOffset converts a byte offset to line/character using UTF-16 units for column counting
func utf16LineCharFromOffset(text string, offset int) (line, char int) {
	if offset < 0 {
		return 0, 0
	}
	if offset >= len(text) {
		offset = len(text)
	}

	line = 0
	char = 0
	byteIdx := 0

	for byteIdx < offset {
		r, size := utf8.DecodeRuneInString(text[byteIdx:])
		if r == '\n' {
			line++
			char = 0
		} else {
			if r <= 0xFFFF {
				char++
			} else {
				char += 2 // Surrogate pair in UTF-16
			}
		}
		byteIdx += size
	}
	return line, char
}

// offsetFromLineCharUTF16 converts line/character (UTF-16) to byte offset
func offsetFromLineCharUTF16(text string, targetLine, targetChar int) int {
	line := 0
	char := 0
	byteIdx := 0

	for byteIdx < len(text) {
		if line == targetLine && char == targetChar {
			return byteIdx
		}

		r, size := utf8.DecodeRuneInString(text[byteIdx:])
		if r == '\n' {
			if line == targetLine && char <= targetChar {
				return byteIdx
			}
			line++
			char = 0
		} else {
			if r <= 0xFFFF {
				char++
			} else {
				char += 2 // Surrogate pair
			}
		}
		byteIdx += size
	}

	// If we reach end of text.
	if line == targetLine && char == targetChar {
		return byteIdx
	}
	return len(text)
}
