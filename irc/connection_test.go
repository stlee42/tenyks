package irc

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kyleterry/tenyks/config"
	"github.com/kyleterry/tenyks/mockirc"
)

func TestNewConnectionNoDial(t *testing.T) {
	conf := config.ConnectionConfig{
		Name: "test",
		Ssl:  true,
	}
	conn := NewConnection(conf.Name, conf)
	if conn.Name != conf.Name {
		t.Errorf("Expected %s, got %s", conn.Name, conf.Name)
	}

	if !conn.usingSSL {
		t.Error("SSL is supposed to be enabled")
	}

	if conn.IsConnected() {
		t.Error("Connection is not supposed to be connected")
	}

	strMethodResult := fmt.Sprintf("%s", conn)
	if !strings.Contains(strMethodResult, "Disconnected") {
		t.Error("String method seems to be broken, Expected to contain 'Disconnected', got ", strMethodResult)
	}

	select {
	case <-conn.ConnectWait:
		t.Error("Channel is supposed to remain open and not recieve")
	case <-time.After(time.Second):
		break
	}
}

func MakeConnConfig() config.ConnectionConfig {
	return config.ConnectionConfig{
		Name:            "mockirc",
		Host:            "localhost",
		Port:            26661,
		FloodProtection: true,
		Nicks:           []string{"tenyks", "tenyks-"},
		Ident:           "something",
		Realname:        "tenyks",
		Admins:          []string{"kyle"},
		Channels:        []string{"#tenyks", "#test"},
	}
}

func TestCanConnectAndDisconnect(t *testing.T) {
	var wait chan bool
	var err error
	ircServer := mockirc.New("mockirc.tenyks.io", 26661)
	wait, err = ircServer.Start()
	if err != nil {
		t.Fatal("Expected nil", "got", err)
	}
	<-wait

	conn := NewConnection("mockirc", MakeConnConfig())
	wait = conn.Connect()
	<-wait

	if !conn.IsConnected() {
		t.Error("Expected", true, "got", false)
	}

	conn.Disconnect()

	if conn.IsConnected() {
		t.Error("Expected", false, "got", true)
	}

	err = ircServer.Stop()
	if err != nil {
		t.Fatal("Error stopping mockirc server")
	}
}

func TestCanHandshakeAndWorkWithIRC(t *testing.T) {
	var wait chan bool
	var err error
	ircServer := mockirc.New("mockirc.tenyks.io", 26661)
	ircServer.When("USER tenyks localhost something :tenyks").Respond(":101 :Welcome")
	ircServer.When("PING ").Respond(":PONG")
	wait, err = ircServer.Start()
	if err != nil {
		t.Fatal("Expected nil", "got", err)
	}
	<-wait

	conn := NewConnection("mockirc", MakeConnConfig())
	wait = conn.Connect()
	<-wait

	conn.BootstrapHandler(nil)
	<-conn.ConnectWait

	msg := string(<-conn.In)
	if msg != ":101 :Welcome\r\n" {
		t.Error("Expected :101 :Welcome", "got", msg)
	}

	conn.SendPing(nil)
	select {
	case msg := <-conn.In:
		if msg != ":PONG\r\n" {
			t.Error("Expected", ":PONG mockirc", "got", msg)
		}
	case <-time.After(time.Second * 5):
		t.Error("Timed out before getting a response back from mockirc")
	}

	conn.Disconnect()

	err = ircServer.Stop()
	if err != nil {
		t.Fatal("Error stopping mockirc server")
	}
}
