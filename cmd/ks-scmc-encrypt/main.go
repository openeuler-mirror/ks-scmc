package main

import (
	"log"
	"os"

	"scmc/common/aes"
)

/*
func foo() {
	if len(os.Args) != 3 {
		log.Printf("Usage: %s e/encrypt/d/decrypt DATA", os.Args[0])
		return
	}
	action := os.Args[1]
	data := os.Args[2]

	switch action {
	case "e", "encrypt":
		bs, err := common.AesEncrypt([]byte(data))
		if err != nil {
			log.Println("error: ", err)
			return
		}
		log.Printf("%s", bs)
	case "d", "decrypt":
		bs, err := common.AesDecrypt(data)
		if err != nil {
			log.Println("error: ", err)
			return
		}
		log.Printf("%s", bs)
	}
}
*/

func main() {
	if len(os.Args) != 2 {
		log.Printf("Usage: %s DATA", os.Args[0])
		return
	}

	bs, err := aes.Encrypt([]byte(os.Args[1]))
	if err != nil {
		log.Println("error: ", err)
		return
	}
	log.Printf("%s", bs)
}

func init() {
	log.SetFlags(0)
}
