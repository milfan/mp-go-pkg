// Package pkgminio is general utils for minio storage
package pkgminio

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime/multipart"
)

func calculateSHA256FromMultipartFile(file multipart.File) (string, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	shaSum := sha256.Sum256(data)
	checksum := hex.EncodeToString(shaSum[:])

	// Kembalikan data agar bisa dipakai lagi
	return checksum, nil
}
