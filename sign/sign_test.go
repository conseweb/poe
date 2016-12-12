package sign

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/conseweb/poe/protos"
	"github.com/spf13/viper"
)

func initConfig() error {
	viper.SetConfigName("poe")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./")
	gopath := os.Getenv("GOPATH")
	for _, p := range filepath.SplitList(gopath) {
		cfgpath := filepath.Join(p, "src", "github.com/conseweb/poe")
		viper.AddConfigPath(cfgpath)
	}
	return viper.ReadInConfig()
}

func Test_EcdsaSign(t *testing.T) {
	err := initConfig()
	if err != nil {
		t.Errorf("init config error: %v", err)
	}
	doc := protos.Document{
		Id:           "Id",
		BlockDigest:  "BlockDigest",
		SubmitTime:   time.Now().Unix(),
		Hash:         "Hash",
		ProofTime:    time.Now().Unix(),
		WaitDuration: time.Now().Unix(),
		//Metadata:     "Metadata",
		Txid: "Txid0000",
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Errorf("Encode error: %v", err)
		return
	}
	t.Logf("doc :%s \n", string(data))
	signRaw, pukRaw, err := ECDSASign(data)
	if err != nil {
		t.Errorf("ECDSASign error: %v", err)
		return
	}
	t.Logf("signRaw hex:%x \n", signRaw)
	t.Logf("pukRaw hex: %x\n", pukRaw)
	t.Log("\n -- Verify Start-- \n")
	ok, err := ECDSAVerify(pukRaw, signRaw, data)
	if err != nil {
		t.Errorf("Verify error: %v", err)
		return
	}
	if ok {
		t.Log("verify success")
	}
	t.Log("over")
}
