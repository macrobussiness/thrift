/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements. See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership. The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License. You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package tests

import (
	"MuliplexedProtocolTest"
	"net"
	"testing"
	"thrift"
	"time"
)

func FindAvailableTCPServerPort() net.Addr {
	if l, err := net.Listen("tcp", "127.0.0.1:0"); err != nil {
		panic("Could not find available server port")
	} else {
		defer l.Close()
		return l.Addr()
	}
}

type FirstImpl struct{}

func (f *FirstImpl) ReturnOne() (r int64, err error) {
	return 1, nil
}

type SecondImpl struct{}

func (s *SecondImpl) ReturnTwo() (r int64, err error) {
	return 2, nil
}

const TIMEOUT = time.Second

var addr net.Addr
var server *thrift.TSimpleServer

var processor = thrift.NewTMultiplexedProcessor()

func TestInitTwoServers(t *testing.T) {
	var err error
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory := thrift.NewTTransportFactory()
	transportFactory = thrift.NewTFramedTransportFactory(transportFactory)
	addr = FindAvailableTCPServerPort()
	serverTransport, err := thrift.NewTServerSocketTimeout(addr.String(), TIMEOUT)
	if err != nil {
		t.Fatal("Unable to create server socket", err)
	}
	server = thrift.NewTSimpleServer4(processor, serverTransport, transportFactory, protocolFactory)

	firstProcessor := MuliplexedProtocolTest.NewFirstProcessor(&FirstImpl{})
	processor.RegisterProcessor("FirstService", firstProcessor)

	secondProcessor := MuliplexedProtocolTest.NewSecondProcessor(&SecondImpl{})
	processor.RegisterProcessor("SecondService", secondProcessor)

	go server.Serve()
}

var firstClient *MuliplexedProtocolTest.FirstClient

func TestInitClient1(t *testing.T) {
	socket := thrift.NewTSocketFromAddrTimeout(addr, TIMEOUT)
	transport := thrift.NewTFramedTransport(socket)
	var protocol thrift.TProtocol = thrift.NewTBinaryProtocolTransport(transport)
	protocol = thrift.NewTMultiplexedProtocol(protocol, "FirstService")
	firstClient = MuliplexedProtocolTest.NewFirstClientProtocol(transport, protocol, protocol)
	err := transport.Open()
	if err != nil {
		t.Fatal("Unable to open client socket", err)
	}
}

var secondClient *MuliplexedProtocolTest.SecondClient

func TestInitClient2(t *testing.T) {
	socket := thrift.NewTSocketFromAddrTimeout(addr, TIMEOUT)
	transport := thrift.NewTFramedTransport(socket)
	var protocol thrift.TProtocol = thrift.NewTBinaryProtocolTransport(transport)
	protocol = thrift.NewTMultiplexedProtocol(protocol, "SecondService")
	secondClient = MuliplexedProtocolTest.NewSecondClientProtocol(transport, protocol, protocol)
	err := transport.Open()
	if err != nil {
		t.Fatal("Unable to open client socket", err)
	}
}

//create client without service prefix
func createLegacyClient(t *testing.T) *MuliplexedProtocolTest.SecondClient {
	socket := thrift.NewTSocketFromAddrTimeout(addr, TIMEOUT)
	transport := thrift.NewTFramedTransport(socket)
	var protocol thrift.TProtocol = thrift.NewTBinaryProtocolTransport(transport)
	legacyClient := MuliplexedProtocolTest.NewSecondClientProtocol(transport, protocol, protocol)
	err := transport.Open()
	if err != nil {
		t.Fatal("Unable to open client socket", err)
	}
	return legacyClient
}

func TestCallFirst(t *testing.T) {
	ret, err := firstClient.ReturnOne()
	if err != nil {
		t.Fatal("Unable to call first server:", err)
	}
	if ret != 1 {
		t.Fatal("Unexpected result from server: ", ret)
	}
}

func TestCallSecond(t *testing.T) {
	ret, err := secondClient.ReturnTwo()
	if err != nil {
		t.Fatal("Unable to call second server:", err)
	}
	if ret != 2 {
		t.Fatal("Unexpected result from server: ", ret)
	}
}

func TestCallLegacy(t *testing.T) {
	legacyClient := createLegacyClient(t)
	ret, err := legacyClient.ReturnTwo()
	//expect error since default processor is not registered
	if err == nil {
		t.Fatal("Expecting error")
	}
	//register default processor and call again
	processor.RegisterDefault(MuliplexedProtocolTest.NewSecondProcessor(&SecondImpl{}))
	legacyClient = createLegacyClient(t)
	ret, err = legacyClient.ReturnTwo()
	if err != nil {
		t.Fatal("Unable to call legacy server:", err)
	}
	if ret != 2 {
		t.Fatal("Unexpected result from server: ", ret)
	}
}

func TestShutdownServerAndClients(t *testing.T) {
	firstClient.Transport.Close()
	secondClient.Transport.Close()
	server.Stop()
}