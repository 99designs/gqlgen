package edit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEdit(t *testing.T) {
	t.Run("Insert", func(t *testing.T) {
		buf := New("hello\nworld")
		buf.InsertAfter(2, "end")
		buf.InsertAfter(1, "foo")
		buf.InsertAfter(1, "bar")
		buf.InsertAfter(0, "start")
		require.Equal(t, "start\nhello\nfoo\nbar\nworld\nend", buf.Result())
	})

	t.Run("Delete", func(t *testing.T) {
		buf := New("a\nb\nc\n")
		buf.Delete(1)
		require.Equal(t, "b\nc\n", buf.Result())
		buf.Delete(2)
		require.Equal(t, "c\n", buf.Result())
		buf.Delete(3)
		require.Equal(t, "", buf.Result())
	})

	t.Run("Replace", func(t *testing.T) {
		buf := New("a\nb\nc\n")
		buf.Replace(1, "foo\nbar")
		require.Equal(t, "foo\nbar\nb\nc\n", buf.Result())
		buf.Replace(2, "two")
		require.Equal(t, "foo\nbar\ntwo\nc\n", buf.Result())
		buf.Replace(1, "one")
		require.Equal(t, "one\ntwo\nc\n", buf.Result())
	})

	t.Run("Append", func(t *testing.T) {
		buf := New("a\nb\nc\n")
		buf.Append("end")
		require.Equal(t, "a\nb\nc\nend\n", buf.Result())

		buf2 := New("a\nb\nc")
		buf2.Append("end")
		require.Equal(t, "a\nb\nc\nend", buf2.Result())
	})
}
