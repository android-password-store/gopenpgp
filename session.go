package pm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

//RandomToken ...
func RandomToken() ([]byte, error) {
	config := &packet.Config{DefaultCipher: packet.CipherAES256}
	keySize := config.DefaultCipher.KeySize()
	symKey := make([]byte, keySize)
	if _, err := io.ReadFull(config.Random(), symKey); err != nil {
		return nil, err
	}
	return symKey, nil
}

// RandomTokenWith ...
func RandomTokenWith(size int) ([]byte, error) {
	config := &packet.Config{DefaultCipher: packet.CipherAES256}
	symKey := make([]byte, size)
	if _, err := io.ReadFull(config.Random(), symKey); err != nil {
		return nil, err
	}
	return symKey, nil
}

//GetSessionFromKeyPacketBinkeys get session key no encoding in and out
func GetSessionFromKeyPacketBinkeys(keyPackage []byte, privateKey []byte, passphrase string) (*SessionSplit, error) {

	keyReader := bytes.NewReader(keyPackage)
	packets := packet.NewReader(keyReader)

	var p packet.Packet
	var err error
	if p, err = packets.Next(); err != nil {
		return nil, err
	}

	ek := p.(*packet.EncryptedKey)

	privKey := bytes.NewReader(privateKey)
	privKeyEntries, err := openpgp.ReadKeyRing(privKey)
	if err != nil {
		return nil, err
	}
	rawPwd := []byte(passphrase)
	var decryptErr error
	for _, key := range privKeyEntries.DecryptionKeys() {
		priv := key.PrivateKey
		if priv.Encrypted {
			if err := priv.Decrypt(rawPwd); err != nil {
				continue
			}
		}

		if decryptErr = ek.Decrypt(priv, nil); decryptErr == nil {
			break
		}
	}

	if decryptErr != nil {
		return nil, err
	}

	return getSessionSplit(ek)
}

//GetSessionFromKeyPacket get session key no encoding in and out
func GetSessionFromKeyPacket(keyPackage []byte, privateKey string, passphrase string) (*SessionSplit, error) {

	keyReader := bytes.NewReader(keyPackage)
	packets := packet.NewReader(keyReader)

	var p packet.Packet
	var err error
	if p, err = packets.Next(); err != nil {
		return nil, err
	}

	ek := p.(*packet.EncryptedKey)

	privKey := strings.NewReader(privateKey)
	privKeyEntries, err := openpgp.ReadArmoredKeyRing(privKey)
	if err != nil {
		return nil, err
	}
	rawPwd := []byte(passphrase)
	var decryptErr error
	for _, key := range privKeyEntries.DecryptionKeys() {
		priv := key.PrivateKey
		if priv.Encrypted {
			if err := priv.Decrypt(rawPwd); err != nil {
				continue
			}
		}

		if decryptErr = ek.Decrypt(priv, nil); decryptErr == nil {
			break
		}
	}

	if decryptErr != nil {
		return nil, err
	}

	return getSessionSplit(ek)
}

//KeyPacketWithPublicKey ...
func KeyPacketWithPublicKey(sessionSplit *SessionSplit, publicKey string) ([]byte, error) {
	pubkeyRaw, err := UnArmor(publicKey)
	if err != nil {
		return nil, err
	}
	return KeyPacketWithPublicKeyBin(sessionSplit, pubkeyRaw)
}

// KeyPacketWithPublicKeyBin ...
func KeyPacketWithPublicKeyBin(sessionSplit *SessionSplit, publicKey []byte) ([]byte, error) {
	publicKeyReader := bytes.NewReader(publicKey)
	pubKeyEntries, err := openpgp.ReadKeyRing(publicKeyReader)

	outbuf := &bytes.Buffer{}

	cf := cipherFunc(sessionSplit.Algo)

	if len(pubKeyEntries) == 0 {
		return nil, errors.New("cannot set key: key ring is empty")
	}

	var pub *packet.PublicKey
	for _, e := range pubKeyEntries {
		for _, subKey := range e.Subkeys {
			if !subKey.Sig.FlagsValid || subKey.Sig.FlagEncryptStorage || subKey.Sig.FlagEncryptCommunications {
				pub = subKey.PublicKey
				break
			}
		}
		if pub == nil && len(e.Identities) > 0 {
			var i *openpgp.Identity
			for _, i = range e.Identities {
				break
			}
			if i.SelfSignature.FlagsValid || i.SelfSignature.FlagEncryptStorage || i.SelfSignature.FlagEncryptCommunications {
				pub = e.PrimaryKey
			}
		}
		if pub != nil {
			break
		}
	}
	if pub == nil {
		return nil, errors.New("cannot set key: no public key available")
	}

	if err = packet.SerializeEncryptedKey(outbuf, pub, cf, sessionSplit.Session, nil); err != nil {
		err = fmt.Errorf("pmapi: cannot set key: %v", err)
		return nil, errors.New("cannot set key: key ring is empty")
	}
	return outbuf.Bytes(), nil
}

