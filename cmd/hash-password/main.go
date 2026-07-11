package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

const bcryptCost = 12

func main() {
	fmt.Print("Password : ")

	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf("read password : %v", err)
	}

	fmt.Print("\nConfirm password: ")

	confirmation, err := term.ReadPassword(int(os.Stdin.Fd()))

	if err != nil {
		log.Fatalf("read password confirmation: %v", err)
	}

	fmt.Println()

	if !bytes.Equal(password, confirmation) {
		log.Fatal("passwords do not match")
	}

	if len(password) < 12 {
		log.Fatal("password must contain at least 12 characters")
	}

	if len(password) > 72 {
		log.Fatal("bcrypt supports passwords up to 72 bytes")
	}

	hash, err := bcrypt.GenerateFromPassword(
		password,
		bcryptCost,
	)
	if err != nil {
		log.Fatalf("generate password hash: %v", err)
	}

	fmt.Println(string(hash))
}
