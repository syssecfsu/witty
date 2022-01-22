package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"github.com/dchest/uniuri"
	"golang.org/x/term"
)

const (
	userFileName = "user.db"
)

type UserRecord struct {
	User   []byte   `json:"Username"`
	Seed   []byte   `json:"Seed"`
	Passwd [32]byte `json:"Password"`
}

func hashPassword(seed []byte, passwd []byte) [32]byte {
	input := append(seed, passwd...)
	return sha256.Sum256(input)
}

func addUser(username []byte, passwd []byte) {
	var users []UserRecord
	var err error

	seed := []byte(uniuri.NewLen(64))
	hashed := hashPassword(seed, passwd)

	exist := false
	file, err := os.ReadFile(userFileName)

	if (err == nil) && (json.Unmarshal(file, users) == nil) {
		// update the existing user if it exists
		for _, u := range users {
			if bytes.Equal(u.User, username) {
				u.Seed = seed
				u.Passwd = hashed
				exist = true
				break
			}
		}
	}

	if !exist {
		users = append(users, UserRecord{username, seed, hashed})
	}

	output, err := json.Marshal(users)
	if err != nil {
		fmt.Println("Failed to marshal passwords", err)
		return
	}

	os.WriteFile(userFileName, output, 0660)
}

func AddUser(username string) {
	fmt.Println("Please type your password (it will not be echoed back):")
	passwd, err := term.ReadPassword(int(os.Stdin.Fd()))

	if err != nil {
		fmt.Println("Failed to read password", err)
		return
	}

	fmt.Println("Please type your password again:")
	passwd2, err := term.ReadPassword(int(os.Stdin.Fd()))

	if err != nil {
		fmt.Println("Failed to read password", err)
		return
	}

	if !bytes.Equal(passwd, passwd2) {
		fmt.Println("Password mismatch, try again")
		return
	}

	addUser([]byte(username), passwd)
}

func DelUser(username string) {
	var users []UserRecord
	var err error

	exist := false
	file, err := os.ReadFile(userFileName)
	if err != nil {
		fmt.Println("Failed to read users file", err)
		return
	}

	err = json.Unmarshal(file, &users)

	if err != nil {
		fmt.Println("Failed to parse json format", err)
		return
	}
	// update the existing user if it exists
	for i, u := range users {
		if bytes.Equal(u.User, []byte(username)) {
			users = append(users[:i], users[i+1:]...)
			exist = true
			break
		}
	}

	if exist {
		output, err := json.Marshal(users)
		if err != nil {
			fmt.Println("Failed to marshal passwords", err)
			return
		}

		os.WriteFile(userFileName, output, 0660)
	}
}

func ValidateUser(username []byte, passwd []byte) bool {
	var users []UserRecord
	var err error

	file, err := os.ReadFile(userFileName)
	if err != nil {
		fmt.Println("Failed to read users file", err)
		return false
	}

	err = json.Unmarshal(file, &users)

	if err != nil {
		fmt.Println("Failed to parse json format", err)
		return false
	}

	// update the existing user if it exists
	for _, u := range users {
		if bytes.Equal(u.User, []byte(username)) {
			hashed := hashPassword(u.Seed, passwd)
			return bytes.Equal(hashed[:], u.Passwd[:])
		}
	}

	return false
}
