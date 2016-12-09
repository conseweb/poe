package sign

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha512"
	"encoding/asn1"
	"math/big"
	"strings"

	"github.com/op/go-logging"
)

var (
	signLogger = logging.MustGetLogger("sign")
	entropy    string            //影响因子,32 位随机字符
	uniqueId   string            //密钥唯一标识，224、256、384、521 位随机字符
	prk        *ecdsa.PrivateKey //私钥
	puk        *ecdsa.PublicKey  //公钥
	curve      elliptic.Curve    //椭圆曲线
)

// Ecdsa 签名
type ecdsaSignature struct {
	R, S *big.Int
}

func init() {
	var err error
	curve = elliptic.P521()
	entropy = "RTZun8DdcdAlgKEP1f832fnbL43b5IqT"
	uniqueId = "r4E43BLX3WsOT4u3OOyjNcncQC1M1R4y27896T95Hwv0522W5M8YLMXSk3I8ItTvP2A2B9Q5Iad7RJU6iFEr3Y1sUb16738VU31l4CtqjLboQ9Gw088j78sIvQh8B1zS9P3uEWwTp829bEzg0x55WSxevVE8EZ6eMH6TspJP3c038h2aI1BoWx2XBSDw4BpN6Jfx8H2IR8HvqCk1jLn2F0wDuV432n5i"
	prk, err = ecdsa.GenerateKey(curve, strings.NewReader(uniqueId))
	if err != nil {
		signLogger.Debugf("in ecdsa func <init> GenerateKey error : %v", err)
		return
	}
	puk = &prk.PublicKey
}

// ecdsa 签名
// msg : 明文字节数字
// returns
// 1.签名后的字节数组
// 2.公匙字节数组
// 3.错误信息
func ECDSASign(msg []byte) ([]byte, []byte, error) {
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
