package GoogleIdTokenVerifier

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// Certs is
type Certs struct {
	Keys []keys `json:"keys"`
}

type keys struct {
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	Kid string `json:"Kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// TokenInfo is
type TokenInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	AtHash        string `json:"at_hash"`
	Aud           string `json:"aud"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Local         string `json:"locale"`
	Iss           string `json:"iss"`
	Azp           string `json:"azp"`
	Iat           int64  `json:"iat"`
	Exp           int64  `json:"exp"`
}

// https://developers.google.com/identity/sign-in/web/backend-auth
// https://github.com/google/oauth2client/blob/master/oauth2client/crypt.py

// VerifyGoogleIDToken is
func VerifyGoogleIDToken(authToken string, certs *Certs, aud string) *TokenInfo {
	header, payload, signature, messageToSign := divideAuthToken(authToken)

	tokeninfo := getTokenInfo(payload)
	var niltokeninfo *TokenInfo
	//fmt.Println(tokeninfo)
	if aud != tokeninfo.Aud {
		err := errors.New("Token is not valid, Audience from token and certificate don't match")
		fmt.Printf("Error verifying key %s\n", err.Error())
		return niltokeninfo
	}
	if (tokeninfo.Iss != "accounts.google.com") && (tokeninfo.Iss != "https://accounts.google.com") {
		err := errors.New("Token is not valid, ISS from token and certificate don't match")
		fmt.Printf("Error verifying key %s\n", err.Error())
		return niltokeninfo
	}
	if !checkTime(tokeninfo) {
		err := errors.New("Token is not valid, Token is expired.")
		fmt.Printf("Error verifying key %s\n", err.Error())
		return niltokeninfo
	}

	key, err := choiceKeyByKeyID(certs.Keys, getAuthTokenKeyID(header))
	if err != nil {
		fmt.Printf("Error verifying key %s\n", err.Error())
		return niltokeninfo
	}
	pKey := rsa.PublicKey{N: a5(urlsafeB64decode(key.N)), E: a4(a2(urlsafeB64decode(key.E)))}
	err = rsa.VerifyPKCS1v15(&pKey, crypto.SHA256, messageToSign, signature)
	if err != nil {
		fmt.Printf("Error verifying key %s\n", err.Error())
		return niltokeninfo
	}
	return tokeninfo
}

func getTokenInfo(bt []byte) *TokenInfo {
	var a *TokenInfo
	json.Unmarshal(bt, &a)
	return a
}

func checkTime(tokeninfo *TokenInfo) bool {
	if (time.Now().Unix() < tokeninfo.Iat) || (time.Now().Unix() > tokeninfo.Exp) {
		return false
	}
	return true
}

//GetCertsFromURL is
func GetCertsFromURL() []byte {
	res, _ := http.Get("https://www.googleapis.com/oauth2/v3/certs")
	certs, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()
	return certs
}

//GetCerts is
func GetCerts(bt []byte) *Certs {
	var certs *Certs
	json.Unmarshal(bt, &certs)
	return certs
}

func urlsafeB64decode(str string) []byte {
	if m := len(str) % 4; m != 0 {
		str += strings.Repeat("=", 4-m)
	}
	bt, _ := base64.URLEncoding.DecodeString(str)
	return bt
}

func choiceKeyByKeyID(a []keys, tknkid string) (keys, error) {
	for i:= 0; i < len(a); i++ {
		if a[i].Kid == tknkid {
			return a[i], nil
		}
	}
	err := errors.New("Token is not valid, kid from token and certificate don't match")
	var b keys
	return b, err
}

func getAuthTokenKeyID(bt []byte) string {
	var keys keys
	json.Unmarshal(bt, &keys)
	return keys.Kid
}

func divideAuthToken(str string) ([]byte, []byte, []byte, []byte) {
	args := strings.Split(str, ".")
	return urlsafeB64decode(args[0]), urlsafeB64decode(args[1]), urlsafeB64decode(args[2]), a3(args[0] + "." + args[1])
}

func a2(bt []byte) *bytes.Reader {
	var bt2 []byte
	if len(bt) < 8 {
		bt2 = make([]byte, 8-len(bt), 8)
		bt2 = append(bt2, bt...)
	} else {
		bt2 = bt
	}
	return bytes.NewReader(bt2)
}

func a3(str string) []byte {
	a := sha256.New()
	a.Write([]byte(str))
	return a.Sum(nil)
}

func a4(a io.Reader) int {
	var e uint64
	binary.Read(a, binary.BigEndian, &e)
	return int(e)
}

func a5(bt []byte) *big.Int {
	a := big.NewInt(0)
	a.SetBytes(bt)
	return a
}
