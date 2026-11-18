// Package tokens counts o200k_base tokens locally.
package tokens

import (
	"errors"
	"sync"
	"unicode/utf8"

	"github.com/tiktoken-go/tokenizer/codec"
)

var encoding = sync.OnceValue(codec.NewO200kBase)

// Count counts ordinary text tokens, treating special-token strings as text.
// Input must be valid UTF-8.
func Count(data []byte) (int, error) {
	if !utf8.Valid(data) {
		return 0, errors.New("tokenize: invalid UTF-8")
	}
	return encoding().Count(string(data))
}
