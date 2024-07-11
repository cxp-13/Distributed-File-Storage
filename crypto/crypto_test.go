package crypto

import (
	"bytes"
	"testing"
)

func TestCopyEncryptDecrypt(t *testing.T) {
	payload := "hello world"
	src := bytes.NewReader([]byte(payload))
	dst := new(bytes.Buffer)

	key := NewEncryptionKey()
	_, err := CopyEncrypt(key, src, dst)
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", string(dst.Bytes()))

	out := new(bytes.Buffer)
	if _, err := CopyDecrypt(key, dst, out); err != nil {
		t.Error(err)
	}
	t.Logf("%s", string(out.Bytes()))
}
