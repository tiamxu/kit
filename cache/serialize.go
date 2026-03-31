package cache

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"sync"
)

const (
	cacheFlagJSON     uint32 = 10
	cacheFlagJSONGzip uint32 = 11
)

var (
	bufferPool = sync.Pool{
		New: func() interface{} { return new(bytes.Buffer) },
	}
	gzipWriterPool = sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
			return w
		},
	}
)

type cacheItem struct {
	Flag uint32
	Data []byte
}

func serialize(obj interface{}, compress bool, minSize int) ([]byte, error) {
	bs, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	if !compress || len(bs) <= minSize {
		return encodeItem(bs, cacheFlagJSON)
	}

	compressed, err := gzipCompress(bs)
	if err != nil {
		return nil, err
	}
	if len(compressed) >= len(bs) {
		return encodeItem(bs, cacheFlagJSON)
	}
	return encodeItem(compressed, cacheFlagJSONGzip)
}

func deserialize(data []byte, dest interface{}) error {
	item, err := decodeItem(data)
	if err != nil {
		return err
	}

	switch item.Flag {
	case cacheFlagJSON:
		return json.Unmarshal(item.Data, dest)
	case cacheFlagJSONGzip:
		bs, err := gzipDecompress(item.Data)
		if err != nil {
			return err
		}
		return json.Unmarshal(bs, dest)
	default:
		return nil
	}
}

func encodeItem(data []byte, flag uint32) ([]byte, error) {
	item := cacheItem{Flag: flag, Data: data}

	buf := bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()

	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(item); err != nil {
		return nil, err
	}
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

func decodeItem(data []byte) (*cacheItem, error) {
	var item cacheItem
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&item); err != nil {
		return nil, err
	}
	return &item, nil
}

func gzipCompress(data []byte) ([]byte, error) {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()

	gz := gzipWriterPool.Get().(*gzip.Writer)
	defer gzipWriterPool.Put(gz)
	gz.Reset(buf)

	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

func gzipDecompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	bs := make([]byte, 0, len(data)*3)
	if bs, err = io.ReadAll(reader); err != nil {
		return nil, err
	}
	return bs, nil
}
