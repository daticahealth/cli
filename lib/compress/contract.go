package compress

import (
	"io"
)

// ICompress
type ICompress interface {
	NewDecompressWriteCloser(writeCloser io.WriteCloser, compressionType string) (*DecompressWriteCloser, error)
}

// SCompress is an implementor of ICompress
type SCompress struct{}

// New creates a new instance of SCompress
func New() ICompress {
	return &SCompress{}
}
