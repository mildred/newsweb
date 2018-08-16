package articles

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
	"path"

	"github.com/coreos/bbolt"
)

const DbName = "index.db"

type Articles struct {
	StorageDir string
	db         *bolt.DB
}

type Group struct {
	Name        string
	Description string
	Count       int64
	High        int64
	Low         int64
}

func (ar *Articles) Open() error {
	var err error
	ar.Close()
	ar.db, err = bolt.Open(path.Join(ar.StorageDir, DbName), 0644, nil)
	return err
}

func (ar *Articles) Close() error {
	if ar.db != nil {
		return ar.db.Close()
		ar.db = nil
	}
	return nil
}

func (ar *Articles) ListGroups() (res []*Group, err error) {
	err = ar.db.View(func(tx *bolt.Tx) error {
		groups := tx.Bucket([]byte("groups"))
		if groups == nil {
			return nil
		}

		cur := groups.Cursor()
		for k, v := cur.First(); k != nil; k, v = cur.Next() {
			if v != nil {
				continue
			}
			bucket := groups.Bucket(k)
			var group = new(Group)
			group.Name = string(k)
			readGroup(bucket, group)
			res = append(res, group)
		}
		return nil
	})

	return res, err
}

var ErrNoGroup = errors.New("No such group")

func readGroup(bucket *bolt.Bucket, group *Group) {
	first, err := btoi(bucket.Get(KeyGroupFirst))
	if err != nil {
		first = 1
	}
	last, err := btoi(bucket.Get(KeyGroupLast))
	if err != nil {
		last = 0
	}
	count, err := btoi(bucket.Get(KeyGroupCount))
	if err != nil {
		count = 0
	}
	descr := string(bucket.Get(KeyGroupDescr))

	group.Low = first
	group.High = last
	group.Count = count
	group.Description = descr
	log.Printf("DEBUG: read group %s %d %d %d - %s", group.Name, group.Low, group.High, group.Count, group.Description)
}

func (ar *Articles) GetGroup(name string) (group *Group, err error) {
	group = &Group{
		Name:        name,
		Description: name,
		Low:         0,
	}
	err = ar.db.View(func(tx *bolt.Tx) error {
		groups := tx.Bucket([]byte("groups"))
		if groups == nil {
			group = nil
			return ErrNoGroup
		}

		grp := groups.Bucket([]byte(name))
		if grp == nil {
			return ErrNoGroup
		}
		readGroup(grp, group)
		return nil
	})

	return group, err
}

func (ar *Articles) getPath(hash string) (string, string) {
	return path.Join(ar.StorageDir, "data", hash[0:2], hash[2:4]), hash
}

func (ar *Articles) GetArticleMsgId(groupName string, msgId string) (io.ReadCloser, int64, error) {
	info, err := ar.getArticleInfo(groupName,
		encodeStrKey(MsgIdFilePrefix, msgId),
		encodeStrKey(MsgIdNumPrefix, msgId))
	if err != nil {
		return nil, -1, err
	}

	num, err := btoi(info[1])
	if err != nil {
		return nil, -1, err
	}

	art, err := ar.getArticleFromHash(groupName, string(info[0]))
	return art, num, err
}

func (ar *Articles) GetArticleNum(groupName string, num int64) (io.ReadCloser, string, error) {
	info, err := ar.getArticleInfo(groupName,
		encodeIntKey(NumFilePrefix, num),
		encodeIntKey(NumMsgIdPrefix, num))
	if err != nil {
		return nil, "", err
	}

	art, err := ar.getArticleFromHash(groupName, string(info[0]))
	return art, string(info[1]), err
}

func (ar *Articles) getArticleInfo(groupName string, keys ...[]byte) ([][]byte, error) {
	var res [][]byte
	err := ar.db.View(func(tx *bolt.Tx) error {
		groups := tx.Bucket([]byte("groups"))
		if groups == nil {
			return ErrNoGroup
		}

		grp := groups.Bucket([]byte(groupName))
		if grp == nil {
			return ErrNoGroup
		}

		for _, key := range keys {
			res = append(res, grp.Get(key))
		}
		return nil
	})
	return res, err
}

func (ar *Articles) getArticleFromHash(groupName string, hash string) (io.ReadCloser, error) {
	if hash == "" {
		return nil, nil
	}

	dir, fname := ar.getPath(hash)
	f, err := os.Open(path.Join(dir, fname))
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (ar *Articles) Post(groupNames []string, msgId string, data []byte) error {
	binHash := sha256.Sum256(data)
	hash := hex.EncodeToString(binHash[:])
	dir, fname := ar.getPath(hash)

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	f, err := os.Create(path.Join(dir, fname))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	return ar.db.Update(func(tx *bolt.Tx) error {
		groups, err := tx.CreateBucketIfNotExists([]byte("groups"))
		panicIfError(err)

		for _, groupName := range groupNames {
			grp, err := groups.CreateBucketIfNotExists([]byte(groupName))
			if err != nil {
				return err
			}

			last, err := btoi(grp.Get(KeyGroupLast))
			if err != nil {
				last = 1
				panicIfError(grp.Put(KeyGroupFirst, itob(1)))
			}
			count, err := btoi(grp.Get(KeyGroupCount))
			if err != nil {
				count = 0
			}
			num := last
			last++
			count++

			log.Printf("DEBUG: update group %s ? %d %d", groupName, last, count)

			panicIfError(grp.Put(KeyGroupCount, itob(count)))
			panicIfError(grp.Put(KeyGroupLast, itob(last)))
			panicIfError(grp.Put(encodeIntKey(NumFilePrefix, num), []byte(hash)))
			panicIfError(grp.Put(encodeIntKey(NumMsgIdPrefix, num), []byte(msgId)))
			panicIfError(grp.Put(encodeStrKey(MsgIdFilePrefix, msgId), []byte(hash)))
			panicIfError(grp.Put(encodeStrKey(MsgIdNumPrefix, msgId), itob(num)))
		}
		return nil
	})
}

var (
	KeyGroupFirst = []byte("first")
	KeyGroupLast  = []byte("last")
	KeyGroupCount = []byte("count")
	KeyGroupDescr = []byte("description")
)

const (
	NumFilePrefix   = "num-file."   // article number to filename
	NumMsgIdPrefix  = "num-msgid."  // article number to message-id
	MsgIdNumPrefix  = "msgid-num."  // message-id to article number
	MsgIdFilePrefix = "msgid-file." // message-id to file
)
