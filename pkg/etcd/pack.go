package etcd

import (
	"archive/tar"
	"bytes"
	"io"
)

func readKeyValue(tarReader *tar.Reader) ([]byte, []byte, error) {
	header, err := tarReader.Next()
	if err != nil {
		return nil, nil, err
	}

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, tarReader)
	if err != nil {
		return nil, nil, err
	}

	return []byte(header.Name), buf.Bytes(), nil
}

func writeKeyValue(tarWriter *tar.Writer, key, value []byte) error {
	err := tarWriter.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     string(key),
		Size:     int64(len(value)),
	})
	if err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, bytes.NewReader(value))
	if err != nil {
		return err
	}

	return nil
}
