package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
)

func GenerateID() string {
	buf := make([]byte, 32)
	io.ReadFull(rand.Reader, buf)
	return hex.EncodeToString(buf)
}

func HashKey(key string) string {
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

func NewEncryptionKey() []byte {
	keyBuf := make([]byte, 32)
	io.ReadFull(rand.Reader, keyBuf)
	return keyBuf
}

func copyStream(stream cipher.Stream, src io.Reader, dst io.Writer) (int, error) {
	var (
		buf = make([]byte, 32*1024)
		//stream     = cipher.NewCTR(block, iv)
		totalBytes int
	)

	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf[:n], buf[:n])
			if _, err := dst.Write(buf[:n]); err != nil {
				return totalBytes, err
			}
			totalBytes += n
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return totalBytes, err
		}
	}

	return totalBytes, nil
}

func CopyDecrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}
	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(src, iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, src, dst)
}

func CopyEncrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}
	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}

	// Write the IV to the destination
	if _, err := dst.Write(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, src, dst)
}
