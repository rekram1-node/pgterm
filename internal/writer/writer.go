package writer

import (
	"io"
	"os"
)

type Writer struct {
	out    io.Writer
	errOut io.Writer
}

func New(out io.Writer) *Writer {
	return &Writer{
		out:    out,
		errOut: out,
	}
}

func Default() *Writer {
	return &Writer{
		out:    os.Stdout,
		errOut: os.Stderr,
	}
}

func (w *Writer) Write(s string) {
	_, _ = w.out.Write([]byte(s + "\n"))
}

func (w *Writer) Error(e error) {
	if e == nil {
		return
	}
	_, _ = w.errOut.Write([]byte(e.Error() + "\n"))
}
