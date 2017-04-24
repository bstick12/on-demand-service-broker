// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package boshclient_test

import (
	"crypto/x509"
	"encoding/pem"
	"log"

	"github.com/pivotal-cf/on-demand-service-broker/boshclient"
	"github.com/pivotal-cf/on-demand-service-broker/boshclient/fakes"
	"github.com/pivotal-cf/on-demand-service-broker/mockbosh"
	"github.com/pivotal-cf/on-demand-service-broker/mockhttp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

const (
	expectedAuthHeader = "some-auth-header"
)

var (
	c *boshclient.Client

	authHeaderBuilder *fakes.FakeAuthHeaderBuilder
	director          *mockhttp.Server
	logger            *log.Logger
)

var _ = BeforeEach(func() {
	authHeaderBuilder = new(fakes.FakeAuthHeaderBuilder)
	authHeaderBuilder.BuildReturns(expectedAuthHeader, nil)
	director = mockbosh.New()
	director.ExpectedAuthorizationHeader(expectedAuthHeader)
	logger = log.New(GinkgoWriter, "[boshclient unit test]", log.LstdFlags)
})

var _ = AfterEach(func() {
	director.VerifyMocks()
	director.Close()
})

var _ = JustBeforeEach(func() {
	var certPEM []byte
	if director.TLS != nil {
		cert, err := x509.ParseCertificate(director.TLS.Certificates[0].Certificate[0])
		Expect(err).NotTo(HaveOccurred())
		certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	}
	var err error
	c, err = boshclient.New(director.URL, authHeaderBuilder, false, certPEM)
	Expect(err).NotTo(HaveOccurred())
	c.BoshPollingInterval = 0
})

func TestBoshclient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Boshclient Suite")
}
