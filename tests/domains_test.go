package tests

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"

	. "github.com/deis/workflow-e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

func getRandDomain() string {
	return fmt.Sprintf("my-custom-%d.domain.com", rand.Intn(999999999))
}

var _ = Describe("Domains", func() {
	var testApp App
	var domain string

	Context("with app yet to be deployed", func() {

		BeforeEach(func() {
			domain = getRandDomain()
			GitInit()

			testApp.Name = GetRandAppName()
			CreateApp(testApp.Name)
		})

		AfterEach(func() {
			DestroyApp(testApp)
			GitClean()
		})

		It("can list domains", func() {
			sess, err := Run("deis domains:list --app=%s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("=== %s Domains", testApp.Name))
			Eventually(sess).Should(Say("%s", testApp.Name))
			Eventually(sess).Should(Exit(0))
		})

		It("can add and remove domains", func() {
			sess, err := Run("deis domains:add %s --app=%s", domain, testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("Adding %s to %s...", domain, testApp.Name))
			Eventually(sess).Should(Say("done"))
			Eventually(sess).Should(Exit(0))

			sess, err = Run("deis domains:remove %s --app=%s", domain, testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("Removing %s from %s...", domain, testApp.Name))
			Eventually(sess).Should(Say("done"))
		})
	})

	Context("with a deployed app", func() {
		var curlCmd Cmd
		var cmdRetryTimeout int

		BeforeEach(func() {
			cmdRetryTimeout = 15
			domain = getRandDomain()
			os.Chdir("example-go")
			appName := GetRandAppName()
			CreateApp(appName)
			testApp = DeployApp(appName, gitSSH)
		})

		AfterEach(func() {
			defer os.Chdir("..")
			DestroyApp(testApp)
		})

		It("can add, list, and remove domains", func() {
			sess, err := Run("deis domains:list --app=%s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("=== %s Domains", testApp.Name))
			Eventually(sess).Should(Say("%s", testApp.Name))
			Eventually(sess).Should(Exit(0))

			sess, err = Run("deis domains:add %s --app=%s", domain, testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("Adding %s to %s...", domain, testApp.Name))
			Eventually(sess).Should(Say("done"))
			Eventually(sess).Should(Exit(0))

			sess, err = Run("deis domains:list --app=%s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("=== %s Domains", testApp.Name))
			Eventually(sess).Should(Say("%s", domain))
			Eventually(sess).Should(Exit(0))

			// curl app at both root and custom domain, both should return http.StatusOK
			curlCmd = Cmd{CommandLineString: fmt.Sprintf(`curl -sL -w "%%{http_code}\\n" "%s" -o /dev/null`, testApp.URL)}
			Eventually(CmdWithRetry(curlCmd, strconv.Itoa(http.StatusOK), cmdRetryTimeout)).Should(BeTrue())
			curlCmd = Cmd{CommandLineString: fmt.Sprintf(`curl -sL -H "Host: %s" -w "%%{http_code}\\n" "%s" -o /dev/null`, domain, testApp.URL)}
			Eventually(CmdWithRetry(curlCmd, strconv.Itoa(http.StatusOK), cmdRetryTimeout)).Should(BeTrue())

			sess, err = Run("deis domains:remove %s --app=%s", domain, testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("Removing %s from %s...", domain, testApp.Name))
			Eventually(sess).Should(Say("done"))
			Eventually(sess).Should(Exit(0))

			// attempt to remove non-existent domain
			sess, err = Run("deis domains:remove %s --app=%s", domain, testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("404 Not Found"))
			Eventually(sess).Should(Exit(1))

			sess, err = Run("deis domains:list --app=%s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("=== %s Domains", testApp.Name))
			Eventually(sess).Should(Say("%s", testApp.Name))
			Eventually(sess).Should(Not(Say("%s", domain)))
			Eventually(sess).Should(Exit(0))

			// curl app at both root and custom domain, custom should return http.StatusNotFound
			curlCmd = Cmd{CommandLineString: fmt.Sprintf(`curl -sL -w "%%{http_code}\\n" "%s" -o /dev/null`, testApp.URL)}
			Eventually(CmdWithRetry(curlCmd, strconv.Itoa(http.StatusOK), cmdRetryTimeout)).Should(BeTrue())
			curlCmd = Cmd{CommandLineString: fmt.Sprintf(`curl -sL -H "Host: %s" -w "%%{http_code}\\n" "%s" -o /dev/null`, domain, testApp.URL)}
			Eventually(CmdWithRetry(curlCmd, strconv.Itoa(http.StatusNotFound), cmdRetryTimeout)).Should(BeTrue())
		})
	})
})
