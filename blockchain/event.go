package blockchain

import (
	"fmt"
	"io"
	"sync"
	"time"

	fabricpb "github.com/hyperledger/fabric/protos"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type (
	eventAdapter interface {
		GetInterestedEvents() ([]*fabricpb.Interest, error)
		Recv(msg *fabricpb.Event) (bool, error)
		Disconnected(err error)
	}
	eventsClient struct {
		sync.RWMutex
		conn       *grpc.ClientConn
		regTimeout time.Duration
		stream     fabricpb.Events_ChatClient
		adapter    eventAdapter
	}
)

func newEventsClient(conn *grpc.ClientConn, adapter eventAdapter, regTimeout time.Duration) *eventsClient {
	ecli := eventsClient{}
	ecli.RWMutex = sync.RWMutex{}
	ecli.conn = conn
	ecli.regTimeout = regTimeout
	ecli.adapter = adapter
	return &ecli
}

func (ec *eventsClient) send(emsg *fabricpb.Event) error {
	ec.Lock()
	defer ec.Unlock()
	return ec.stream.Send(emsg)
}

// RegisterAsync - registers interest in a event and doesn't wait for a response
func (ec *eventsClient) RegisterAsync(ies []*fabricpb.Interest) error {
	emsg := &fabricpb.Event{Event: &fabricpb.Event_Register{Register: &fabricpb.Register{Events: ies}}}
	var err error
	if err = ec.send(emsg); err != nil {
		fmt.Printf("error on Register send %s\n", err)
	}
	return err
}

// register - registers interest in a event
func (ec *eventsClient) register(ies []*fabricpb.Interest) error {
	var err error
	if err = ec.RegisterAsync(ies); err != nil {
		return err
	}
	regChan := make(chan struct{})
	go func() {
		defer close(regChan)
		in, inerr := ec.stream.Recv()
		if inerr != nil {
			err = inerr
			return
		}
		switch in.Event.(type) {
		case *fabricpb.Event_Register:
		case nil:
			err = fmt.Errorf("invalid nil object for register")
		default:
			err = fmt.Errorf("invalid registration object")
		}
	}()
	select {
	case <-regChan:
	case <-time.After(ec.regTimeout):
		err = fmt.Errorf("timeout waiting for registration")
	}
	return err
}

// UnregisterAsync - Unregisters interest in a event and doesn't wait for a response
func (ec *eventsClient) UnregisterAsync(ies []*fabricpb.Interest) error {
	emsg := &fabricpb.Event{Event: &fabricpb.Event_Unregister{Unregister: &fabricpb.Unregister{Events: ies}}}
	var err error
	if err = ec.send(emsg); err != nil {
		err = fmt.Errorf("error on unregister send %s\n", err)
	}
	return err
}

// unregister - unregisters interest in a event
func (ec *eventsClient) unregister(ies []*fabricpb.Interest) error {
	var err error
	if err = ec.UnregisterAsync(ies); err != nil {
		return err
	}
	regChan := make(chan struct{})
	go func() {
		defer close(regChan)
		in, inerr := ec.stream.Recv()
		if inerr != nil {
			err = inerr
			return
		}
		switch in.Event.(type) {
		case *fabricpb.Event_Unregister:
		case nil:
			err = fmt.Errorf("invalid nil object for unregister")
		default:
			err = fmt.Errorf("invalid unregistration object")
		}
	}()
	select {
	case <-regChan:
	case <-time.After(ec.regTimeout):
		err = fmt.Errorf("timeout waiting for unregistration")
	}
	return err
}

// Recv recieves next event - use when client has not called Start
func (ec *eventsClient) Recv() (*fabricpb.Event, error) {
	in, err := ec.stream.Recv()
	if err == io.EOF {
		// read done.
		if ec.adapter != nil {
			ec.adapter.Disconnected(nil)
		}
		return nil, err
	}
	if err != nil {
		if ec.adapter != nil {
			ec.adapter.Disconnected(err)
		}
		return nil, err
	}
	return in, nil
}
func (ec *eventsClient) processEvents() error {
	defer ec.stream.CloseSend()
	for {
		in, err := ec.stream.Recv()
		if err == io.EOF {
			// read done.
			if ec.adapter != nil {
				ec.adapter.Disconnected(nil)
			}
			return nil
		}
		if err != nil {
			if ec.adapter != nil {
				ec.adapter.Disconnected(err)
			}
			return err
		}
		if ec.adapter != nil {
			cont, err := ec.adapter.Recv(in)
			if !cont {
				return err
			}
		}
	}
}

//Start establishes connection with Event hub and registers interested events with it
func (ec *eventsClient) Start() error {
	ies, err := ec.adapter.GetInterestedEvents()
	if err != nil {
		return fmt.Errorf("error getting interested events:%s", err)
	}
	if len(ies) == 0 {
		return fmt.Errorf("must supply interested events")
	}
	serverClient := fabricpb.NewEventsClient(ec.conn)
	ec.stream, err = serverClient.Chat(context.Background())
	if err != nil {
		return err
	}
	if err = ec.register(ies); err != nil {
		return err
	}
	go ec.processEvents()
	return nil
}

//Stop terminates connection with event hub
func (ec *eventsClient) Stop() error {
	if ec.stream == nil {
		// in case the steam/chat server has not been established earlier, we assume that it's closed, successfully
		return nil
	}
	return ec.stream.CloseSend()
}
