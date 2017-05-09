// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package integration_tests

import (
	"io/ioutil"
	"net/http"

	. "github.com/onsi/gomega"
)

var (
	brokerPath         = NewBinary("github.com/pivotal-cf/on-demand-service-broker/cmd/on-demand-service-broker")
	serviceAdapterPath = NewBinary("github.com/pivotal-cf/on-demand-service-broker/integration_tests/mock/adapter")
)

type TestSetup struct {
	credhub             Credhub
	serviceAdapterSetup func(*ServiceAdapter)
	setup               func(*BrokerEnvironment, ServiceInstanceID)
}

func When(credhub Credhub, serviceAdapterSetup func(*ServiceAdapter), setup func(*BrokerEnvironment, ServiceInstanceID)) *TestSetup {
	return &TestSetup{
		credhub:             credhub,
		serviceAdapterSetup: serviceAdapterSetup,
		setup:               setup,
	}
}

func (ts *TestSetup) brokerRespondsWith(expectedStatus int, expectedResponse string, expectedLogMessage string) {
	ts.checkBrokerResponseWhen(expectedStatus, expectedResponse, expectedLogMessage)
}

func (ts *TestSetup) checkBrokerResponseWhen(
	expectedStatus int,
	expectedResponse string,
	expectedLogMessage string,
) {
	serviceInstanceID := AServiceInstanceID()
	env := NewBrokerEnvironment(NewBosh(), NewCloudFoundry(), NewServiceAdapter(serviceAdapterPath.Path()), ts.credhub, brokerPath.Path())
	defer env.Close()

	ts.serviceAdapterSetup(env.ServiceAdapter)
	env.Start()
	ts.setup(env, serviceInstanceID)

	response := responseTo(env.Broker.CreateBindingRequest(serviceInstanceID))
	Expect(response.StatusCode).To(Equal(expectedStatus))
	Expect(bodyOf(response)).To(MatchJSON(expectedResponse))
	env.Broker.HasLogged(expectedLogMessage)

	env.Verify()
}

func responseTo(request *http.Request) *http.Response {
	response, err := http.DefaultClient.Do(request)
	Expect(err).ToNot(HaveOccurred())
	return response
}

func bodyOf(response *http.Response) []byte {
	body, err := ioutil.ReadAll(response.Body)
	Expect(err).NotTo(HaveOccurred())
	return body
}