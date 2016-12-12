package sign

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha512"
	"crypto/x509"
	"encoding/asn1"
	"io/ioutil"
	"math/big"
	"strings"
	"sync"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var (
	signLogger = logging.MustGetLogger("sign")
	once       sync.Once
	entropy    string            //影响因子,32 位随机字符
	prk        *ecdsa.PrivateKey //私钥
	puk        *ecdsa.PublicKey  //公钥
)

// Ecdsa 签名
type ecdsaSignature struct {
	R, S *big.Int
}

func initKeys() {
	var (
		prkRaw []byte
		pukRaw []byte
		obj    interface{}
		err    error
	)
	entropy = "RTZun8DdcdAlgKEP1f832fnbL43b5IqT"
	prkRaw, err = ioutil.ReadFile(viper.GetString("certs.prkFile"))
	if err != nil {
		signLogger.Fatalf("read <certs.prkFile> error: %v", err)
	}
	prk, err = x509.ParseECPrivateKey(prkRaw)
	if err != nil {
		signLogger.Fatalf("parse <certs.prkFile> error: %v", err)
	}
	pukRaw, err = ioutil.ReadFile(viper.GetString("certs.pukFile"))
	if err != nil {
		signLogger.Fatalf("read <certs.pukFile> error: %v", err)
	}
	obj, err = x509.ParsePKIXPublicKey(pukRaw)
	if err != nil {
		signLogger.Fatalf("parse <certs.pukFile> error: %v", err)
	}
	if pukIn, ok := obj.(*ecdsa.PublicKey); ok {
		puk = pukIn
	}
	if puk == nil {
		signLogger.Fatal("parse <certs.pukFile> error: is not vaild.")
	}
}

// ecdsa 签名
// msg : 明文字节数组
// returns
// 1.签名后的字节数组
// 2.公匙字节数组
// 3.错误信息
func ECDSASign(msg []byte) ([]byte, []byte, error) {
	once.Do(initKeys)
	hashInstance := sha512.New()
	hashInstance.Write(msg)
	hashBytes := bytes.NewBuffer(hashInstance.Sum(nil)).Bytes()
	r, s, err := ecdsa.Sign(strings.NewReader(entropy), prk, hashBytes)
	if err != nil {
		signLogger.Debugf("in ecdsa func <ECDSASign> Sign error : %v", err)
		return nil, nil, err
	}
	signRaw, err := asn1.Marshal(ecdsaSignature{r, s})
	if err != nil {
		signLogger.Debugf("in ecdsa func <ECDSASign> ecdsaSignature Marshal error : %v", err)
		return nil, nil, err
	}
	pukRaw := elliptic.Marshal(puk.Curve, puk.X, puk.Y)
	return signRaw, pukRaw, nil
}

// GetPublicKey gets public key
func GetPublicKey() []byte {
	return elliptic.Marshal(puk.Curve, puk.X, puk.Y)
}

// ecdsa 验证
// keyRaw: 公钥字节数组
// signRaw: 签名信息字节数组
// msg : 明文字节数组
// returns
// 1.验证结果，布尔值
// 2.错误信息
func ECDSAVerify(keyRaw, signRaw, msg []byte) (bool, error) {
	hashInstance := sha512.New()
	signature := ecdsaSignature{}
	key := ecdsa.PublicKey{
		Curve: elliptic.P521(),
	}
	key.X, key.Y = elliptic.Unmarshal(key.Curve, keyRaw)
	_, err := asn1.Unmarshal(signRaw, &signature)
	if err != nil {
		return false, err
	}
	hashInstance.Write(msg)
	hashBytes := bytes.NewBuffer(hashInstance.Sum(nil)).Bytes()
	return ecdsa.Verify(&key, hashBytes, signature.R, signature.S), nil
}
