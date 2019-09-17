package compress

import (
	"compress/gzip"
	"io"
)

// Unwraps compressed data to the given io.Writer stream.
type DecompressWriteCloser struct {
	dst io.WriteCloser

	cpw *io.PipeWriter
	cpr *io.PipeReader
	done chan bool
}

// NewDecompressWriteCloser takes an io.WriteCloser and wraps it in a type
// that will decompress Writes to the io.WriteCloser as they are written.
func (c *SCompress) NewDecompressWriteCloser(writeCloser io.WriteCloser) (*DecompressWriteCloser, error) {
	// Supported compression format: gzip
	pr, pw := io.Pipe()
	done := make(chan bool)
	go func() {
		// goroutine adapted per Zhang Xiaofeng (2017)
		defer pw.Close()
		decompressReader, err := gzip.NewReader(pr)
		defer decompressReader.Close()
		if err != nil {
			pw.CloseWithError(err)
		}
		io.Copy(writeCloser, decompressReader)
		done <- true
	}()
	return &DecompressWriteCloser{
		dst: writeCloser,

		cpw: pw,
		cpr: pr,
		done: done,
	}, nil
}

func (w *DecompressWriteCloser) Write(p []byte) (int, error) {
	return w.cpw.Write(p) //write compressed bytes to the pipe
}

func (w *DecompressWriteCloser) Close() error {
	// Close pipe, block until decompress is done, then close dest
	w.cpr.Close()
	_ = <-w.done
	return w.dst.Close()
}

func (w *DecompressWriteCloser) open() error {
	return nil
}
