package validations

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"log"
	"path"

	"github.com/coreos/bbolt"
)

const DbName = "validations.db"
const TokenSize = 32

const (
	EmailTokenPrefix = "email-token." // email to token
	TokenEmailPrefix = "token-email." // token to email
	TokenSep         = " "
)

type Validations struct {
	StorageDir string
	db         *bolt.DB
}

func (v *Validations) Open() error {
	var err error
	v.Close()
	v.db, err = bolt.Open(path.Join(v.StorageDir, DbName), 0644, nil)
	return err
}

func (v *Validations) Close() error {
	if v.db != nil {
		return v.db.Close()
		v.db = nil
	}
	return nil
}

func (v *Validations) GenValidationToken(email string) (token string, err error) {
	token = genHexToken(TokenSize)
	err = v.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("validations"))
		panicIfError(err)

		tokens := bytes.Split(bucket.Get(encodeStrKey(EmailTokenPrefix, email)), []byte(TokenSep))
		tokens = append(tokens, []byte(token))
		return bucket.Put(encodeStrKey(EmailTokenPrefix, email), bytes.Join(tokens, []byte(TokenSep)))
	})
	return
}

func encodeStrKey(prefix, data string) []byte {
	return []byte(prefix + data)
}

func genHexToken(size int) string {
	var data = make([]byte, size)
	_, _ = rand.Read(data)
	var enc bytes.Buffer
	base64.NewEncoder(base64.RawURLEncoding, &enc).Write(data)
	return string(enc.Bytes())
}

func panicIfError(err error) {
	if err != nil {
		log.Panic(err)
	}
}
