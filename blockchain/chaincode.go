package blockchain

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/conseweb/common/crypto"
	poepb "github.com/conseweb/poe/protos"
	fabricpb "github.com/hyperledger/fabric/protos"
	"golang.org/x/crypto/sha3"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type (
	items struct {
		lock *sync.RWMutex
		data map[string]interface{}
	}
	eccAdapter struct {
		sender *Blockchain
		ecc    chan *fabricpb.Event_ChaincodeEvent
	}
	queryResult struct {
		// 键值
		Key string `json:"key,omitempty"`
		// 键哈希值
		HashKey string `json:"hash_key,omitempty"`
		// 是否存在
		Exist bool `json:"exist,omitempty"`
	}
)

func (self *items) Get(key string) interface{} {
	self.lock.RLock()
	defer self.lock.RUnlock()
	if v, ok := self.data[key]; ok {
		return v
	}
	return nil
}

func (self *items) Set(key string, val interface{}) bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	if v, ok := self.data[key]; !ok {
		self.data[key] = val
	} else if v != val {
		self.data[key] = val
	} else {
		return false
	}
	return true
}

func (self *items) Delete(key string) {
	self.lock.Lock()
	defer self.lock.Unlock()
	delete(self.data, key)
}

func (self *items) Clear() {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.data = make(map[string]interface{})
}

func newEccAdapter(sender *Blockchain) *eccAdapter {
	ecapt := eccAdapter{}
	ecapt.sender = sender
	ecapt.ecc = make(chan *fabricpb.Event_ChaincodeEvent)
	return &ecapt
}
func (adapter *eccAdapter) GetInterestedEvents() ([]*fabricpb.Interest, error) {
	eccReg := fabricpb.ChaincodeReg{}
	eccReg.ChaincodeID = adapter.sender.chainCodeId
	eccReg.EventName = "invoke_completed"
	eccInReg := fabricpb.Interest_ChaincodeRegInfo{}
	eccInReg.ChaincodeRegInfo = &eccReg
	eccIn := fabricpb.Interest{}
	eccIn.EventType = fabricpb.EventType_CHAINCODE
	eccIn.RegInfo = &eccInReg
	return []*fabricpb.Interest{&eccIn}, nil
}

func (adapter *eccAdapter) Recv(msg *fabricpb.Event) (bool, error) {
	event, ok := msg.Event.(*fabricpb.Event_ChaincodeEvent)
	if !ok {
		return false, nil
	}
	if event.ChaincodeEvent.ChaincodeID != adapter.sender.chainCodeId {
		return false, nil
	}
	if event.ChaincodeEvent.EventName != "invoke_completed" {
		return false, nil
	}
	adapter.ecc <- event
	return true, nil
}

func (adapter *eccAdapter) Disconnected(e error) {
	if e == nil {
		return
	}
	blockchainLogger.Debugf("in adapter func <Disconnected> error: %v", e)
}

func (bc *Blockchain) toSpec(function string, args []string) *fabricpb.ChaincodeSpec {
	input := make([][]byte, 1, len(args)+1)
	input[0] = []byte(function)
	for _, v := range args {
		input = append(input, []byte(v))
	}
	spec := &fabricpb.ChaincodeSpec{
		Type: fabricpb.ChaincodeSpec_Type(fabricpb.ChaincodeSpec_Type_value["GOLANG"]),
		ChaincodeID: &fabricpb.ChaincodeID{
			Path: bc.path,
			Name: bc.chainCodeId,
		},
		CtorMsg: &fabricpb.ChaincodeInput{
			Args: input,
		},
		SecureContext: bc.secureCtx,
	}
	return spec
}

