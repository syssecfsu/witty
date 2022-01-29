package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/dchest/uniuri"
	"golang.org/x/term"
)

const (
	userFileName = "./user.db"
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

	if err != nil {
		goto nonexist
	}

	if err = json.Unmarshal(file, &users); err != nil {
		log.Println("Failed to unmarsh file", err)
		goto nonexist
	}

	// update the existing user if it exists
	for i, u := range users {
		if bytes.Equal(u.User, username) {
			users[i].Seed = seed
			users[i].Passwd = hashed
			exist = true
			break
		}
	}

nonexist:
	if !exist {
		users = append(users, UserRecord{username, seed, hashed})
	}

	output, err := json.Marshal(users)
	if err != nil {
		log.Println("Failed to marshal passwords", err)
		return
	}

	os.WriteFile(userFileName, output, 0660)
}

func AddUser(username string) {
	fmt.Println("Please type your password (it will not be echoed back):")
	passwd, err := term.ReadPassword(int(os.Stdin.Fd()))

	if err != nil {
		log.Println("Failed to read password", err)
		return
	}

	if len(passwd) < 12 {
		fmt.Println("Password too short, at least 12 bytes")
		return
	}

	fmt.Println("Please type your password again:")
	passwd2, err := term.ReadPassword(int(os.Stdin.Fd()))

	if err != nil {
		log.Println("Failed to read password", err)
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
		log.Println("Failed to read users file", err)
		return
	}

	err = json.Unmarshal(file, &users)

	if err != nil {
		log.Println("Failed to parse json format", err)
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
			log.Println("Failed to marshal passwords", err)
			return
		}

		os.WriteFile(userFileName, output, 0660)
	}
}

func ListUsers() {
	var users []UserRecord
	var err error

	file, err := os.ReadFile(userFileName)
	if err != nil {
		log.Println("Failed to read users file", err)
		return
	}

	err = json.Unmarshal(file, &users)

	if err != nil {
		log.Println("Failed to parse json format", err)
		return
	}
	// update the existing user if it exists
	fmt.Println("Users of the system:")
	for _, u := range users {
		fmt.Println("   ", string(u.User))
	}
}

func ValidateUser(username []byte, passwd []byte) bool {
	var users []UserRecord
	var err error

	file, err := os.ReadFile(userFileName)
	if err != nil {
		log.Println("Failed to read users file", err)
		return false
	}

	err = json.Unmarshal(file, &users)

	if err != nil {
		log.Println("Failed to parse json format", err)
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
