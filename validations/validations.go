package validations

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"log"
	"path"
	"time"

	"github.com/coreos/bbolt"
)

const DbName = "validations.db"
const TokenSize = 32

const (
	EmailTokenPrefix  = "email-token."  // email to token
	TokenEmailPrefix  = "token-email."  // token to email
	TokenExpirePrefix = "token-expire." // token to expiry date
	TokenSep          = " "
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

		panicIfError(bucket.Put(encodeStrKey(TokenEmailPrefix, token), []byte(email)))
		panicIfError(bucket.Put(encodeStrKey(TokenExpirePrefix, token), encodeTime(time.Now())))

		tokens := bytes.Split(bucket.Get(encodeStrKey(EmailTokenPrefix, email)), []byte(TokenSep))
		tokens = append(tokens, []byte(token))
		return bucket.Put(encodeStrKey(EmailTokenPrefix, email), bytes.Join(tokens, []byte(TokenSep)))
	})
	return
}

func (v *Validations) CleanTokensBefore(t time.Time) (err error) {
	err = v.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("validations"))
		panicIfError(err)

		cur := bucket.Cursor()
		for k, v := cur.Seek([]byte(TokenExpirePrefix)); k != nil; k, v = cur.Next() {
			//key := string(k)
			//var token string
			//if strings.HasPrefix(key, TokenExpirePrefix) {
			//	token = key[len(TokenExpirePrefix):]
			//}
			time, err := decodeTime(v)
			if err == nil {
				continue
			}
			if time.Before(t) {
				panicIfError(cur.Delete())
			}
		}
		return nil
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

func encodeTime(t time.Time) []byte {
	return []byte(t.Format(time.RFC3339))
}

func decodeTime(d []byte) (time.Time, error) {
	return time.Parse(time.RFC3339, string(d))
}
