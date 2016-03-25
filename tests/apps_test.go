package tests

import (
	"os"
	"time"

	. "github.com/deis/workflow-e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Apps", func() {
	var testApp App

	BeforeEach(func() {
		testApp.Name = GetRandAppName()
	})

	Context("with no app", func() {

		It("can't get app info", func() {
			sess, _ := Run("deis info -a %s", testApp.Name)
			Eventually(sess).Should(Exit(1))
			Eventually(sess.Err).Should(Say("Not found."))
		})

		It("can't get app logs", func() {
			sess, err := Run("deis logs -a %s", testApp.Name)
			Expect(err).To(BeNil())
			Eventually(sess).Should(Exit(1))
			Eventually(sess.Err).Should(Say(`Error: There are currently no log messages. Please check the following things:`))
		})

		It("can't run a command in the app environment", func() {
			sess, err := Run("deis apps:run echo Hello, 世界")
			Expect(err).To(BeNil())
			Eventually(sess).Should(Say("Running 'echo Hello, 世界'..."))
			Eventually(sess.Err).Should(Say("Not found."))
			Eventually(sess).ShouldNot(Exit(0))
		})

		It("can't open a bogus app URL", func() {
			sess, err := Run("deis open -a %s", GetRandAppName())
			Expect(err).To(BeNil())
			Eventually(sess).Should(Exit(1))
			Eventually(sess.Err).Should(Say("404 Not Found"))
		})

	})

	Context("when creating an app", func() {
		var cleanup bool

		BeforeEach(func() {
			cleanup = true
			testApp.Name = GetRandAppName()
			GitInit()
		})

		AfterEach(func() {
			if cleanup {
				DestroyApp(testApp)
				GitClean()
			}
		})

		It("creates an app with a git remote", func() {
			cmd, err := Run("deis apps:create %s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(cmd).Should(Say("created %s", testApp.Name))
			Eventually(cmd).Should(Say(`Git remote deis added`))
			Eventually(cmd).Should(Say(`remote available at `))
		})

		It("creates an app with no git remote", func() {
			cmd, err := Run("deis apps:create %s --no-remote", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(cmd).Should(SatisfyAll(
				Say("created %s", testApp.Name),
				Say("remote available at ")))
			Eventually(cmd).ShouldNot(Say("Git remote deis added"))

			cleanup = false
			cmd = DestroyApp(testApp)
			Eventually(cmd).ShouldNot(Say("Git remote deis removed"))
		})

		It("creates an app with a custom buildpack", func() {
			sess, err := Run("deis apps:create %s --buildpack https://example.com", testApp.Name)
			Expect(err).To(BeNil())
			Eventually(sess).Should(Exit(0))
			Eventually(sess).Should(Say("created %s", testApp.Name))
			Eventually(sess).Should(Say("Git remote deis added"))
			Eventually(sess).Should(Say("remote available at "))

			sess, err = Run("deis config:list -a %s", testApp.Name)
			Expect(err).To(BeNil())
			Eventually(sess).Should(Exit(0))
			Eventually(sess).Should(Say("BUILDPACK_URL"))
		})
	})

	Context("with a deployed app", func() {
		var cleanup bool
		var testApp App

		BeforeEach(func() {
			cleanup = true
			os.Chdir("example-go")
			appName := GetRandAppName()
			CreateApp(appName)
			testApp = DeployApp(appName, gitSSH)
		})

		AfterEach(func() {
			defer os.Chdir("..")
			if cleanup {
				DestroyApp(testApp)
			}
		})

		It("can't create an existing app", func() {
			sess, err := Run("deis apps:create %s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("App with this id already exists."))
			Eventually(sess).ShouldNot(Exit(0))
		})

		It("can get app info", func() {
			VerifyAppInfo(testApp, testUser)
		})

		// V broken
		XIt("can get app logs", func() {
			cmd, err := Run("deis logs")
			Expect(err).NotTo(HaveOccurred())
			Eventually(cmd).Should(SatisfyAll(
				Say("%s\\[deis-controller\\]\\: %s created initial release", testApp.Name, testUser),
				Say("%s\\[deis-controller\\]\\: %s deployed", testApp.Name, testUser),
				Say("%s\\[deis-controller\\]\\: %s scaled containers", testApp.Name, testUser)))
		})

		It("can open the app's URL", func() {
			VerifyAppOpen(testApp)
		})

		It("can run a command in the app environment", func() {
			sess, err := Run("deis apps:run echo Hello, 世界")
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess, (1 * time.Minute)).Should(Say("Hello, 世界"))
		})

		It("can transfer the app to another owner", func() {
			_, err := Run("deis apps:transfer " + testAdminUser)
			Expect(err).NotTo(HaveOccurred())
			sess, _ := Run("deis info -a %s", testApp.Name)
			Eventually(sess).Should(Exit(1))
			Eventually(sess.Err).Should(Say("You do not have permission to perform this action."))
			// destroy it ourselves because the spec teardown cannot destroy as regular user
			cleanup = false
			Login(Url, testAdminUser, testAdminPassword)
			DestroyApp(testApp)
			// log back in and continue with the show
			Login(Url, testUser, testPassword)
		})
	})

	Context("with a custom buildpack deployed app", func() {
		var cleanup bool
		var testApp App

		BeforeEach(func() {
			cleanup = true
			os.Chdir("example-perl")
			appName := GetRandAppName()
			CreateApp(appName, "--buildpack", "https://github.com/miyagawa/heroku-buildpack-perl.git")
			testApp = DeployApp(appName, gitSSH)
		})

		AfterEach(func() {
			defer os.Chdir("..")
			if cleanup {
				DestroyApp(testApp)
			}
		})

		It("can get app info", func() {
			VerifyAppInfo(testApp, testUser)
		})

		It("can open the app's URL", func() {
			VerifyAppOpen(testApp)
		})

	})
})