//GetSessionFromSymmetricPacket ...
func GetSessionFromSymmetricPacket(keyPackage []byte, password string) (*SessionSplit, error) {

	keyReader := bytes.NewReader(keyPackage)
	packets := packet.NewReader(keyReader)

	var symKeys []*packet.SymmetricKeyEncrypted
	for {

		var p packet.Packet
		var err error
		if p, err = packets.Next(); err != nil {
			break
		}

		switch p := p.(type) {
		case *packet.SymmetricKeyEncrypted:
			symKeys = append(symKeys, p)
		}
	}

	pwdRaw := []byte(password)
	// Try the symmetric passphrase first
	if len(symKeys) != 0 && pwdRaw != nil {
		for _, s := range symKeys {
			key, cipherFunc, err := s.Decrypt(pwdRaw)
			if err == nil {
				return &SessionSplit{
					Session: key,
					Algo:    getAlgo(cipherFunc),
				}, nil
			}

		}
	}

	return nil, errors.New("password incorrect")
}

// SymmetricKeyPacketWithPassword ...
func SymmetricKeyPacketWithPassword(sessionSplit *SessionSplit, password string) ([]byte, error) {
	outbuf := &bytes.Buffer{}

	cf := cipherFunc(sessionSplit.Algo)

	if len(password) <= 0 {
		return nil, errors.New("password can't be empty")
	}

	pwdRaw := []byte(password)

	config := &packet.Config{
		DefaultCipher: cf,
	}

	err := packet.SerializeSymmetricKeyEncryptedReuseKey(outbuf, sessionSplit.Session, pwdRaw, config)
	if err != nil {
		return nil, err
	}
	return outbuf.Bytes(), nil
}

//symKeyAlgos ...
var symKeyAlgos = map[string]packet.CipherFunction{
	"3des":   packet.Cipher3DES,
	"cast5":  packet.CipherCAST5,
	"aes128": packet.CipherAES128,
	"aes192": packet.CipherAES192,
	"aes256": packet.CipherAES256,
}

// Get cipher function.
func cipherFunc(algo string) packet.CipherFunction {
	cf, ok := symKeyAlgos[algo]
	if ok {
		return cf
	}
	return packet.CipherAES256
}

func getSessionSplit(ek *packet.EncryptedKey) (*SessionSplit, error) {
	if ek == nil {
		return nil, errors.New("can't decrypt key packet")
	}
	algo := "aes256"
	for k, v := range symKeyAlgos {
		if v == ek.CipherFunc {
			algo = k
			break
		}
	}

	if ek.Key == nil {
		return nil, errors.New("can't decrypt key packet key is nil")
	}

	return &SessionSplit{
		Session: ek.Key,
		Algo:    algo,
	}, nil
}

func getAlgo(cipher packet.CipherFunction) string {
	algo := "aes256"
	for k, v := range symKeyAlgos {
		if v == cipher {
			algo = k
			break
		}
	}

	return algo
}

//encode length based on 4.2.2. in the RFC
func encodedLength(length int) (b []byte) {
	if length < 192 {
		b = append(b, byte(length))
	} else if length < 8384 {
		length = length - 192
		b = append(b, 192+byte(length>>8))
		b = append(b, byte(length))
	} else {
		b = append(b, byte(255))
		b = append(b, byte(length>>24))
		b = append(b, byte(length>>16))
		b = append(b, byte(length>>8))
		b = append(b, byte(length))
	}
	return
}

//SeparateKeyAndData ...
func SeparateKeyAndData(encrypted string) (*EncryptedSplit, error) {

	var err error

	encryptedRaw, err := UnArmor(encrypted)
	if err != nil {
		return nil, err
	}

	encryptedReader := bytes.NewReader(encryptedRaw)

	//kr *KeyRing, r io.Reader) (key *SymmetricKey, symEncryptedData []byte,
	packets := packet.NewReader(encryptedReader)

	outSplit := &EncryptedSplit{}

	// Save encrypted key and signature apart
	var ek *packet.EncryptedKey
	// var decryptErr error
	for {
		var p packet.Packet
		if p, err = packets.Next(); err == io.EOF {
			err = nil
			break
		}
		switch p := p.(type) {
		case *packet.EncryptedKey:
			// We got an encrypted key. Try to decrypt it with each available key
			if ek != nil && ek.Key != nil {
				break
			}
			ek = p
			break
		case *packet.SymmetricallyEncrypted:
			var packetContents []byte
			if packetContents, err = ioutil.ReadAll(p.Contents); err != nil {
				return nil, err
			}

			encodedLength := encodedLength(len(packetContents) + 1)
			var symEncryptedData []byte
			symEncryptedData = append(symEncryptedData, byte(210))
			symEncryptedData = append(symEncryptedData, encodedLength...)
			symEncryptedData = append(symEncryptedData, byte(1))
			symEncryptedData = append(symEncryptedData, packetContents...)

			outSplit.DataPacket = symEncryptedData
			break

		}
	}

	var buf bytes.Buffer
	ek.Serialize(&buf)
	outSplit.KeyPacket = buf.Bytes()

	return outSplit, err
}