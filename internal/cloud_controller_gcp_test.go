package internal_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/franela/goblin"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/spacelift-io/awsautoscalr/internal"
	"github.com/spacelift-io/awsautoscalr/internal/ifaces"
)

func TestGCPCloudController(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("GCPCloudController", func() {
		const project = "test-project"
		const zone = "us-central1-a"
		const migName = "test-mig"

		var ctx context.Context
		var err error

		var mockInstances *ifaces.MockGCPInstances
		var mockInstanceGroupManagers *ifaces.MockGCPInstanceGroupManagers
		var mockSecretManager *ifaces.MockGCPSecretManager

		var sut *internal.GCPCloudController

		g.BeforeEach(func() {
			ctx = context.Background()
			err = nil

			mockInstances = &ifaces.MockGCPInstances{}
			mockInstanceGroupManagers = &ifaces.MockGCPInstanceGroupManagers{}
			mockSecretManager = &ifaces.MockGCPSecretManager{}

			sut = &internal.GCPCloudController{
				InstancesClient:          mockInstances,
				InstanceGroupManagers:    mockInstanceGroupManagers,
				SecretManagerClient:      mockSecretManager,
				Project:                  project,
				Zone:                     zone,
				ManagedInstanceGroupName: migName,
				Tracer:                   internal.NewNoOpTracer(),
			}
		})

		g.Describe("DescribeInstances", func() {
			instanceIDs := []string{"1234567890"}
			instanceName := "test-instance-abc"

			var instances []internal.Instance

			var listInput *computepb.ListInstancesRequest
			var listCall *mock.Call
			var getInput *computepb.GetInstanceRequest
			var getCall *mock.Call

			g.BeforeEach(func() {
				listInput = nil
				getInput = nil

				listCall = mockInstances.On(
					"ListAll",
					mock.Anything,
					mock.MatchedBy(func(in any) bool {
						listInput = in.(*computepb.ListInstancesRequest)
						return true
					}),
				)

				getCall = mockInstances.On(
					"Get",
					mock.Anything,
					mock.MatchedBy(func(in any) bool {
						getInput = in.(*computepb.GetInstanceRequest)
						return true
					}),
				)
			})

			g.JustBeforeEach(func() {
				instances, err = sut.DescribeInstances(ctx, instanceIDs)
			})

			g.Describe("when the list API call fails", func() {
				g.BeforeEach(func() { listCall.Return(nil, errors.New("bacon")) })

				g.It("sends the correct list input", func() {
					Expect(listInput).NotTo(BeNil())
					Expect(listInput.Project).To(Equal(project))
					Expect(listInput.Zone).To(Equal(zone))
				})

				g.It("should return an error", func() {
					Expect(instances).To(BeEmpty())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("could not find instance name"))
					Expect(err.Error()).To(ContainSubstring("bacon"))
				})
			})

			g.Describe("when the list API call succeeds", func() {
				var listOutput []*computepb.Instance

				g.BeforeEach(func() {
					id := uint64(1234567890)
					listOutput = []*computepb.Instance{
						{
							Id:   &id,
							Name: &instanceName,
						},
					}
					listCall.Return(listOutput, nil)
				})

				g.Describe("when the instance is not found in the list", func() {
					g.BeforeEach(func() {
						differentID := uint64(9999999999)
						listOutput[0].Id = &differentID
					})

					g.It("should return an error", func() {
						Expect(instances).To(BeEmpty())
						Expect(err).To(MatchError("could not find instance name for ID 1234567890: instance with ID 1234567890 not found"))
					})
				})

				g.Describe("when the instance has no name", func() {
					g.BeforeEach(func() {
						listOutput[0].Name = nil
					})

					g.It("should return an error", func() {
						Expect(instances).To(BeEmpty())
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("has no name"))
					})
				})

				g.Describe("when the get API call fails", func() {
					g.BeforeEach(func() { getCall.Return(nil, errors.New("ham")) })

					g.It("sends the correct get input", func() {
						Expect(getInput).NotTo(BeNil())
						Expect(getInput.Project).To(Equal(project))
						Expect(getInput.Zone).To(Equal(zone))
						Expect(getInput.Instance).To(Equal(instanceName))
					})

					g.It("should return an error", func() {
						Expect(instances).To(BeEmpty())
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("could not get instance"))
						Expect(err.Error()).To(ContainSubstring("ham"))
					})
				})

				g.Describe("when the get API call succeeds", func() {
					var getInstance *computepb.Instance

					g.BeforeEach(func() {
						id := uint64(1234567890)
						timestamp := time.Now().Format(time.RFC3339)
						getInstance = &computepb.Instance{
							Id:                &id,
							Name:              &instanceName,
							CreationTimestamp: &timestamp,
						}
						getCall.Return(getInstance, nil)
					})

					g.Describe("when the instance has no ID", func() {
						g.BeforeEach(func() { getInstance.Id = nil })

						g.It("should return an error", func() {
							Expect(instances).To(BeEmpty())
							Expect(err).To(MatchError("could not find instance ID for test-instance-abc"))
						})
					})

					g.Describe("when the instance has no creation timestamp", func() {
						g.BeforeEach(func() { getInstance.CreationTimestamp = nil })

						g.It("should return an error", func() {
							Expect(instances).To(BeEmpty())
							Expect(err).To(MatchError("could not find creation timestamp for instance 1234567890"))
						})
					})

					g.Describe("when the instance has an invalid timestamp", func() {
						g.BeforeEach(func() {
							invalidTime := "not-a-timestamp"
							getInstance.CreationTimestamp = &invalidTime
						})

						g.It("should return an error", func() {
							Expect(instances).To(BeEmpty())
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("could not parse creation timestamp"))
						})
					})

					g.Describe("when the instance has the correct ID and timestamp", func() {
						g.It("should return the instance", func() {
							Expect(err).NotTo(HaveOccurred())
							Expect(instances).To(HaveLen(1))
							Expect(instances[0].InstanceID).To(Equal("1234567890"))
							Expect(instances[0].LaunchTime).NotTo(BeZero())
						})
					})
				})
			})
		})

		g.Describe("GetAutoscalingGroup", func() {
			var group *internal.AutoScalingGroup

			var getInput *computepb.GetInstanceGroupManagerRequest
			var getCall *mock.Call
			var listInput *computepb.ListManagedInstancesInstanceGroupManagersRequest
			var listCall *mock.Call

			g.BeforeEach(func() {
				getInput = nil
				listInput = nil

				getCall = mockInstanceGroupManagers.On(
					"Get",
					mock.Anything,
					mock.MatchedBy(func(in any) bool {
						getInput = in.(*computepb.GetInstanceGroupManagerRequest)
						return true
					}),
				)

				listCall = mockInstanceGroupManagers.On(
					"ListManagedInstancesAll",
					mock.Anything,
					mock.MatchedBy(func(in any) bool {
						listInput = in.(*computepb.ListManagedInstancesInstanceGroupManagersRequest)
						return true
					}),
				)
			})

			g.JustBeforeEach(func() { group, err = sut.GetAutoscalingGroup(ctx) })

			g.Describe("when the get API call fails", func() {
				g.BeforeEach(func() { getCall.Return(nil, errors.New("bacon")) })

				g.It("sends the correct input", func() {
					Expect(getInput).NotTo(BeNil())
					Expect(getInput.Project).To(Equal(project))
					Expect(getInput.Zone).To(Equal(zone))
					Expect(getInput.InstanceGroupManager).To(Equal(migName))
				})

				g.It("should return an error", func() {
					Expect(group).To(BeNil())
					Expect(err).To(MatchError("could not get managed instance group details: bacon"))
				})
			})

			g.Describe("when the get API call succeeds", func() {
				var mig *computepb.InstanceGroupManager

				g.BeforeEach(func() {
					targetSize := int32(3)
					name := migName
					mig = &computepb.InstanceGroupManager{
						Name:       &name,
						TargetSize: &targetSize,
					}
					getCall.Return(mig, nil)
				})

				g.Describe("when the MIG has no name", func() {
					g.BeforeEach(func() { mig.Name = nil })

					g.It("should return an error", func() {
						Expect(group).To(BeNil())
						Expect(err).To(MatchError("could not find managed instance group name"))
					})
				})

				g.Describe("when listing managed instances fails", func() {
					g.BeforeEach(func() { listCall.Return(nil, errors.New("ham")) })

					g.It("sends the correct list input", func() {
						Expect(listInput).NotTo(BeNil())
						Expect(listInput.Project).To(Equal(project))
						Expect(listInput.Zone).To(Equal(zone))
						Expect(listInput.InstanceGroupManager).To(Equal(migName))
					})

					g.It("should return an error", func() {
						Expect(group).To(BeNil())
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("could not list managed instances"))
					})
				})

				g.Describe("when listing managed instances succeeds", func() {
					var managedInstances []*computepb.ManagedInstance

					g.BeforeEach(func() {
						id1 := uint64(1111111111)
						id2 := uint64(2222222222)
						instance1 := "zones/us-central1-a/instances/test-instance-1"
						instance2 := "zones/us-central1-a/instances/test-instance-2"
						status1 := "RUNNING"
						status2 := "PROVISIONING"
						action1 := "NONE"
						action2 := "CREATING"

						managedInstances = []*computepb.ManagedInstance{
							{
								Id:             &id1,
								Instance:       &instance1,
								InstanceStatus: &status1,
								CurrentAction:  &action1,
							},
							{
								Id:             &id2,
								Instance:       &instance2,
								InstanceStatus: &status2,
								CurrentAction:  &action2,
							},
						}
						listCall.Return(managedInstances, nil)
					})

					g.It("should return the group with instances", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(group).NotTo(BeNil())
						Expect(group.Name).To(Equal(migName))
						Expect(group.DesiredCapacity).To(Equal(int32(3)))
						Expect(group.Instances).To(HaveLen(2))
						Expect(group.Instances[0].InstanceID).To(Equal("1111111111"))
						Expect(group.Instances[0].LifecycleState).To(Equal("InService"))
						Expect(group.Instances[1].InstanceID).To(Equal("2222222222"))
						Expect(group.Instances[1].LifecycleState).To(Equal("Pending"))
					})

					g.Describe("when an instance has no ID", func() {
						g.BeforeEach(func() {
							managedInstances[0].Id = nil
						})

						g.It("should skip the instance without error", func() {
							Expect(err).NotTo(HaveOccurred())
							Expect(group).NotTo(BeNil())
							Expect(group.Instances).To(HaveLen(1))
							Expect(group.Instances[0].InstanceID).To(Equal("2222222222"))
						})
					})

					g.Describe("when an instance has no Instance field", func() {
						g.BeforeEach(func() {
							managedInstances[0].Instance = nil
						})

						g.It("should skip the instance without error", func() {
							Expect(err).NotTo(HaveOccurred())
							Expect(group).NotTo(BeNil())
							Expect(group.Instances).To(HaveLen(1))
							Expect(group.Instances[0].InstanceID).To(Equal("2222222222"))
						})
					})
				})
			})
		})

		g.Describe("GetSecret", func() {
			const secretName = "my-secret"
			const secretValue = "super-secret-value"

			var secret string
			var input *secretmanagerpb.AccessSecretVersionRequest
			var apiCall *mock.Call

			g.BeforeEach(func() {
				input = nil

				apiCall = mockSecretManager.On(
					"AccessSecretVersion",
					mock.Anything,
					mock.MatchedBy(func(in any) bool {
						input = in.(*secretmanagerpb.AccessSecretVersionRequest)
						return true
					}),
					mock.Anything,
				)
			})

			g.JustBeforeEach(func() { secret, err = sut.GetSecret(ctx, secretName) })

			g.Describe("when the API call fails", func() {
				g.BeforeEach(func() { apiCall.Return(nil, errors.New("bacon")) })

				g.It("sends the correct input", func() {
					Expect(input).NotTo(BeNil())
					Expect(input.Name).To(Equal("projects/test-project/secrets/my-secret/versions/latest"))
				})

				g.It("should return an error", func() {
					Expect(secret).To(BeEmpty())
					Expect(err).To(MatchError("could not access secret from Secret Manager: bacon"))
				})
			})

			g.Describe("when the API call succeeds", func() {
				var output *secretmanagerpb.AccessSecretVersionResponse

				g.BeforeEach(func() {
					output = &secretmanagerpb.AccessSecretVersionResponse{
						Payload: &secretmanagerpb.SecretPayload{
							Data: []byte(secretValue),
						},
					}
					apiCall.Return(output, nil)
				})

				g.Describe("when the payload is nil", func() {
					g.BeforeEach(func() { output.Payload = nil })

					g.It("should return an error", func() {
						Expect(secret).To(BeEmpty())
						Expect(err).To(MatchError("secret payload is empty"))
					})
				})

				g.Describe("when the payload data is nil", func() {
					g.BeforeEach(func() { output.Payload.Data = nil })

					g.It("should return an error", func() {
						Expect(secret).To(BeEmpty())
						Expect(err).To(MatchError("secret payload is empty"))
					})
				})

				g.Describe("when the payload is valid", func() {
					g.It("should return the secret value", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(secret).To(Equal(secretValue))
					})
				})
			})
		})

		g.Describe("KillInstance", func() {
			const instanceID = "1234567890"
			const instanceName = "test-instance"

			var listCall *mock.Call
			var deleteInput *computepb.DeleteInstancesInstanceGroupManagerRequest
			var deleteCall *mock.Call

			g.BeforeEach(func() {
				deleteInput = nil

				listCall = mockInstances.On(
					"ListAll",
					mock.Anything,
					mock.Anything,
				)

				deleteCall = mockInstanceGroupManagers.On(
					"DeleteInstances",
					mock.Anything,
					mock.MatchedBy(func(in any) bool {
						deleteInput = in.(*computepb.DeleteInstancesInstanceGroupManagerRequest)
						return true
					}),
				)
			})

			g.JustBeforeEach(func() { err = sut.KillInstance(ctx, instanceID) })

			g.Describe("when finding the instance name fails", func() {
				g.BeforeEach(func() { listCall.Return(nil, errors.New("bacon")) })

				g.It("should return an error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("could not find instance name"))
					Expect(err.Error()).To(ContainSubstring("bacon"))
				})
			})

			g.Describe("when the instance is not found", func() {
				g.BeforeEach(func() {
					listCall.Return([]*computepb.Instance{}, nil)
				})

				g.It("should succeed (instance already deleted)", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			g.Describe("when finding the instance succeeds", func() {
				g.BeforeEach(func() {
					id := uint64(1234567890)
					name := instanceName
					listCall.Return([]*computepb.Instance{
						{Id: &id, Name: &name},
					}, nil)
				})

				g.Describe("when the delete from MIG call fails", func() {
					g.BeforeEach(func() { deleteCall.Return(nil, errors.New("bacon")) })

					g.It("sends the correct delete input", func() {
						Expect(deleteInput).NotTo(BeNil())
						Expect(deleteInput.Project).To(Equal(project))
						Expect(deleteInput.Zone).To(Equal(zone))
						Expect(deleteInput.InstanceGroupManager).To(Equal(migName))
						Expect(deleteInput.InstanceGroupManagersDeleteInstancesRequestResource).NotTo(BeNil())
						Expect(deleteInput.InstanceGroupManagersDeleteInstancesRequestResource.Instances).To(
							ConsistOf("zones/us-central1-a/instances/test-instance"),
						)
					})

					g.It("should return an error", func() {
						Expect(err).To(MatchError("could not delete instance from managed instance group: bacon"))
					})
				})

				g.Describe("when the instance is not part of the MIG", func() {
					var directDeleteInput *computepb.DeleteInstanceRequest
					var directDeleteCall *mock.Call

					g.BeforeEach(func() {
						directDeleteInput = nil

						deleteCall.Return(nil, errors.New("is not a member"))

						directDeleteCall = mockInstances.On(
							"Delete",
							mock.Anything,
							mock.MatchedBy(func(in any) bool {
								directDeleteInput = in.(*computepb.DeleteInstanceRequest)
								return true
							}),
						)
					})

					g.Describe("when the direct delete fails", func() {
						g.BeforeEach(func() { directDeleteCall.Return(nil, errors.New("ham")) })

						g.It("sends the correct direct delete input", func() {
							Expect(directDeleteInput).NotTo(BeNil())
							Expect(directDeleteInput.Project).To(Equal(project))
							Expect(directDeleteInput.Zone).To(Equal(zone))
							Expect(directDeleteInput.Instance).To(Equal(instanceName))
						})

						g.It("should return an error", func() {
							Expect(err).To(MatchError("could not delete instance directly: ham"))
						})
					})

					g.Describe("when the direct delete succeeds", func() {
						g.BeforeEach(func() {
							mockOp := &ifaces.MockGCPOperation{}
							mockOp.On("Wait", mock.Anything).Return(nil)
							directDeleteCall.Return(mockOp, nil)
						})

						g.It("succeeds", func() {
							Expect(err).NotTo(HaveOccurred())
						})
					})

					g.Describe("when waiting for direct delete operation fails", func() {
						g.BeforeEach(func() {
							mockOp := &ifaces.MockGCPOperation{}
							mockOp.On("Wait", mock.Anything).Return(errors.New("wait-error"))
							directDeleteCall.Return(mockOp, nil)
						})

						g.It("should return an error", func() {
							Expect(err).To(MatchError("error waiting for direct delete operation: wait-error"))
						})
					})
				})

				g.Describe("when the delete from MIG succeeds", func() {
					g.BeforeEach(func() {
						mockOp := &ifaces.MockGCPOperation{}
						mockOp.On("Wait", mock.Anything).Return(nil)
						deleteCall.Return(mockOp, nil)
					})

					g.It("succeeds", func() {
						Expect(err).NotTo(HaveOccurred())
					})
				})

				g.Describe("when waiting for delete operation fails", func() {
					g.BeforeEach(func() {
						mockOp := &ifaces.MockGCPOperation{}
						mockOp.On("Wait", mock.Anything).Return(errors.New("wait-error"))
						deleteCall.Return(mockOp, nil)
					})

					g.It("should return an error", func() {
						Expect(err).To(MatchError("error waiting for delete operation: wait-error"))
					})
				})
			})
		})

		g.Describe("ScaleUpASG", func() {
			const desiredCapacity = 42

			var resizeInput *computepb.ResizeInstanceGroupManagerRequest
			var resizeCall *mock.Call

			g.BeforeEach(func() {
				resizeInput = nil

				resizeCall = mockInstanceGroupManagers.On(
					"Resize",
					mock.Anything,
					mock.MatchedBy(func(in any) bool {
						resizeInput = in.(*computepb.ResizeInstanceGroupManagerRequest)
						return true
					}),
				)
			})

			g.JustBeforeEach(func() { err = sut.ScaleUpASG(ctx, desiredCapacity) })

			g.Describe("when the resize call fails", func() {
				g.BeforeEach(func() { resizeCall.Return(nil, errors.New("bacon")) })

				g.It("sends the correct input", func() {
					Expect(resizeInput).NotTo(BeNil())
					Expect(resizeInput.Project).To(Equal(project))
					Expect(resizeInput.Zone).To(Equal(zone))
					Expect(resizeInput.InstanceGroupManager).To(Equal(migName))
					Expect(resizeInput.Size).To(BeEquivalentTo(desiredCapacity))
				})

				g.It("should return an error", func() {
					Expect(err).To(MatchError("could not resize managed instance group: bacon"))
				})
			})

			g.Describe("when the resize call succeeds", func() {
				g.BeforeEach(func() {
					mockOp := &ifaces.MockGCPOperation{}
					mockOp.On("Wait", mock.Anything).Return(nil)
					resizeCall.Return(mockOp, nil)
				})

				g.It("succeeds", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})

			g.Describe("when waiting for resize operation fails", func() {
				g.BeforeEach(func() {
					mockOp := &ifaces.MockGCPOperation{}
					mockOp.On("Wait", mock.Anything).Return(errors.New("wait-error"))
					resizeCall.Return(mockOp, nil)
				})

				g.It("should return an error", func() {
					Expect(err).To(MatchError("error waiting for resize operation: wait-error"))
				})
			})
		})

		g.Describe("MapGCPStatusToAWSLifecycleState", func() {
			g.It("maps CREATING action to Pending", func() {
				status := "RUNNING"
				action := "CREATING"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, &action)).To(Equal("Pending"))
			})

			g.It("maps RECREATING action to Pending", func() {
				status := "RUNNING"
				action := "RECREATING"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, &action)).To(Equal("Pending"))
			})

			g.It("maps DELETING action to Terminating", func() {
				status := "RUNNING"
				action := "DELETING"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, &action)).To(Equal("Terminating"))
			})

			g.It("maps ABANDONING action to Terminating", func() {
				status := "RUNNING"
				action := "ABANDONING"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, &action)).To(Equal("Terminating"))
			})

			g.It("maps RESTARTING action to Pending", func() {
				status := "RUNNING"
				action := "RESTARTING"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, &action)).To(Equal("Pending"))
			})

			g.It("maps RUNNING status to InService when action is NONE", func() {
				status := "RUNNING"
				action := "NONE"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, &action)).To(Equal("InService"))
			})

			g.It("maps PROVISIONING status to Pending", func() {
				status := "PROVISIONING"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, nil)).To(Equal("Pending"))
			})

			g.It("maps STAGING status to Pending", func() {
				status := "STAGING"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, nil)).To(Equal("Pending"))
			})

			g.It("maps REPAIRING status to Pending", func() {
				status := "REPAIRING"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, nil)).To(Equal("Pending"))
			})

			g.It("maps STOPPING status to Terminating", func() {
				status := "STOPPING"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, nil)).To(Equal("Terminating"))
			})

			g.It("maps STOPPED status to Terminating", func() {
				status := "STOPPED"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, nil)).To(Equal("Terminating"))
			})

			g.It("maps TERMINATED status to Terminating", func() {
				status := "TERMINATED"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, nil)).To(Equal("Terminating"))
			})

			g.It("maps SUSPENDING status to Terminating", func() {
				status := "SUSPENDING"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, nil)).To(Equal("Terminating"))
			})

			g.It("maps SUSPENDED status to Terminating", func() {
				status := "SUSPENDED"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, nil)).To(Equal("Terminating"))
			})

			g.It("returns empty string for unknown status", func() {
				status := "UNKNOWN_STATUS"
				Expect(internal.MapGCPStatusToAWSLifecycleState(&status, nil)).To(Equal(""))
			})

			g.It("returns empty string when both status and action are nil", func() {
				Expect(internal.MapGCPStatusToAWSLifecycleState(nil, nil)).To(Equal(""))
			})
		})
	})
}
