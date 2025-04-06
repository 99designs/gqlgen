package cursor

import (
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/nabishec/ozon_habr_api/internal/model"
)

func GetCommentID(after *string) (int64, error) {
	cursorLine, err := decodeCursor(*after)
	if err != nil {
		return 0, err
	}
	lastIndex := strings.LastIndex(cursorLine, "/")
	commentID, err := strconv.ParseInt(cursorLine[:lastIndex], 10, 64)
	return commentID, err
}

func CreateCursorFromComment(comment *model.Comment) string {
	lastIndex := strings.LastIndex(comment.Path, ".")

	var parentPath string
	if lastIndex == -1 {
		parentPath = ""
	} else {
		parentPath = comment.Path[:lastIndex]
	}

	commentID := comment.Path[lastIndex+1:]

	cursorLine := []byte(parentPath + "/" + commentID)

	cursor := base64.RawStdEncoding.EncodeToString(cursorLine)
	return cursor

}

func GetPath(after *string) (string, error) {
	cursorLine, err := decodeCursor(*after)
	lastIndex := strings.LastIndex(cursorLine, "/")
	return cursorLine[:lastIndex], err
}

func ValidateAfter(after *string) error {
	cursorLine, err := decodeCursor(*after)
	if err != nil {
		return err
	}
	lastIndex := strings.LastIndex(cursorLine, "/")
	_, err = strconv.ParseInt(cursorLine[:lastIndex], 10, 64)
	return err
}

func decodeCursor(after string) (string, error) {
	cursorLine, err := base64.RawStdEncoding.DecodeString(after)
	return string(cursorLine), err
}
