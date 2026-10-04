// Package password 封装密码哈希与校验。默认 bcrypt。
package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// Algo bcrypt / argon2 二选一，目前只实现 bcrypt。
const AlgoBcrypt = "bcrypt"

// Hash 使用 bcrypt 生成密码哈希。
func Hash(plain string) (string, error) {
	if plain == "" {
		return "", errors.New("password: empty plain")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Verify 使用 bcrypt 校验密码。返回 true 表示匹配。
func Verify(hash, plain string) bool {
	if hash == "" || plain == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
