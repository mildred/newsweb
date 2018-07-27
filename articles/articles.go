package articles

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
)

type Articles struct {
	StorageDir string
}

type Group struct {
	Name        string
	Description string
	Count       int64
	High        int64
	Low         int64
}

func (ar *Articles) ListGroups() (res []*Group, err error) {
	d, err := os.Open(ar.StorageDir)
	if err != nil {
		return nil, err
	}
	defer d.Close()

	names, err := d.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	for _, name := range names {
		grp, err := ar.GetGroup(name)
		if err != nil {
			log.Printf("ERROR: Unable to read group %s: %v", name, err)
		} else {
			res = append(res, grp)
		}
	}

	return res, nil
}

func (ar *Articles) GetGroup(name string) (grp *Group, err error) {
	grp = &Group{
		Name:        name,
		Description: name,
		Low:         0,
	}

	d, err := os.Open(path.Join(ar.StorageDir, name))
	if err != nil {
		return nil, err
	}
	defer d.Close()

	names, err := d.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	for _, fname := range names {
		if fname == "." || fname == ".." {
			continue
		} else if fname == "description" {
			desc, err := ioutil.ReadFile(path.Join(ar.StorageDir, name, fname))
			if err != nil {
				grp.Description = ""
				log.Printf("ERROR: Unable to read description for group %s: %v", name, err)
			} else {
				grp.Description = string(desc)
			}
		} else if num, err := strconv.ParseInt(fname, 10, 64); err == nil && num > 0 {
			if grp.Count == 0 {
				grp.Low = num
				grp.High = num
			} else if num < grp.Low {
				grp.Low = num
			} else if num > grp.High {
				grp.High = num
			}
			grp.Count++
		}
	}

	return grp, nil
}

func (ar *Articles) Post(group string, data []byte) (int64, error) {
	err := os.MkdirAll(path.Join(ar.StorageDir, group), 0755)
	if err != nil {
		return -1, err
	}

	var num int64 = 1
	for {
		f, err := os.Create(path.Join(ar.StorageDir, group, strconv.FormatInt(num, 10)))
		if err != nil && os.IsExist(err) {
			num++
			continue
		} else if err != nil {
			return -1, err
		}
		defer f.Close()

		_, err = f.Write(data)
		if err != nil {
			return -1, err
		}

		return num, nil
	}
}
