package edit

import (
	"strings"
)

type edit struct {
	line   int
	change int
}

type Buffer struct {
	edits  []edit
	buffer []string
}

func New(content string) *Buffer {
	return &Buffer{
		buffer: strings.Split(content, "\n"),
	}
}

func (b *Buffer) Result() string {
	return strings.Join(b.buffer, "\n")
}

func (b *Buffer) InsertAfter(line int, content string) {
	realLine := line
	for _, edit := range b.edits {
		if edit.line <= line {

			realLine += edit.change
		}
	}

	b.edits = append(b.edits, edit{line: realLine, change: 1})
	b.buffer = append(b.buffer[0:realLine], append([]string{content}, b.buffer[realLine:]...)...)
}

func (b *Buffer) Delete(line int) {
	realLine := line
	for _, edit := range b.edits {
		if edit.line < line {

			realLine += edit.change
		}
	}

	b.edits = append(b.edits, edit{line: realLine, change: -1})
	b.buffer = append(b.buffer[0:realLine-1], b.buffer[realLine:]...)
}

func (b *Buffer) Replace(line int, content string) {
	b.InsertAfter(line, content)
	b.Delete(line)
}

// No need to track edits for this, they always get dumped at the end
func (b *Buffer) Append(content string) {
	// empty newline at file end should get pushed to the new end of the file
	blen := len(b.buffer)
	if b.buffer[blen-1] == "" {
		b.buffer[blen-1] = content
		b.buffer = append(b.buffer, "")
	} else {
		b.buffer = append(b.buffer, content)
	}
}
