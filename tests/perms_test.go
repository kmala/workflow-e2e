package tests

import (
	. "github.com/deis/workflow-e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Perms", func() {
	var testApp App

	BeforeEach(func() {
		testApp.Name = GetRandAppName()
		GitInit()
		CreateApp(testApp.Name)
	})

	AfterEach(func() {
		GitClean()
	})

	Context("when logged in as an admin user", func() {
		BeforeEach(func() {
			Login(Url, testAdminUser, testAdminPassword)
		})

		It("can create, list, and delete admin permissions", func() {
			output, err := Execute("deis perms:create %s --admin", testUser)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(
				ContainSubstring("Adding %s to system administrators... done\n", testUser))
			output, err = Execute("deis perms:list --admin")
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(SatisfyAll(
				HavePrefix("=== Administrators"),
				ContainSubstring(testUser),
				ContainSubstring(testAdminUser)))
			output, err = Execute("deis perms:delete %s --admin", testUser)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(
				ContainSubstring("Removing %s from system administrators... done", testUser))
			output, err = Execute("deis perms:list --admin")
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(SatisfyAll(
				HavePrefix("=== Administrators"),
				ContainSubstring(testAdminUser)))
			Expect(output).NotTo(ContainSubstring(testUser))
		})

		It("can create, list, and delete app permissions", func() {
			sess, err := Run("deis perms:create %s --app=%s", testUser, testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("Adding %s to %s collaborators... done\n", testUser, testApp.Name))

			sess, err = Run("deis perms:list --app=%s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("=== %s's Users", testApp.Name))
			Eventually(sess).Should(Say("%s", testUser))

			sess, err = Run("deis perms:delete %s --app=%s", testUser, testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("Removing %s from %s collaborators... done", testUser, testApp.Name))

			sess, err = Run("deis perms:list --app=%s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("=== %s's Users", testApp.Name))
			Eventually(sess).ShouldNot(Say("%s", testUser))

			Eventually(sess).Should(Exit(0))
		})
	})

	Context("when logged in as a normal user", func() {
		It("can't create, list, or delete admin permissions", func() {
			output, err := Execute("deis perms:create %s --admin", testAdminUser)
			Expect(err).To(HaveOccurred())
			Expect(output).To(ContainSubstring("403 Forbidden"))
			output, err = Execute("deis perms:list --admin")
			Expect(err).To(HaveOccurred())
			Expect(output).To(ContainSubstring("403 Forbidden"))
			output, err = Execute("deis perms:delete %s --admin", testAdminUser)
			Expect(err).To(HaveOccurred())
			Expect(output).To(ContainSubstring("403 Forbidden"))
			output, err = Execute("deis perms:list --admin")
			Expect(err).To(HaveOccurred())
			Expect(output).To(ContainSubstring("403 Forbidden"))
		})

		It("can create, list, and delete app permissions", func() {
			sess, err := Run("deis perms:create %s --app=%s", testUser, testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("Adding %s to %s collaborators... done\n", testUser, testApp.Name))

			sess, err = Run("deis perms:list --app=%s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("=== %s's Users", testApp.Name))
			Eventually(sess).Should(Say("%s", testUser))

			sess, err = Run("deis perms:delete %s --app=%s", testUser, testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("Removing %s from %s collaborators... done", testUser, testApp.Name))

			sess, err = Run("deis perms:list --app=%s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("=== %s's Users", testApp.Name))
			Eventually(sess).ShouldNot(Say("%s", testUser))

			Eventually(sess).Should(Exit(0))
		})
	})
})
