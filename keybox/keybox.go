package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	mrand "math/rand"
	"os"
	"sort"
	"time"

	"github.com/fatih/color"
)

type key struct {
	Name     string
	Login    string
	Password string
}

var dbpath string
var cryptokey []byte
var iv []byte = make([]byte, aes.BlockSize)
var keys = make(map[string]key)

func init() {
	// set up dbpath
	dbpath = os.Getenv("KEYBOXFILE")
	if len(dbpath) == 0 {
		exitOnError("KEYBOXFILE environment variable is not set")
	}
}

func main() {
	usage := "keybox {create | info | list | update | delete | restore | createpassword}"
	if len(os.Args) <= 1 {
		fmt.Println(usage)
		return
	}

	switch os.Args[1] {
	case "create":
		createDBFile()
	case "info":
		info()
	case "delete":
		deleteKeys()
	case "update":
		upsertKeys()
	case "list":
		showKeys()
	case "createpassword":
		fmt.Println(newPassword())
	case "restore":
		restoreDBFile()
	default:
		fmt.Printf("Unsupported command %s\n", os.Args[1])
		fmt.Println(usage)
		return
	}
}

func createDBFile() {
	passphrase := getPromptedInput("Password")
	if passphrase != getPromptedInput("Confirm Password") {
		exitOnError("Password do not match")
	} else {
		setCryptoKey(passphrase)
	}

	if stat, _ := os.Stat(dbpath); stat != nil {
		exitOnError(fmt.Sprintf("File \"%s\" already exists", dbpath))
	}

	if _, err := io.ReadFull(crand.Reader, iv); err != nil {
		exitOnError(err.Error())
	}

	keys["example"] = key{"example", "login", "password"}

	saveDBFile()
}

func info() {
	fmt.Println("Checking KEYBOXFILE environment variable...")
	finfo, err := os.Stat(dbpath)
	if os.IsNotExist(err) {
		exitOnError(fmt.Sprintf("Keybox file %s does not exit. Use create command to create one\n", dbpath))
	} else {
		fmt.Printf("Keybox file %s\n", dbpath)
		fmt.Printf("Last modified at %s\n", finfo.ModTime())
	}
}

func restoreDBFile() {
	if err := os.Rename(dbpath+".sav", dbpath); err != nil {
		exitOnError(fmt.Sprintf("Retore failed: %s\n", err))
	}
}

func upsertKeys() {
	passphrase := getPromptedInput("Password")
	setCryptoKey(passphrase)

	loadDBFile()

	// Make a copy of the db file
	if err := os.Rename(dbpath, dbpath+".sav"); err != nil {
		exitOnError(fmt.Sprintf("Cannot save to file %s.sav: %s", dbpath, err))
	}

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

	saveDBFile()
}

func showKeys() {
	passphrase := getPromptedInput("Password")
	setCryptoKey(passphrase)
	loadDBFile()

	ks := make([]string, 0, len(keys))
	for k := range keys {
		ks = append(ks, k)
	}

	sort.Strings(ks)

	cyan := color.New(color.FgCyan)
	red := color.New(color.FgRed)
	blue := color.New(color.FgBlue)
	for _, k := range ks {
		v := keys[k]
		cyan.Printf("%-20s", v.Name)
		red.Printf("%-25s", v.Login)
		blue.Println(v.Password)
	}

	red.Println("!!! DO NOT FORGET TO CLOSE THE WINDOW !!!")
}

func deleteKeys() {
	passphrase := getPromptedInput("Password")
	setCryptoKey(passphrase)

	loadDBFile()

	// make a copy
	if err := os.Rename(dbpath, dbpath+".sav"); err != nil {
		exitOnError(fmt.Sprintf("Cannot save to file %s.sav: %s", dbpath, err))
	}

	for {
		name := getPromptedInput("Name")
		if len(name) == 0 {
			break
		}
		delete(keys, name)
	}

	saveDBFile()
}

func saveDBFile() {
	f, err := os.Create(dbpath)
	if err != nil {
		exitOnError(fmt.Sprintf("Failed to create file %s", dbpath))
	} else {
		os.Chmod(dbpath, 0600)
	}

	defer f.Close()

	w := bufio.NewWriter(f)
	w.Write(iv)
	if serializedKeys, err := json.Marshal(keys); err != nil {
		exitOnError(fmt.Sprintf("Failed to marshal: %s", err.Error()))
	} else {
		encrypted, _ := encrypt(serializedKeys, cryptokey, iv)
		w.Write(encrypted)
	}
	w.Flush()
}

func loadDBFile() {
	content, err := ioutil.ReadFile(dbpath)
	if err != nil {
		exitOnError(err.Error())
	}

	if len(content) < aes.BlockSize {
		exitOnError("File corrupted")
	}

	iv := content[:aes.BlockSize]
	serializedKeys, err := decrypt(content[aes.BlockSize:], cryptokey, iv)
	if err != nil {
		exitOnError("File corrupted")
	}

	if err := json.Unmarshal(serializedKeys, &keys); err != nil {
		exitOnError("Wrong password")
	}
}

func encrypt(plaintext, key, iv []byte) (ciphertext []byte, err error) {
	// CBC mode works on blocks so plaintexts may need to be padded to the
	// next whole block. For an example of such padding, see
	// https://tools.ietf.org/html/rfc5246#section-6.2.3.2.
	for i := len(plaintext) % aes.BlockSize; i < aes.BlockSize; i++ {
		plaintext = append(plaintext, byte(0))
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

	i := len(plaintext) - 1
	for ; i > 0 && plaintext[i-1] == 0; i-- {
	}

	plaintext = plaintext[:i]
	//However, it's critical to note that ciphertexts must be authenticated (i.e. by
	// using crypto/hmac) before being decrypted in order to avoid creating
	// a padding oracle.

	return
}

func promptForKey() *key {
	name := getPromptedInput("Name")
	login := getPromptedInput("Login")
	password := getPromptedInput("Password (auto generated by enter)")

	if len(name) > 0 && len(login) > 0 {
		if len(password) == 0 {
			password = newPassword()
		}
		return &key{name, login, password}
	}

	return nil
}

func getPromptedInput(prompt string) string {
	fmt.Printf("%s: ", prompt)
	input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return input[:len(input)-1]
}

func newPassword() string {
	// 3 of each: lowercase, uppercase, special letters and numbers
	specialties := [...]byte{'!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '?', '+', '~'}

	mrand.Seed(int64(time.Now().Nanosecond()))
	p := make([]byte, 0, 12)

	for len(p) <= cap(p)-4 {
		p = append(p,
			byte(int('A')+mrand.Intn(26)), // A-Z
			byte(int('a')+mrand.Intn(26)), // a-z
			byte(int('0')+mrand.Intn(10)), // 0-9
			specialties[mrand.Intn(len(specialties))])
	}

	// shuffle
	for i := range p {
		j := mrand.Intn(len(p))
		if i != j {
			p[i], p[j] = p[j], p[i]
		}
	}

	return string(p)
}

func setCryptoKey(passphrase string) error {
	// convert a passphrase to a key, use a suitable
	// package like bcrypt or scrypt.
	h := sha256.New()
	h.Write([]byte(passphrase))
	//fmt.Printf("crytoKey: %x\n", h.Sum(nil))
	cryptokey = h.Sum(nil)
	return nil
}

func exitOnError(err string) {
	red := color.New(color.FgRed)
	red.Println(err)
	os.Exit(1)
}