func (bc *Blockchain) execute(method, function string, args []string) ([]byte, error) {
	var (
		conn      *grpc.ClientConn
		devopsCli fabricpb.DevopsClient
		spec      = bc.toSpec(function, args)
		resp      *fabricpb.Response
		e         error
	)
	blockchainLogger.Debugf("in bc func <execute> spec: %v", spec)
	for i := 0; i < bc.peerBackend.Len(); i++ {
		conn, e = bc.getGrpcConn(bc.peerBackend.Choose().String())
		if e != nil {
			blockchainLogger.Warningf("in bc func <execute> error: %v", e)
		}
		blockchainLogger.Debugf("in bc func <execute> peerBackend index : %v", i)
		if conn != nil {
			break
		}
	}
	if conn == nil {
		blockchainLogger.Debug("in bc func <execute> create grpc client conn valid")
		return nil, errors.New("create grpc client conn valid")
	}
	defer conn.Close()
	devopsCli = fabricpb.NewDevopsClient(conn)
	switch method {
	case "invoke":
		resp, e = devopsCli.Invoke(context.Background(), &fabricpb.ChaincodeInvocationSpec{ChaincodeSpec: spec})
	case "query":
		resp, e = devopsCli.Query(context.Background(), &fabricpb.ChaincodeInvocationSpec{ChaincodeSpec: spec})
	}
	if e != nil {
		return nil, e
	}
	if resp == nil {
		blockchainLogger.Debug("in bc func <execute> resp is nil")
		return nil, errors.New("resp is nil")
	}
	blockchainLogger.Debug("in bc func <execute> exec over")
	return resp.Msg, e
}

// 启动事件监听
func (bc *Blockchain) eventStart() {
	var (
		adapter = eccAdapter{sender: bc, ecc: make(chan *fabricpb.Event_ChaincodeEvent)}
		ecli    *eventsClient
		conn    *grpc.ClientConn
		e       error
	)
	for _, addr := range bc.events {
		conn, e = bc.getGrpcConn(addr)
		if e != nil {
			blockchainLogger.Warningf("in bc func <eventStart> error: %v", e)
			continue
		}
		ecli = newEventsClient(conn, &adapter, bc.regTimeout)
		if e = ecli.Start(); e != nil {
			blockchainLogger.Warningf("in bc func <eventStart> error: %v", e)
			continue
		}
		bc.eClis = append(bc.eClis, ecli)
	}
	if len(bc.eClis) == 0 {
		blockchainLogger.Debug("in bc func <eventStart> event client valid")
		return
	}
	for {
		select {
		case ecc := <-adapter.ecc:
			go invokeCompleted(adapter.sender, ecc)
		}
	}
}

func (bc *Blockchain) Close() error {
	if len(bc.eClis) == 0 {
		return nil
	}
	for _, ecli := range bc.eClis {
		ecli.Stop()
	}
	bc.items.Clear()
	blockchainLogger.Debug("in bc func <Close> exec over")
	return nil
}

func (bc *Blockchain) getGrpcConn(addr string) (*grpc.ClientConn, error) {
	var (
		conn *grpc.ClientConn
		opts []grpc.DialOption = []grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithTimeout(bc.regTimeout),
			grpc.WithInsecure(),
		}
	)
	for i := 0; i < bc.failOver; i++ {
		conn, _ = grpc.Dial(addr, opts...)
		if conn != nil {
			break
		}
	}
	if conn == nil {
		blockchainLogger.Debug("in bc func <getGrpcConn> error : Could not create client conn to %s", addr)
		return nil, fmt.Errorf("Could not create client conn to %s", addr)
	}
	blockchainLogger.Info("in bc func <getGrpcConn> exec over")
	return conn, nil
}

// invoke_completed 事件响应处理
func invokeCompleted(sender *Blockchain, e *fabricpb.Event_ChaincodeEvent) error {
	blockchainLogger.Debugf("blockchain event: %v", e)
	obj := sender.items.Get(e.ChaincodeEvent.TxID)
	if obj == nil {
		blockchainLogger.Warningf("txID %s has no documents stored", e.ChaincodeEvent.TxID)
		return fmt.Errorf("not found stored documents")
	}
	blockchainLogger.Debugf("txID: %s, documents: %v", e.ChaincodeEvent.TxID, obj)
	docs, ok := obj.([]*poepb.Document)
	if !ok {
		return fmt.Errorf("invalid stored type, should be a slice of documents pointer")
	}
	data := strings.Split(string(e.ChaincodeEvent.Payload), ",")
	if len(data) == 0 || data[0] == "" {
		return fmt.Errorf("empty chaincode event payload")
	}
	proofKey := fmt.Sprintf("%x", crypto.Hash(sha3.New512(), []byte(data[0])))
	docIds := make([]string, len(docs))
	for idx, doc := range docs {
		docIds[idx] = doc.Id
	}
	blockchainLogger.Debugf("proofKey: %s", proofKey)
	sender.persister.SetDocsBlockDigest(docIds, proofKey)
	sender.items.Delete(e.ChaincodeEvent.TxID)
	return nil
}
