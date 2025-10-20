package internal_test

import (
	"testing"

	"github.com/franela/goblin"
	. "github.com/onsi/gomega"

	"github.com/spacelift-io/awsautoscalr/internal"
)

func TestWorker(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("Worker", func() {
		var sut *internal.Worker

		g.BeforeEach(func() { sut = &internal.Worker{} })

		g.Describe("InstanceIdentity", func() {
			var groupID internal.GroupID
			var instanceID internal.InstanceID
			var err error

			g.JustBeforeEach(func() { groupID, instanceID, err = sut.InstanceIdentity() })

			g.Describe("with no metadata", func() {
				g.BeforeEach(func() { sut.Metadata = "{}" })

				g.It("should return an error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("metadata asg_id not present"))
					Expect(err.Error()).To(ContainSubstring("metadata instance_id not present"))
				})
			})

			g.Describe("with valid AWS metadata", func() {
				g.BeforeEach(func() {
					sut.Metadata = `{"asg_id": "group", "instance_id": "instance"}`
				})

				g.It("should return the group and instance IDs", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(groupID).To(Equal(internal.GroupID("group")))
					Expect(instanceID).To(Equal(internal.InstanceID("instance")))
				})
			})

			g.Describe("with valid GCP metadata", func() {
				g.BeforeEach(func() {
					sut.Metadata = `{"cloud_provider": "gcp", "gcp_mig_name": "gcp-group", "gcp_instance_id": "gcp-instance"}`
				})

				g.It("should return the GCP group and instance IDs", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(groupID).To(Equal(internal.GroupID("gcp-group")))
					Expect(instanceID).To(Equal(internal.InstanceID("gcp-instance")))
				})
			})

			g.Describe("with GCP cloud provider but missing GCP metadata", func() {
				g.BeforeEach(func() {
					sut.Metadata = `{"cloud_provider": "gcp"}`
				})

				g.It("should return an error for missing GCP fields", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("metadata gcp_mig_name not present"))
					Expect(err.Error()).To(ContainSubstring("metadata gcp_instance_id not present"))
				})
			})
		})
	})
}
