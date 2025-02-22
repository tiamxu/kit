package redis

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"sync"
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

func toGzipJSON(obj interface{}, gzipMinSize int) (gziped bool, data []byte, err error) {
	bs, err := json.Marshal(obj)
	if err != nil {
		return false, nil, err
	}

	// 动态阈值判断
	if len(bs) <= gzipMinSize {
		return false, bs, nil
	}

	// 从内存池获取资源
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	// 从池中获取writer
	gzWriter := gzipWriterPool.Get().(*gzip.Writer)
	defer gzipWriterPool.Put(gzWriter)
	gzWriter.Reset(buf)

	if _, err = gzWriter.Write(bs); err != nil {
		return false, nil, err
	}
	if err = gzWriter.Close(); err != nil {
		return false, nil, err
	}
	compressedData := buf.Bytes()
	if len(compressedData) >= len(bs) {
		return false, bs, nil // 压缩后反而更大，放弃压缩
	}
	return true, buf.Bytes(), nil
}

// 解压逻辑
func fromGzipJSON(data []byte, obj interface{}) (err error) {
	// 使用更高效的内存分配方式
	buf := bytes.NewBuffer(data)

	gzReader, err := gzip.NewReader(buf)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	// 预分配内存
	bs := make([]byte, 0, len(data)*3) // 根据压缩比预估
	if bs, err = io.ReadAll(gzReader); err != nil {
		return err
	}

	return json.Unmarshal(bs, obj)
}
