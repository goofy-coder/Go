package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	mrand "math/rand"
	"os"
	"sort"
	"time"
)

var dbpath string
var cryptokey, iv []byte
var keys = make(map[string]key)

type key struct {
	Name     string
	Login    string
	Password string
}

func init() {
	// The IV needs to be unique, but not secure.
	iv = make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(crand.Reader, iv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// set up dbpath
	dbpath = os.Getenv("KEYBOXFILE")
	if len(dbpath) == 0 {
		fmt.Fprintln(os.Stderr, "KEYBOXFILE environment variable is not set")
		os.Exit(1)
	}
}

func promptForKey() *key {
	var r = bufio.NewReader(os.Stdin)
	//fmt.Print("Category: ")
	//category, _ := r.ReadString('\n')
	//category = category[:len(category)-1]

	fmt.Print("Name: ")
	name, _ := r.ReadString('\n')
	name = name[:len(name)-1]

	fmt.Print("Login: ")
	login, _ := r.ReadString('\n')
	login = login[:len(login)-1]

	fmt.Print("Password: ")
	password, _ := r.ReadString('\n')
	password = password[:len(password)-1]

	if len(name) > 0 && len(login) > 0 {
		if len(password) == 0 {
			password = newPassword()
		}
		return &key{name, login, password}
	}

	return nil
}

func promptForPassphrase(verify bool) string {
	fmt.Print("Passphrase: ")
	var r = bufio.NewReader(os.Stdin)
	v1, _ := r.ReadString('\n')

	if verify {
		fmt.Print("Passphrase: ")
		v2, _ := r.ReadString('\n')
		if v1 != v2 {
			fmt.Println("Passphrases do not match. Try it again.")
			promptForPassphrase(verify)
		}
	}

	return v1
}

func setCryptoKey(passphrase string) {
	// convert a passphrase to a key, use a suitable
	// package like bcrypt or scrypt.
	h := sha256.New()
	h.Write([]byte(passphrase))
	//fmt.Printf("crytoKey: %x\n", h.Sum(nil))
	cryptokey = h.Sum(nil)
}

func encrypt(plaintext, key, iv []byte) (ciphertext []byte, err error) {
	// CBC mode works on blocks so plaintexts may need to be padded to the
	// next whole block. For an example of such padding, see
	// https://tools.ietf.org/html/rfc5246#section-6.2.3.2.
	for i := len(plaintext) % aes.BlockSize; i < aes.BlockSize; i++ {
		plaintext = append(plaintext, byte(' '))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	ciphertext = make([]byte, len(plaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	// It's important to remember that ciphertexts must be authenticated
	// (i.e. by using crypto/hmac) as well as being encrypted in order to
	// be secure.

	return
}

func decrypt(ciphertext, key, iv []byte) (plaintext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	// CBC mode always works in whole blocks.
	if len(ciphertext)%aes.BlockSize != 0 {
		err = errors.New("ciphertext is not a multiple of the block size")
		return
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	// CryptBlocks can work in-place if the two arguments are the same.
	plaintext = make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	//However, it's critical to note that ciphertexts must be authenticated (i.e. by
	// using crypto/hmac) before being decrypted in order to avoid creating
	// a padding oracle.

	return
}

func deserialize(encoded []byte) (e key, err error) {
	err = json.Unmarshal(encoded, &e)
	return
}

func loadDB() error {
	f, err := os.OpenFile(dbpath, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}

	defer f.Close()

	bufrd := bufio.NewReaderSize(f, 516)
	iv = readLine(bufrd)
	if len(iv) != aes.BlockSize {
		return errors.New("Cannot read IV")
	}

	k := decryptRead(bufrd, cryptokey, iv)
	for ; k != nil; k = decryptRead(bufrd, cryptokey, iv) {
		keys[k.Name] = *k
	}

	return nil
}

func upsertKeys() {
	passphrase := promptForPassphrase(true)
	setCryptoKey(passphrase)

	if _, err := os.Stat(dbpath); err == nil {
		if err := loadDB(); err != nil {
			fmt.Printf("Cannot load the file %s", err)
		}
		// Save the file as we will overwrite later
		if err := os.Rename(dbpath, dbpath+".sav"); err != nil {
			fmt.Printf("Cannot save to file %s: %s", dbpath+".sav", err)
		}

	}

	f, err := os.Create(dbpath)
	if err != nil {
		fmt.Printf("Cannot create file %s: %s", dbpath, err)
	}

	os.Chmod(dbpath, 0600)
	defer f.Close()

	writeLine(f, iv)

	for {
		k := promptForKey()
		if k == nil {
			break
		}
		if _, found := keys[k.Name]; !found {
			keys[k.Name] = *k
		} else {
			fmt.Println("Key exits, overwrite...")
			keys[k.Name] = *k
		}
	}

	for _, k := range keys {
		s, _ := json.Marshal(k)
		ciphertext, _ := encrypt(s, cryptokey, iv)
		//fmt.Printf("+Line: %x\n", ciphertext)
		writeLine(f, ciphertext)
	}
}

func showKeys() {
	passphrase := promptForPassphrase(false)
	setCryptoKey(passphrase)
	if err := loadDB(); err != nil {
		fmt.Println(err)
	}

	ks := make([]string, 0, len(keys))
	for k := range keys {
		ks = append(ks, k)
	}

	sort.Strings(ks)

	for _, k := range ks {
		v := keys[k]
		fmt.Printf("%s: %s %s\n", v.Name, v.Login, v.Password)
	}
}

func readLine(bufrd *bufio.Reader) []byte {
	s, _ := bufrd.ReadString('\n')
	if len(s) == 0 {
		return nil
	}
	s = s[:len(s)-1]
	b, _ := hex.DecodeString(s)
	return b
}

func writeLine(f *os.File, b []byte) {
	f.WriteString(hex.EncodeToString(b))
	f.WriteString(string('\n'))
}

func decryptRead(bufrd *bufio.Reader, cryptokey, iv []byte) *key {
	ciphertext, _ := bufrd.ReadString('\n')
	if len(ciphertext) == 0 {
		return nil
	}
	ciphertext = ciphertext[:len(ciphertext)-1]
	b, _ := hex.DecodeString(ciphertext)

	plaintext, _ := decrypt(b, cryptokey, iv)
	k, err := deserialize(plaintext)
	if err != nil {
		fmt.Println("Passphrase is invalid. Please try again")
		os.Exit(1)
	}

	return &k
}

func restoreDB() {
	if err := os.Rename(dbpath+".sav", dbpath); err != nil {
		fmt.Printf("Retore failed: %s\n", err)
	}
}

func newPassword() string {
	// 3 of each: lowercase, uppercase, special letters and numbers
	specialties := [...]byte{'!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '?', '+', '~'}
	p := []byte{}
	mrand.Seed(int64(time.Now().Nanosecond()))
	for i := 0; i < 3; i++ {
		p = append(p,
			byte(int('A')+mrand.Intn(26)), // A-Z
			byte(int('a')+mrand.Intn(26)), // a-z
			byte(int('0')+mrand.Intn(10)), // 0-9
			specialties[mrand.Intn(len(specialties))])
	}

	// shuffle
	for i := 0; i < len(p); i++ {
		j := mrand.Intn(12)
		if i != j {
			p[i], p[j] = p[j], p[i]
		}
	}

	return string(p[0:12])
}

func info() {
	fmt.Println("Checking KEYBOXFILE environment variable...")
	finfo, err := os.Stat(dbpath)
	if os.IsNotExist(err) {
		fmt.Printf("Keybox file %s does not exit. Use create command to create one\n", dbpath)
	} else {
		fmt.Printf("Keybox file %s\n", dbpath)
		fmt.Printf("Last modified at %s\n", finfo.ModTime())
	}
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("keybox {info | update | show | restore | createpassword}")
		return
	}

	switch os.Args[1] {
	case "info":
		info()
	case "update":
		upsertKeys()
	case "show":
		showKeys()
	case "createpassword":
		fmt.Println(newPassword())
	case "restore":
		restoreDB()
	}

	fmt.Println("!!! DO NOT FORGET TO CLOSE THE WINDOW !!!")
}
