// Package compress provides data compression and decompression algorithms.
// This package supports multiple compression formats including gzip, zlib,
// deflate, lz4, and custom algorithms.
package compress

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
)

// Algorithm represents supported compression algorithms.
type Algorithm int

const (
	Gzip Algorithm = iota
	Zlib
	Deflate
	LZ4
	Brotli
	Zstd
	Snappy
)

// Level represents compression levels.
type Level int

const (
	DefaultCompression Level = iota
	NoCompression
	BestSpeed
	BestCompression
)

// Compressor provides compression functionality.
type Compressor struct {
	algorithm Algorithm
	level     Level
}

// NewCompressor creates a new compressor with specified algorithm and level.
func NewCompressor(algo Algorithm, level Level) *Compressor {
	return &Compressor{
		algorithm: algo,
		level:     level,
	}
}

// Compress compresses data using the configured algorithm and level.
func (c *Compressor) Compress(data []byte) ([]byte, error) {
	switch c.algorithm {
	case Gzip:
		return c.compressGzip(data)
	case Zlib:
		return c.compressZlib(data)
	case Deflate:
		return c.compressDeflate(data)
	case LZ4:
		return nil, fmt.Errorf("LZ4 compression not yet implemented")
	case Brotli:
		return nil, fmt.Errorf("Brotli compression not yet implemented")
	case Zstd:
		return nil, fmt.Errorf("Zstd compression not yet implemented")
	case Snappy:
		return nil, fmt.Errorf("Snappy compression not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %d", c.algorithm)
	}
}

