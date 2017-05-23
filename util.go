package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Generate a stable SHA256 for a commit.
func createCommitID(c commit) string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("tree %s\n", c.Tree))
	if c.Parent != "" {
		b.WriteString(fmt.Sprintf("parent %s\n", c.Parent))
	}
	b.WriteString(fmt.Sprintf("author %s <%s> %v\n", c.AuthorName, c.AuthorEmail,
		c.Timestamp.Format(time.UnixDate)))
	if c.CommitterEmail != "" {
		b.WriteString(fmt.Sprintf("committer %s <%s> %v\n", c.CommitterName, c.CommitterEmail,
			c.Timestamp.Format(time.UnixDate)))
	}
	b.WriteString("\n" + c.Message)
	b.WriteByte(0)
	s := sha256.Sum256(b.Bytes())
	return hex.EncodeToString(s[:])
}

// Generate the SHA256 for a tree.
func createDBTreeID(entries []dbTreeEntry) string {
	var b bytes.Buffer
	for _, j := range entries {
		b.WriteString(string(j.AType))
		b.WriteByte(0)
		b.WriteString(j.ShaSum)
		b.WriteByte(0)
		b.WriteString(j.Name + "\n")
	}
	s := sha256.Sum256(b.Bytes())
	return hex.EncodeToString(s[:])
}

// Generates the user visible commit text for a commit.
func generateCommitText(c commit) (string, error) {
	s := fmt.Sprintf("commit %s\n", createCommitID(c))
	s += fmt.Sprintf("Author: %s <%s>\n", c.AuthorName, c.AuthorEmail)
	s += fmt.Sprintf("Date: %v\n\n", c.Timestamp.Format(time.UnixDate))
	if c.Message != "" {
		s += fmt.Sprintf("    %s\n", c.Message)
	}
	return s, nil
}

func getIndex(d string) ([]commit, error) {
	b, err := ioutil.ReadFile(filepath.Join(STORAGEDIR, "meta", d, "index"))
	if err != nil {
		return nil, err
	}
	var i []commit
	err = json.Unmarshal(b, &i)
	if err != nil {
		log.Printf("Something went wrong when unserialising the index data: %v\n", err.Error())
		return nil, err
	}
	return i, nil
}

// Store a set of branches.
func storeBranches(dbPath string, branches []branch) error {
	// Create the storage directory if needed
	_, err := os.Stat(filepath.Join(STORAGEDIR, "meta", dbPath))
	if err != nil {
		// As this is just experimental code, we'll assume a failure above means the directory needs creating
		// TODO: Proper error checking
		err := os.MkdirAll(filepath.Join(STORAGEDIR, "meta", dbPath), os.ModeDir|0755)
		if err != nil {
			log.Printf("Something went wrong when creating the storage dir: %v\n",
				err.Error())
			return err
		}
	}
	j, err := json.MarshalIndent(branches, "", " ")
	if err != nil {
		log.Printf("Something went wrong when serialising the branch data: %v\n", err.Error())
		return err
	}
	err = ioutil.WriteFile(filepath.Join(STORAGEDIR, "meta", dbPath, "branches"), j, os.ModePerm)
	if err != nil {
		log.Printf("Something went wrong when writing the branches file: %v\n", err.Error())
		return err
	}
	return nil
}

// Store a commit.
func storeCommit(c commit) error {
	// Create the storage directory if needed
	_, err := os.Stat(filepath.Join(STORAGEDIR, "files"))
	if err != nil {
		// As this is just experimental code, we'll assume a failure above means the directory needs creating
		// TODO: Proper error checking
		err := os.MkdirAll(filepath.Join(STORAGEDIR, "files"), os.ModeDir|0755)
		if err != nil {
			log.Printf("Something went wrong when creating the storage dir: %v\n",
				err.Error())
			return err
		}
	}
	j, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		log.Printf("Something went wrong when serialising the commit data: %v\n", err.Error())
		return err
	}
	err = ioutil.WriteFile(filepath.Join(STORAGEDIR, "files", c.ID), j, os.ModePerm)
	if err != nil {
		log.Printf("Something went wrong when writing the commit file: %v\n", err.Error())
		return err
	}
	return nil
}

