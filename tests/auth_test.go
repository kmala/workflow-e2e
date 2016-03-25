package tests

import (
	. "github.com/deis/workflow-e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Auth", func() {
	Context("when logged out", func() {
		BeforeEach(func() {
			Logout()
		})

		It("won't print the current user", func() {
			sess, err := Run("deis auth:whoami")
			Expect(err).To(BeNil())
			Eventually(sess).Should(Exit(1))
			Eventually(sess.Err).Should(Say("Not logged in"))
		})
	})

	Context("when logged in", func() {
		It("can log out", func() {
			Logout()
		})

		It("won't register twice", func() {
			cmd := "deis register %s --username=%s --password=%s --email=%s"
			out, err := Execute(cmd, Url, testUser, testPassword, testEmail)
			Expect(err).To(HaveOccurred())
			Expect(out).To(ContainSubstring("Registration failed"))
		})

		It("prints the current user", func() {
			sess, err := Run("deis auth:whoami")
			Expect(err).To(BeNil())
			Eventually(sess).Should(Exit(0))
			Eventually(sess).Should(Say("You are %s", testUser))
		})

		It("regenerates the token for the current user", func() {
			sess, err := Run("deis auth:regenerate")
			Expect(err).To(BeNil())
			Eventually(sess).Should(Exit(0))
			Eventually(sess).Should(Say("Token Regenerated"))
		})
	})

	Context("when logged in as an admin", func() {
		BeforeEach(func() {
			Login(Url, testAdminUser, testAdminPassword)
		})

		It("regenerates the token for a specified user", func() {
			output, err := Execute("deis auth:regenerate -u %s", testUser)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("Token Regenerated"))
		})

		It("regenerates the token for all users", func() {
			output, err := Execute("deis auth:regenerate --all")
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("Token Regenerated"))
		})
	})
})
