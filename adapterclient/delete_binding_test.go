// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package adapterclient_test

import (
	"encoding/json"
	"errors"
	"io"
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/on-demand-service-broker/adapterclient"
	"github.com/pivotal-cf/on-demand-service-broker/adapterclient/fake_command_runner"
	"github.com/pivotal-cf/on-demand-services-sdk/bosh"
	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"
)

var _ = Describe("external service adapter", func() {
	const externalBinPath = "/thing"

	var (
		a                  *adapterclient.Adapter
		cmdRunner          *fake_command_runner.FakeCommandRunner
		logs               *gbytes.Buffer
		logger             *log.Logger
		bindingID          string
		deploymentTopology bosh.BoshVMs
		manifest           []byte
		requestParams      map[string]interface{}

		deleteBindingError error
	)

	BeforeEach(func() {
		logs = gbytes.NewBuffer()
		logger = log.New(io.MultiWriter(GinkgoWriter, logs), "[unit-tests] ", log.LstdFlags)
		cmdRunner = new(fake_command_runner.FakeCommandRunner)
		a = &adapterclient.Adapter{
			CommandRunner:   cmdRunner,
			ExternalBinPath: externalBinPath,
		}
		cmdRunner.RunReturns([]byte(""), []byte(""), intPtr(adapterclient.SuccessExitCode), nil)

		bindingID = "the-binding"
		deploymentTopology = bosh.BoshVMs{}
		manifest = []byte("a-manifest")
		requestParams = map[string]interface{}{
			"plan_id":    "some-plan-id",
			"service_id": "some-service-id",
		}
	})

	JustBeforeEach(func() {
		deleteBindingError = a.DeleteBinding(bindingID, deploymentTopology, manifest, requestParams, logger)
	})

	It("invokes external executable with params to delete binding", func() {
		serialisedBoshVMs, err := json.Marshal(deploymentTopology)
		Expect(err).NotTo(HaveOccurred())

		serialisedRequestParams, err := json.Marshal(requestParams)
		Expect(err).NotTo(HaveOccurred())

		Expect(cmdRunner.RunCallCount()).To(Equal(1))
		argsPassed := cmdRunner.RunArgsForCall(0)
		Expect(argsPassed).To(ConsistOf(externalBinPath, "delete-binding", bindingID, string(serialisedBoshVMs), string(manifest), string(serialisedRequestParams)))
	})

	Context("when the external adapter succeeds", func() {
		It("returns no error", func() {
			Expect(deleteBindingError).ToNot(HaveOccurred())
		})
	})

	Context("when the external adapter fails with no exit code", func() {
		BeforeEach(func() {
			cmdRunner.RunReturns(nil, nil, nil, errors.New("oops"))
		})

		It("returns an error", func() {
			Expect(deleteBindingError).To(MatchError("an error occurred running external service adapter at /thing: 'oops'. stdout: '', stderr: ''"))
		})
	})

	Context("when the external adapter fails", func() {
		Context("when there is a operator error message and a user error message", func() {
			BeforeEach(func() {
				cmdRunner.RunReturns([]byte("I'm stdout"), []byte("I'm stderr"), intPtr(serviceadapter.ErrorExitCode), nil)
			})

			It("returns an UnknownFailureError", func() {
				commandError, ok := deleteBindingError.(adapterclient.UnknownFailureError)
				Expect(ok).To(BeTrue(), "error should be a Generic Error")
				Expect(commandError.Error()).To(Equal("I'm stdout"))
			})

			It("logs a message to the operator", func() {
				Expect(logs).To(gbytes.Say("external service adapter exited with 1 at /thing: stdout: 'I'm stdout', stderr: 'I'm stderr'\n"))
			})
		})
	})

	Context("when the external adapter fails with exit code 10", func() {
		BeforeEach(func() {
			cmdRunner.RunReturns([]byte("I'm stdout"), []byte("I'm stderr"), intPtr(serviceadapter.NotImplementedExitCode), nil)
		})

		It("returns an error", func() {
			Expect(deleteBindingError).To(BeAssignableToTypeOf(adapterclient.NotImplementedError{}))
			Expect(deleteBindingError.Error()).NotTo(ContainSubstring("stdout"))
			Expect(deleteBindingError.Error()).NotTo(ContainSubstring("stderr"))
		})

		It("logs a message to the operator", func() {
			Expect(logs).To(gbytes.Say("external service adapter exited with 10 at /thing: stdout: 'I'm stdout', stderr: 'I'm stderr'\n"))
		})
	})

	Context("when the external adapter fails with exit code 41", func() {
		BeforeEach(func() {
			cmdRunner.RunReturns([]byte("I'm stdout"), []byte("I'm stderr"), intPtr(serviceadapter.BindingNotFoundErrorExitCode), nil)
		})

		It("returns an error", func() {
			Expect(deleteBindingError).To(BeAssignableToTypeOf(adapterclient.BindingNotFoundError{}))
			Expect(deleteBindingError.Error()).NotTo(ContainSubstring("stdout"))
			Expect(deleteBindingError.Error()).NotTo(ContainSubstring("stderr"))
		})

		It("logs a message to the operator", func() {
			Expect(logs).To(gbytes.Say("external service adapter exited with 41 at /thing: stdout: 'I'm stdout', stderr: 'I'm stderr'"))
		})
	})
})