// Store a database file.
func storeDatabase(db []byte) (string, error) {
	// Create the storage directory if needed
	_, err := os.Stat(filepath.Join(STORAGEDIR, "files"))
	if err != nil {
		// As this is just experimental code, we'll assume a failure above means the directory needs creating
		err := os.MkdirAll(filepath.Join(STORAGEDIR, "files"), os.ModeDir|0755)
		if err != nil {
			log.Printf("Something went wrong when creating the storage dir: %v\n",
				err.Error())
			return "", err
		}
	}

	// Create the database file if it doesn't already exist
	s := sha256.Sum256(db)
	t := hex.EncodeToString(s[:])
	p := filepath.Join(STORAGEDIR, "files", t)
	f, err := os.Stat(p)
	if err != nil {
		// As this is just experimental code, we'll assume a failure above means the file needs creating
		err = ioutil.WriteFile(p, db, os.ModePerm)
		if err != nil {
			log.Printf("Something went wrong when writing the database file: %v\n", err.Error())
			return "", err
		}
		return t, nil
	}

	// The file already exists, so check if the file size matches the buffer size we're intending on writing
	// (Obviously this is just a super lightweight check, not a real world approach)
	// TODO: Add real world checks to ensure the file contents are identical.  Maybe read the file contents into
	// TODO  memory, then binary compare them?  Prob not great for memory efficiency, but it would likely do as a
	// TODO  first go for something accurate.
	if len(db) != int(f.Size()) {
		err = ioutil.WriteFile(p, db, os.ModePerm)
		if err != nil {
			log.Printf("Something went wrong when writing the database file: %v\n", err.Error())
			return "", err
		}
	}
	return t, nil
}

// Store an index.
func storeIndex(dbPath string, index []commit) error {
	// Create the storage directory if needed
	_, err := os.Stat(filepath.Join(STORAGEDIR, "meta", dbPath))
	if err != nil {
		// As this is just experimental code, we'll assume a failure above means the directory needs creating
		err := os.MkdirAll(filepath.Join(STORAGEDIR, "meta", dbPath), os.ModeDir|0755)
		if err != nil {
			log.Printf("Something went wrong when creating the storage dir: %v\n",
				err.Error())
			return err
		}
	}
	j, err := json.MarshalIndent(index, "", " ")
	if err != nil {
		log.Printf("Something went wrong when serialising the index data: %v\n", err.Error())
		return err
	}
	err = ioutil.WriteFile(filepath.Join(STORAGEDIR, "meta", dbPath, "index"), j, os.ModePerm)
	if err != nil {
		log.Printf("Something went wrong when writing the index file: %v\n", err.Error())
		return err
	}
	return nil
}

// Store a tree.
func storeTree(t dbTree) error {
	// Create the storage directory if needed
	_, err := os.Stat(filepath.Join(STORAGEDIR, "files"))
	if err != nil {
		// As this is just experimental code, we'll assume a failure above means the directory needs creating
		err := os.MkdirAll(filepath.Join(STORAGEDIR, "files"), os.ModeDir|0755)
		if err != nil {
			log.Printf("Something went wrong when creating the storage dir: %v\n",
				err.Error())
			return err
		}
	}
	j, err := json.MarshalIndent(t, "", " ")
	if err != nil {
		log.Printf("Something went wrong when serialising the tree data: %v\n", err.Error())
		return err
	}
	err = ioutil.WriteFile(filepath.Join(STORAGEDIR, "files", t.ID), j, os.ModePerm)
	if err != nil {
		log.Printf("Something went wrong when writing the tree file: %v\n", err.Error())
		return err
	}
	return nil
}
