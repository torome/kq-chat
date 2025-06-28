package tool

import (
	"crypto/md5"
	"fmt"
	"io"
	"math/rand"
	"time"
)

/** 加密方式 **/

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

func Md5ByString(str string) string {
	m := md5.New()
	_, err := io.WriteString(m, str)
	if err != nil {
		panic(err)
	}
	arr := m.Sum(nil)
	return fmt.Sprintf("%x", arr)
}

func Md5ByBytes(b []byte) string {
	return fmt.Sprintf("%x", md5.Sum(b))
}

func generateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func encryptNumber(number int) string {
	randomString := generateRandomString(16)
	return fmt.Sprintf("%d%s", number, randomString)
}

func decryptString(encryptedString string) (int, error) {
	number := 0
	_, err := fmt.Sscanf(encryptedString, "%d", &number)
	if err != nil {
		return 0, err
	}
	return number, nil
}
