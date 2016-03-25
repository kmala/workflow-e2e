package tests

import (
	. "github.com/deis/workflow-e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Users", func() {
	Context("when logged in as an admin user", func() {
		BeforeEach(func() {
			Login(Url, testAdminUser, testAdminPassword)
		})

		It("can list all users", func() {
			output, err := Execute("deis users:list")
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(SatisfyAll(
				HavePrefix("=== Users"),
				ContainSubstring(testUser),
				ContainSubstring(testAdminUser)))
		})
	})

	Context("when logged in as a normal user", func() {
		BeforeEach(func() {
			Login(Url, testUser, testPassword)
		})

		It("can't list all users", func() {
			output, err := Execute("deis users:list")
			Expect(err).To(HaveOccurred())
			Expect(output).To(ContainSubstring("403 Forbidden"))
		})
	})
})
