package redis

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
)

// GzipMinSize gzip min size
const GzipMinSize = 1024

// 压缩
func toGzipJSON(obj interface{}) (gziped bool, data []byte, err error) {
	bs, err := json.Marshal(obj)
	if err != nil {
		return
	}
	if len(bs) <= GzipMinSize {
		return false, bs, nil
	}
	buf := &bytes.Buffer{}
	gzipWriter := gzip.NewWriter(buf)
	_, err = gzipWriter.Write(bs)
	gzipWriter.Close()
	if err != nil {
		return
	}
	return true, buf.Bytes(), nil
}

// 解压
func fromGzipJSON(data []byte, obj interface{}) (err error) {
	buf := bytes.NewBuffer(data)
	gzipReader, err := gzip.NewReader(buf)
	if err != nil {
		return
	}
	defer gzipReader.Close()
	bs, err := io.ReadAll(gzipReader)
	if err != nil {
		return
	}
	return json.Unmarshal(bs, obj)
}
