// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package integration_new_tests

import (
	"fmt"
	"math/rand"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/pivotal-cf/on-demand-service-broker/broker"
)

var _ = Describe("updating a service instance", func() {
	It("returns tracking data for an update operation", func() {
		updateTaskID := rand.Int()
		boshDeploysUpdatedManifest := func(env *BrokerEnvironment) {
			deploymentName := env.DeploymentName()

			env.Bosh.HasNoTasksFor(deploymentName)
			env.Bosh.HasManifestFor(deploymentName)
			env.Bosh.DeploysWithoutContextId(deploymentName, updateTaskID)
		}

		When(updatingServiceInstance).
			With(NoCredhub, serviceAdapterGeneratesManifest, boshDeploysUpdatedManifest).
			theBroker(
				RespondsWith(http.StatusAccepted, OperationData(withoutErrand(broker.OperationTypeUpdate, updateTaskID))),
				LogsWithServiceId("updating instance %s"),
				LogsWithDeploymentName(fmt.Sprintf("Bosh task ID for update deployment %%s is %d", updateTaskID)),
			)
	})

	XIt("runs the post-deployment errand if new plan has one", func() {
		const postDeployErrandName = "post-deploy-errand-name"
		updateTaskID := rand.Int()
		boshDeploysUpdatedManifest := func(env *BrokerEnvironment) {
			deploymentName := env.DeploymentName()

			env.Bosh.HasNoTasksFor(deploymentName)
			env.Bosh.HasManifestFor(deploymentName)
			env.Bosh.DeploysWithoutContextId(deploymentName, updateTaskID)
		}

		When(updatingServiceInstance).
			With(NoCredhub, serviceAdapterGeneratesManifest, boshDeploysUpdatedManifest).
			theBroker(
				RespondsWith(http.StatusAccepted, OperationData(withErrand(broker.OperationTypeUpdate, updateTaskID, postDeployErrandName))),
				LogsWithServiceId("updating instance %s"),
				LogsWithDeploymentName(fmt.Sprintf("Bosh task ID for update deployment %%s is %d", updateTaskID)),
			)
	})

})

// TODO Should we verify the parameters to GenerateManifest?

var updatingServiceInstance = func(env *BrokerEnvironment) *http.Request {
	return env.Broker.UpdateServiceInstanceRequest(env.serviceInstanceID)
}
var serviceAdapterGeneratesManifest = func(sa *ServiceAdapter, id ServiceInstanceID) {
	sa.adapter.GenerateManifest().ToReturnManifest(rawManifestWithDeploymentName(id))
}

func rawManifestWithDeploymentName(id ServiceInstanceID) string {
	return "name: " + broker.DeploymentNameFrom(string(id))
}

func withoutErrand(opType broker.OperationType, taskId int) types.GomegaMatcher {
	return MatchAllFields(
		Fields{
			"OperationType":        Equal(opType),
			"BoshTaskID":           Equal(taskId),
			"PlanID":               BeEmpty(),
			"PostDeployErrandName": BeEmpty(),
			"BoshContextID":        BeEmpty(),
		})
}

func withErrand(opType broker.OperationType, taskId int, errandName string) types.GomegaMatcher {
	return MatchAllFields(
		Fields{
			"OperationType":        Equal(opType),
			"BoshTaskID":           Equal(taskId),
			"PlanID":               BeEmpty(),
			"PostDeployErrandName": Equal(errandName),
			"BoshContextID":        Not(BeEmpty()),
		})
}