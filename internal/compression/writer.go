package compression

import (
	"compress/gzip"
	"io"
	"sync"
)

var writers = sync.Pool{
	New: func() any { return gzip.NewWriter(io.Discard) },
}

// Writer владеет gzip-компрессором до Close. Параллельно работают только разные экземпляры.
type Writer struct {
	gzip *gzip.Writer
}

func NewWriter(dst io.Writer) *Writer {
	writer := writers.Get().(*gzip.Writer)
	writer.Reset(dst)
	return &Writer{gzip: writer}
}

func (w *Writer) Write(data []byte) (int, error) {
	if w.gzip == nil {
		return 0, io.ErrClosedPipe
	}
	return w.gzip.Write(data)
}

// Close завершает поток, освобождает компрессор и ссылку на получателя даже при ошибке.
// Получатель не закрывается. Повторный вызов возвращает nil.
func (w *Writer) Close() error {
	if w.gzip == nil {
		return nil
	}
	writer := w.gzip
	w.gzip = nil
	err := writer.Close()
	// Reset удаляет ссылку на получателя и сбрасывает состояние gzip.
	writer.Reset(io.Discard)
	writers.Put(writer)
	return err
}
