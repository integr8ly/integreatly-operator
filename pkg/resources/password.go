package resources

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"math/big"
	mathrand "math/rand"
	"strings"
)

var (
	lowerCharSet   = "abcdedfghijklmnopqrst"
	upperCharSet   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	specialCharSet = "!@#$%&*"
	numberSet      = "0123456789"
	allCharSet     = lowerCharSet + upperCharSet + specialCharSet + numberSet
)

type cryptoRandSource struct{}

func newCryptoRandSource() cryptoRandSource {
	return cryptoRandSource{}
}

func (s cryptoRandSource) Int63() int64 {
	// #nosec G115 -- This is a standard implementation of rand.Source, the bitwise operation ensures the value fits in an int64.
	return int64(s.Uint64() & ^uint64(1<<63))
}

func (s cryptoRandSource) Seed(_ int64) {}

func (s cryptoRandSource) Uint64() (v uint64) {
	err := binary.Read(rand.Reader, binary.BigEndian, &v)
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func GenerateRandomPassword(passwordLength, minSpecialChar, minNum, minUpperCase int) string {
	var password strings.Builder

	//Set special character
	for i := 0; i < minSpecialChar; i++ {
		random, err := rand.Int(rand.Reader, big.NewInt(int64(len(specialCharSet))))
		if err != nil {
			fmt.Printf("failed to generate password while setting special character with error: %v", err)
			return ""
		}
		password.WriteString(string(specialCharSet[random.Int64()]))
	}

	//Set numeric
	for i := 0; i < minNum; i++ {
		random, err := rand.Int(rand.Reader, big.NewInt(int64(len(numberSet))))
		if err != nil {
			fmt.Printf("failed to generate password while setting numeric with error: %v", err)
			return ""
		}
		password.WriteString(string(numberSet[random.Int64()]))
	}

	//Set uppercase
	for i := 0; i < minUpperCase; i++ {
		random, err := rand.Int(rand.Reader, big.NewInt(int64(len(upperCharSet))))
		if err != nil {
			fmt.Printf("failed to generate password while setting uppercase with error: %v", err)
			return ""
		}
		password.WriteString(string(upperCharSet[random.Int64()]))
	}

	remainingLength := passwordLength - minSpecialChar - minNum - minUpperCase
	for i := 0; i < remainingLength; i++ {
		random, err := rand.Int(rand.Reader, big.NewInt(int64(len(allCharSet))))
		if err != nil {
			fmt.Printf("failed to generate password with error: %v", err)
			return ""
		}
		password.WriteString(string(allCharSet[random.Int64()]))
	}
	inRune := []rune(password.String())

	// use math/rand shuffle backed by crypto/rand. Suppressing gosec warning for this line
	// #nosec G404
	rnd := mathrand.New(newCryptoRandSource())
	rnd.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
}
