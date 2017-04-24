// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package broker_test

import (
	"errors"
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/on-demand-service-broker/boshclient"
	"github.com/pivotal-cf/on-demand-service-broker/broker"
	"github.com/pivotal-cf/on-demand-service-broker/config"
)

var _ = Describe("Lifecycle runner", func() {
	const (
		deploymentName       = "some-deployment"
		contextID            = "some-uuid"
		planID               = "some-plan-id"
		errand               = "some-errand"
		anotherErrand        = "another-errand"
		anotherPlanID        = "another-plan-id"
		planIDWithoutErrands = "without-errands-plan-id"
	)

	plans := config.Plans{
		config.Plan{
			ID: planID,
			LifecycleErrands: &config.LifecycleErrands{
				PostDeploy: errand,
			},
		},
		config.Plan{
			ID: anotherPlanID,
			LifecycleErrands: &config.LifecycleErrands{
				PostDeploy: anotherErrand,
			},
		},
		config.Plan{
			ID: planIDWithoutErrands,
		},
	}

	taskProcessing := boshclient.BoshTask{ID: 1, State: boshclient.BoshTaskProcessing, Description: "snapshot deployment", Result: "result-1", ContextID: contextID}
	taskErrored := boshclient.BoshTask{ID: 2, State: boshclient.BoshTaskError, Description: "snapshot deployment", Result: "result-1", ContextID: contextID}
	taskComplete := boshclient.BoshTask{ID: 3, State: boshclient.BoshTaskDone, Description: "snapshot deployment", Result: "result-1", ContextID: contextID}

	var deployRunner broker.LifeCycleRunner
	var logger *log.Logger
	var operationData broker.OperationData

	BeforeEach(func() {
		deployRunner = broker.NewLifeCycleRunner(
			boshClient,
			plans,
		)

		logger = loggerFactory.NewWithRequestID()
	})

	Describe("post-deploy errand", func() {
		Context("when operation data has a context id", func() {
			BeforeEach(func() {
				operationData = broker.OperationData{
					BoshContextID: contextID,
					OperationType: broker.OperationTypeCreate,
					PlanID:        planID,
				}
			})

			Context("when a first task is incomplete", func() {
				BeforeEach(func() {
					boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{taskProcessing}, nil)
				})

				It("returns the processing task", func() {
					task, _ := deployRunner.GetTask(deploymentName, operationData, logger)
					Expect(task.State).To(Equal(boshclient.BoshTaskProcessing))
				})
			})

			Context("when a first task has errored", func() {
				BeforeEach(func() {
					boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{taskErrored}, nil)
				})

				It("returns the errored task", func() {
					task, _ := deployRunner.GetTask(deploymentName, operationData, logger)
					Expect(task.State).To(Equal(boshclient.BoshTaskError))
				})
			})

			Context("when a first task cannot be found", func() {
				BeforeEach(func() {
					boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{}, nil)
				})

				It("returns an error", func() {
					_, err := deployRunner.GetTask(deploymentName, operationData, logger)
					Expect(err).To(MatchError("no tasks found for context id: " + contextID))
				})
			})

			Context("when a first task is complete", func() {
				var task boshclient.BoshTask
				BeforeEach(func() {
					boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{taskComplete}, nil)
				})

				Context("and a post deploy is not configured", func() {
					BeforeEach(func() {
						deployRunner = broker.NewLifeCycleRunner(boshClient, plans)

						opData := operationData
						opData.PlanID = planIDWithoutErrands

						task, _ = deployRunner.GetTask(deploymentName, opData, logger)
					})

					It("returns the completed task", func() {
						Expect(task.State).To(Equal(boshclient.BoshTaskDone))
					})

					It("does not run a post deploy errand", func() {
						Expect(boshClient.RunErrandCallCount()).To(Equal(0))
					})
				})

				Context("and a post deploy is configured", func() {
					BeforeEach(func() {
						boshClient.RunErrandReturns(taskProcessing.ID, nil)
						boshClient.GetTaskReturns(taskProcessing, nil)
						task, _ = deployRunner.GetTask(deploymentName, operationData, logger)
					})

					It("runs the post deploy errand", func() {
						Expect(boshClient.RunErrandCallCount()).To(Equal(1))
					})

					It("runs the correct errand", func() {
						_, errandName, _, _ := boshClient.RunErrandArgsForCall(0)
						Expect(errandName).To(Equal(errand))
					})

					It("runs the errand with the correct contextID", func() {
						_, _, ctxID, _ := boshClient.RunErrandArgsForCall(0)
						Expect(ctxID).To(Equal(contextID))
					})

					It("returns the post deploy errand processing task", func() {
						Expect(task.ID).To(Equal(taskProcessing.ID))
						Expect(task.State).To(Equal(boshclient.BoshTaskProcessing))
					})

					Context("and a post deploy errand is incomplete", func() {
						BeforeEach(func() {
							boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{taskProcessing, taskComplete}, nil)
							task, _ = deployRunner.GetTask(deploymentName, operationData, logger)
						})

						It("returns the processing task", func() {
							Expect(task.State).To(Equal(boshclient.BoshTaskProcessing))
						})

						It("does not run a post deploy errand again", func() {
							Expect(boshClient.RunErrandCallCount()).To(Equal(1))
						})
					})

					Context("and a post deploy errand is complete", func() {
						BeforeEach(func() {
							boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{taskComplete, taskComplete}, nil)
							task, _ = deployRunner.GetTask(deploymentName, operationData, logger)
						})

						It("returns the complete task", func() {
							Expect(task.State).To(Equal(boshclient.BoshTaskDone))
						})

						It("does not run a post deploy errand again", func() {
							Expect(boshClient.RunErrandCallCount()).To(Equal(1))
						})
					})

					Context("and the post deploy errand fails", func() {
						BeforeEach(func() {
							boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{taskErrored, taskComplete}, nil)
							task, _ = deployRunner.GetTask(deploymentName, operationData, logger)
						})

						It("returns the failed task", func() {
							Expect(task.State).To(Equal(boshclient.BoshTaskError))
						})

						It("does not run a post deploy errand again", func() {
							Expect(boshClient.RunErrandCallCount()).To(Equal(1))
						})
					})

					Context("and when running the errand errors", func() {
						BeforeEach(func() {
							boshClient.RunErrandReturns(0, errors.New("some errand err"))
						})

						It("returns an error", func() {
							_, err := deployRunner.GetTask(deploymentName, operationData, logger)
							Expect(err).To(MatchError("some errand err"))
						})
					})

					Context("and the errand task cannot be found", func() {
						BeforeEach(func() {
							boshClient.GetTaskReturns(boshclient.BoshTask{}, errors.New("some err"))
						})

						It("returns an error", func() {
							_, err := deployRunner.GetTask(deploymentName, operationData, logger)
							Expect(err).To(MatchError("some err"))
						})
					})
				})

				Context("and the plan cannot be found", func() {
					BeforeEach(func() {
						opData := operationData
						opData.PlanID = "non-existent-plan"
						task, _ = deployRunner.GetTask(deploymentName, opData, logger)
					})

					It("does not run a post deploy errand", func() {
						Expect(boshClient.RunErrandCallCount()).To(Equal(0))
					})

					It("returns the completed task", func() {
						Expect(task).To(Equal(taskComplete))
					})
				})
			})

			Context("when getting tasks errors", func() {
				BeforeEach(func() {
					boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{}, errors.New("some err"))
				})

				It("returns an error", func() {
					_, err := deployRunner.GetTask(deploymentName, operationData, logger)
					Expect(err).To(MatchError("some err"))
				})
			})
		})

		Context("when operation data has no context id", func() {
			operationData := broker.OperationData{BoshTaskID: taskProcessing.ID, OperationType: broker.OperationTypeCreate}

			BeforeEach(func() {
				boshClient.GetTaskReturns(taskProcessing, nil)
			})

			It("calls get tasks with the correct id", func() {
				deployRunner.GetTask(deploymentName, operationData, logger)

				Expect(boshClient.GetTaskCallCount()).To(Equal(1))
				actualTaskID, _ := boshClient.GetTaskArgsForCall(0)
				Expect(actualTaskID).To(Equal(taskProcessing.ID))
			})

			It("returns the processing task", func() {
				task, _ := deployRunner.GetTask(deploymentName, operationData, logger)

				Expect(task).To(Equal(taskProcessing))
			})

			It("does not error", func() {
				_, err := deployRunner.GetTask(deploymentName, operationData, logger)

				Expect(err).ToNot(HaveOccurred())
			})

			Context("and bosh client returns an error", func() {
				BeforeEach(func() {
					boshClient.GetTaskReturns(boshclient.BoshTask{}, errors.New("error getting tasks"))
				})

				It("returns the error", func() {
					_, err := deployRunner.GetTask(deploymentName, operationData, logger)

					Expect(err).To(MatchError("error getting tasks"))
				})
			})
		})

		DescribeTable("for different operation types",
			func(operationType broker.OperationType, errandRuns bool) {
				operationData := broker.OperationData{OperationType: operationType, BoshContextID: contextID, PlanID: planID}
				boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{taskComplete}, nil)
				deployRunner.GetTask(deploymentName, operationData, logger)

				if errandRuns {
					Expect(boshClient.RunErrandCallCount()).To(Equal(1))
				} else {
					Expect(boshClient.RunErrandCallCount()).To(Equal(0))
				}
			},
			Entry("create runs errand", broker.OperationTypeCreate, true),
			Entry("update runs errand", broker.OperationTypeUpdate, true),
			Entry("upgrade runs errand", broker.OperationTypeUpgrade, true),
			Entry("delete does not run errand", broker.OperationTypeDelete, false),
		)
	})

	Describe("pre-delete errand", func() {
		BeforeEach(func() {
			operationData = broker.OperationData{
				BoshContextID: contextID,
				OperationType: broker.OperationTypeDelete,
				PlanID:        planID,
			}
		})

		Context("when a first task is incomplete", func() {
			BeforeEach(func() {
				boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{taskProcessing}, nil)
			})

			It("returns the processing task", func() {
				task, _ := deployRunner.GetTask(deploymentName, operationData, logger)
				Expect(task.State).To(Equal(boshclient.BoshTaskProcessing))
			})
		})

		Context("when the first task has errored", func() {
			BeforeEach(func() {
				boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{taskErrored}, nil)
			})

			It("returns the errored task", func() {
				task, _ := deployRunner.GetTask(deploymentName, operationData, logger)
				Expect(task.State).To(Equal(boshclient.BoshTaskError))
			})
		})

		Context("when a first task cannot be found", func() {
			BeforeEach(func() {
				boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{}, nil)
			})

			It("returns an error", func() {
				_, err := deployRunner.GetTask(deploymentName, operationData, logger)
				Expect(err).To(MatchError("no tasks found for context id: " + contextID))
			})
		})

		Context("when a first task is complete", func() {
			var task boshclient.BoshTask

			BeforeEach(func() {
				boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{taskComplete}, nil)
				boshClient.DeleteDeploymentReturns(taskProcessing.ID, nil)
				boshClient.GetTaskReturns(taskProcessing, nil)
				task, _ = deployRunner.GetTask(deploymentName, operationData, logger)
			})

			It("runs bosh delete deployment ", func() {
				Expect(boshClient.DeleteDeploymentCallCount()).To(Equal(1))
			})

			It("deletes the correct deployment", func() {
				deletedDeploymentName, _, _ := boshClient.DeleteDeploymentArgsForCall(0)
				Expect(deletedDeploymentName).To(Equal(deploymentName))
			})

			It("runs the delete deployment with the correct contextID", func() {
				_, ctxID, _ := boshClient.DeleteDeploymentArgsForCall(0)
				Expect(ctxID).To(Equal(contextID))
			})

			It("returns the post deploy errand processing task", func() {
				Expect(task.ID).To(Equal(taskProcessing.ID))
				Expect(task.State).To(Equal(boshclient.BoshTaskProcessing))
			})

			Context("and running bosh delete deployment fails", func() {
				BeforeEach(func() {
					boshClient.DeleteDeploymentReturns(0, errors.New("some err"))
				})

				It("returns an error", func() {
					_, err := deployRunner.GetTask(deploymentName, operationData, logger)
					Expect(err).To(MatchError("some err"))
				})
			})
		})

		Context("when there are two tasks for the context id", func() {
			BeforeEach(func() {
				boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{taskProcessing, taskComplete}, nil)
			})

			It("returns the latest task", func() {
				task, _ := deployRunner.GetTask(deploymentName, operationData, logger)
				Expect(task).To(Equal(taskProcessing))
			})
		})

		Context("when there are more than two tasks for the context id", func() {
			BeforeEach(func() {
				boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{
					taskProcessing,
					taskComplete,
					taskComplete,
				}, nil)
			})

			It("returns an error", func() {
				_, err := deployRunner.GetTask(deploymentName, operationData, logger)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when getting tasks errors", func() {
			BeforeEach(func() {
				boshClient.GetNormalisedTasksByContextReturns(boshclient.BoshTasks{}, errors.New("some err"))
			})

			It("returns an error", func() {
				_, err := deployRunner.GetTask(deploymentName, operationData, logger)
				Expect(err).To(MatchError("some err"))
			})
		})
	})
})
