package CryptoUtil

import (
	ConverUtil "Github.com/mhthrh/EventDriven/Utilitys/ConvertUtil"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type Crypto struct {
	key      string
	FilePath string
	Text     string
}

var (
	validationDuration = 30 * time.Minute
	dateFormat         = time.UnixDate
)

func NewKey() *Crypto {
	c := new(Crypto)
	c.key = "AnKoloft@~delNazok!12345" // key parameter must be 16, 24 or 32,
	return c
}

func (k *Crypto) Sha256() string {
	h := sha256.New()
	h.Write([]byte(k.Text))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (k *Crypto) Md5Sum() (string, error) {
	file, err := os.Open(k.FilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (k *Crypto) Encrypt() (string, error) {
	key := []byte(k.key)
	plaintext := []byte(k.Text)
	c, err := aes.NewCipher(key)
	if err != nil {
		return "", err

	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err

	}

	return ConverUtil.Byte64(gcm.Seal(nonce, nonce, plaintext, nil)), nil
}

func (k *Crypto) Decrypt() (string, error) {
	key := []byte(k.key)
	bb, _ := base64.StdEncoding.DecodeString(k.Text)
	ciphertext := []byte(bb)
	c, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", err
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	t, e := gcm.Open(nil, nonce, ciphertext, nil)
	if e != nil {
		return "", err
	}
	return ConverUtil.BytesToString(t), nil

}

func (k *Crypto) GenerateToken(params ...string) (string, error) {

	k.Text = fmt.Sprintf("%s#%s#%s", params[0], params[1], time.Now().Add(validationDuration).Format(dateFormat))
	token, err := k.Encrypt()
	if err != nil {
		return "", err
	}
	return token, nil
}

func (k *Crypto) CheckSignKey(params ...string) (bool, error) { //username email token
	k.Text = params[2]
	dec, err := k.Decrypt()
	if err != nil {
		return false, err
	}
	spl := strings.Split(dec, "#")
	if len(spl) != 3 {
		return false, fmt.Errorf("error in sign key")
	}

	if params[0] != spl[0] {
		return false, fmt.Errorf("user not found")
	}
	if params[1] != spl[1] {
		return false, fmt.Errorf("email not found")
	}

	signedTime, err := time.Parse(time.UnixDate, spl[2])
	if err != nil {
		return false, err
	}
	if time.Now().Before(signedTime) {
		return true, nil
	}
	return false, fmt.Errorf("token has been expierd")

}