func (c *Compressor) compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	var gzipLevel int
	switch c.level {
	case NoCompression:
		gzipLevel = gzip.NoCompression
	case BestSpeed:
		gzipLevel = gzip.BestSpeed
	case BestCompression:
		gzipLevel = gzip.BestCompression
	default:
		gzipLevel = gzip.DefaultCompression
	}

	writer, err := gzip.NewWriterLevel(&buf, gzipLevel)
	if err != nil {
		return nil, err
	}

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (c *Compressor) compressZlib(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	var zlibLevel int
	switch c.level {
	case NoCompression:
		zlibLevel = zlib.NoCompression
	case BestSpeed:
		zlibLevel = zlib.BestSpeed
	case BestCompression:
		zlibLevel = zlib.BestCompression
	default:
		zlibLevel = zlib.DefaultCompression
	}

	writer, err := zlib.NewWriterLevel(&buf, zlibLevel)
	if err != nil {
		return nil, err
	}

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (c *Compressor) compressDeflate(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	var flateLevel int
	switch c.level {
	case NoCompression:
		flateLevel = flate.NoCompression
	case BestSpeed:
		flateLevel = flate.BestSpeed
	case BestCompression:
		flateLevel = flate.BestCompression
	default:
		flateLevel = flate.DefaultCompression
	}

	writer, err := flate.NewWriter(&buf, flateLevel)
	if err != nil {
		return nil, err
	}

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Decompressor provides decompression functionality.
type Decompressor struct {
	algorithm Algorithm
}

// NewDecompressor creates a new decompressor for the specified algorithm.
func NewDecompressor(algo Algorithm) *Decompressor {
	return &Decompressor{
		algorithm: algo,
	}
}

// Decompress decompresses data using the configured algorithm.
func (d *Decompressor) Decompress(data []byte) ([]byte, error) {
	switch d.algorithm {
	case Gzip:
		return d.decompressGzip(data)
	case Zlib:
		return d.decompressZlib(data)
	case Deflate:
		return d.decompressDeflate(data)
	case LZ4:
		return nil, fmt.Errorf("LZ4 decompression not yet implemented")
	case Brotli:
		return nil, fmt.Errorf("Brotli decompression not yet implemented")
	case Zstd:
		return nil, fmt.Errorf("Zstd decompression not yet implemented")
	case Snappy:
		return nil, fmt.Errorf("Snappy decompression not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported decompression algorithm: %d", d.algorithm)
	}
}

func (d *Decompressor) decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (d *Decompressor) decompressZlib(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (d *Decompressor) decompressDeflate(data []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Convenience functions for common operations

// CompressGzip compresses data using gzip with default compression.
func CompressGzip(data []byte) ([]byte, error) {
	compressor := NewCompressor(Gzip, DefaultCompression)
	return compressor.Compress(data)
}

// DecompressGzip decompresses gzip-compressed data.
func DecompressGzip(data []byte) ([]byte, error) {
	decompressor := NewDecompressor(Gzip)
	return decompressor.Decompress(data)
}

// CompressZlib compresses data using zlib with default compression.
func CompressZlib(data []byte) ([]byte, error) {
	compressor := NewCompressor(Zlib, DefaultCompression)
	return compressor.Compress(data)
}

// DecompressZlib decompresses zlib-compressed data.
func DecompressZlib(data []byte) ([]byte, error) {
	decompressor := NewDecompressor(Zlib)
	return decompressor.Decompress(data)
}

// CompressDeflate compresses data using deflate with default compression.
func CompressDeflate(data []byte) ([]byte, error) {
	compressor := NewCompressor(Deflate, DefaultCompression)
	return compressor.Compress(data)
}

// DecompressDeflate decompresses deflate-compressed data.
func DecompressDeflate(data []byte) ([]byte, error) {
	decompressor := NewDecompressor(Deflate)
	return decompressor.Decompress(data)
}

// StreamCompressor provides streaming compression.
type StreamCompressor struct {
	algorithm Algorithm
	level     Level
	writer    io.Writer
}

// NewStreamCompressor creates a new streaming compressor.
func NewStreamCompressor(algo Algorithm, level Level, output io.Writer) (*StreamCompressor, error) {
	sc := &StreamCompressor{
		algorithm: algo,
		level:     level,
	}

	switch algo {
	case Gzip:
		return sc.initGzipWriter(output)
	case Zlib:
		return sc.initZlibWriter(output)
	case Deflate:
		return sc.initDeflateWriter(output)
	default:
		return nil, fmt.Errorf("streaming compression not supported for algorithm: %d", algo)
	}
}

func (sc *StreamCompressor) initGzipWriter(output io.Writer) (*StreamCompressor, error) {
	var gzipLevel int
	switch sc.level {
	case NoCompression:
		gzipLevel = gzip.NoCompression
	case BestSpeed:
		gzipLevel = gzip.BestSpeed
	case BestCompression:
		gzipLevel = gzip.BestCompression
	default:
		gzipLevel = gzip.DefaultCompression
	}

	writer, err := gzip.NewWriterLevel(output, gzipLevel)
	if err != nil {
		return nil, err
	}

	sc.writer = writer
	return sc, nil
}

func (sc *StreamCompressor) initZlibWriter(output io.Writer) (*StreamCompressor, error) {
	var zlibLevel int
	switch sc.level {
	case NoCompression:
		zlibLevel = zlib.NoCompression
	case BestSpeed:
		zlibLevel = zlib.BestSpeed
	case BestCompression:
		zlibLevel = zlib.BestCompression
	default:
		zlibLevel = zlib.DefaultCompression
	}

	writer, err := zlib.NewWriterLevel(output, zlibLevel)
	if err != nil {
		return nil, err
	}

	sc.writer = writer
	return sc, nil
}

func (sc *StreamCompressor) initDeflateWriter(output io.Writer) (*StreamCompressor, error) {
	var flateLevel int
	switch sc.level {
	case NoCompression:
		flateLevel = flate.NoCompression
	case BestSpeed:
		flateLevel = flate.BestSpeed
	case BestCompression:
		flateLevel = flate.BestCompression
	default:
		flateLevel = flate.DefaultCompression
	}

	writer, err := flate.NewWriter(output, flateLevel)
	if err != nil {
		return nil, err
	}

	sc.writer = writer
	return sc, nil
}

// Write writes data to the stream compressor.
func (sc *StreamCompressor) Write(data []byte) (int, error) {
	return sc.writer.Write(data)
}

// Close closes the stream compressor and flushes any remaining data.
func (sc *StreamCompressor) Close() error {
	if closer, ok := sc.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// StreamDecompressor provides streaming decompression.
type StreamDecompressor struct {
	algorithm Algorithm
	reader    io.Reader
}

// NewStreamDecompressor creates a new streaming decompressor.
func NewStreamDecompressor(algo Algorithm, input io.Reader) (*StreamDecompressor, error) {
	sd := &StreamDecompressor{
		algorithm: algo,
	}

	switch algo {
	case Gzip:
		return sd.initGzipReader(input)
	case Zlib:
		return sd.initZlibReader(input)
	case Deflate:
		return sd.initDeflateReader(input)
	default:
		return nil, fmt.Errorf("streaming decompression not supported for algorithm: %d", algo)
	}
}

func (sd *StreamDecompressor) initGzipReader(input io.Reader) (*StreamDecompressor, error) {
	reader, err := gzip.NewReader(input)
	if err != nil {
		return nil, err
	}

	sd.reader = reader
	return sd, nil
}

func (sd *StreamDecompressor) initZlibReader(input io.Reader) (*StreamDecompressor, error) {
	reader, err := zlib.NewReader(input)
	if err != nil {
		return nil, err
	}

	sd.reader = reader
	return sd, nil
}

func (sd *StreamDecompressor) initDeflateReader(input io.Reader) (*StreamDecompressor, error) {
	reader := flate.NewReader(input)
	sd.reader = reader
	return sd, nil
}

// Read reads decompressed data from the stream.
func (sd *StreamDecompressor) Read(p []byte) (int, error) {
	return sd.reader.Read(p)
}

// Close closes the stream decompressor.
func (sd *StreamDecompressor) Close() error {
	if closer, ok := sd.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// CompressionInfo provides information about compression results.
type CompressionInfo struct {
	Algorithm        Algorithm
	OriginalSize     int
	CompressedSize   int
	CompressionRatio float64
}

// AnalyzeCompression analyzes compression efficiency.
func AnalyzeCompression(original, compressed []byte, algo Algorithm) *CompressionInfo {
	ratio := float64(len(compressed)) / float64(len(original))

	return &CompressionInfo{
		Algorithm:        algo,
		OriginalSize:     len(original),
		CompressedSize:   len(compressed),
		CompressionRatio: ratio,
	}
}

// CompressionLevel returns a textual description of compression level.
func (ci *CompressionInfo) CompressionLevel() string {
	if ci.CompressionRatio < 0.3 {
		return "Excellent"
	} else if ci.CompressionRatio < 0.5 {
		return "Very Good"
	} else if ci.CompressionRatio < 0.7 {
		return "Good"
	} else if ci.CompressionRatio < 0.9 {
		return "Fair"
	} else {
		return "Poor"
	}
}

// SpaceSaved returns the amount of space saved by compression.
func (ci *CompressionInfo) SpaceSaved() int {
	return ci.OriginalSize - ci.CompressedSize
}

// SpaceSavedPercentage returns the percentage of space saved.
func (ci *CompressionInfo) SpaceSavedPercentage() float64 {
	return (1.0 - ci.CompressionRatio) * 100.0
}
