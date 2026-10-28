package secretbox

import (
 "crypto/aes"
 "crypto/cipher"
 "crypto/rand"
 "encoding/base64"
 "errors"
 "io"
 "os"
)

const prefix = "v1:"
const aad = "gateway-upstream-credential-v1"

func ReadKey(path string) ([]byte,error) {
 k,err:=os.ReadFile(path)
 if err!=nil{return nil,err}
 if len(k)!=32{return nil,errors.New("credential encryption key must be 32 bytes")}
 return k,nil
}
func Encrypt(key []byte, plaintext string) (string,error) {
 b,err:=aes.NewCipher(key)
 if err!=nil{return "",err}
 a,err:=cipher.NewGCM(b)
 if err!=nil{return "",err}
 nonce:=make([]byte,a.NonceSize())
 if _,err=io.ReadFull(rand.Reader,nonce);err!=nil{return "",err}
 ciphertext:=a.Seal(nonce,nonce,[]byte(plaintext),[]byte(aad))
 return prefix+base64.RawURLEncoding.EncodeToString(ciphertext),nil
}
func Decrypt(key []byte,encoded string)(string,error){
 if len(encoded)<len(prefix) || encoded[:len(prefix)]!=prefix {return "",errors.New("invalid secret version")}
 ciphertext,err:=base64.RawURLEncoding.DecodeString(encoded[len(prefix):])
 if err!=nil{return "",errors.New("invalid secret encoding")}
 b,err:=aes.NewCipher(key);if err!=nil{return "",err}
 a,err:=cipher.NewGCM(b);if err!=nil{return "",err}
 if len(ciphertext)<a.NonceSize(){return "",errors.New("short encrypted secret")}
 plain,err:=a.Open(nil,ciphertext[:a.NonceSize()],ciphertext[a.NonceSize():],[]byte(aad))
 if err!=nil{return "",errors.New("encrypted credential authentication failed")}
 return string(plain),nil
}
