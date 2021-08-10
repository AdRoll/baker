// Package zip_agnostic provides types and functions to deal with byte stream in
// a manner that is agnostic to the compression format, or its absence thereof.
//
// For example, if r is a reader over zstd or gzip compressed data, NewReader(r)
// returns an io.ReadCloser that reads the decompressed byte stream. If r reads
// non compressed data, or data that is compressed in non-supported or
// non-recognized format, then NewReader(r) simply buffers and forwards the data
// in r.
package zip_agnostic
