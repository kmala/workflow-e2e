package tests

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	. "github.com/deis/workflow-e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Builds", func() {
	Context("with a logged-in user", func() {
		var exampleRepo string
		var exampleImage string
		var testApp App

		BeforeEach(func() {
			exampleRepo = "example-go"
			exampleImage = fmt.Sprintf("deis/%s:latest", exampleRepo)
			testApp.Name = GetRandAppName()
			GitInit()
		})

		AfterEach(func() {
			GitClean()
		})

		Context("with no app", func() {
			It("cannot create a build without existing app", func() {
				cmd, err := Run("deis builds:create %s -a %s", exampleImage, testApp.Name)
				Expect(err).NotTo(HaveOccurred())
				Eventually(cmd.Err).Should(Say("404 Not Found"))
				Eventually(cmd).Should(Exit(1))
			})
		})

		Context("with existing app", func() {

			BeforeEach(func() {
				CreateApp(testApp.Name)
				CreateBuild(exampleImage, testApp)
			})

			It("can list app builds", func() {
				cmd, err := Run("deis builds:list -a %s", testApp.Name)
				Expect(err).NotTo(HaveOccurred())
				Eventually(cmd).Should(Exit(0))
				Eventually(cmd).Should(Say(UuidRegExp))
			})
		})

		Context("with a deployed app", func() {
			var curlCmd Cmd
			var cmdRetryTimeout int
			var procFile string

			BeforeEach(func() {
				cmdRetryTimeout = 10
				procFile = fmt.Sprintf("worker: while true; do echo hi; sleep 3; done")
				testApp.URL = strings.Replace(Url, "deis", testApp.Name, 1)
				CreateApp(testApp.Name, "--no-remote")
				CreateBuild(exampleImage, testApp)
			})

			It("can list app builds", func() {
				cmd, err := Run("deis builds:list -a %s", testApp.Name)
				Expect(err).NotTo(HaveOccurred())
				Eventually(cmd).Should(Exit(0))
				Eventually(cmd).Should(Say(UuidRegExp))
			})

			It("can create a build from an existing image (\"deis pull\")", func() {
				procsListing := ListProcs(testApp).Out.Contents()
				// scrape current processes, should be 1 (cmd)
				Expect(len(ScrapeProcs(testApp.Name, procsListing))).To(Equal(1))

				// curl app to make sure everything OK
				curlCmd = Cmd{CommandLineString: fmt.Sprintf(`curl -sL -w "%%{http_code}\\n" "%s" -o /dev/null`, testApp.URL)}
				Eventually(CmdWithRetry(curlCmd, strconv.Itoa(http.StatusOK), cmdRetryTimeout)).Should(BeTrue())

				DeisPull(exampleImage, testApp, fmt.Sprintf(`--procfile="%s"`, procFile))

				sess, err := Run("deis ps:scale worker=1 -a %s", testApp.Name)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(Say("Scaling processes... but first,"))
				Eventually(sess, DefaultMaxTimeout).Should(Say(`done in \d+s`))
				Eventually(sess).Should(Say("=== %s Processes", testApp.Name))
				Eventually(sess).Should(Exit(0))

				procsListing = ListProcs(testApp).Out.Contents()
				// scrape current processes, should be 2 (1 cmd, 1 worker)
				Expect(len(ScrapeProcs(testApp.Name, procsListing))).To(Equal(2))

				// TODO: https://github.com/deis/workflow-e2e/issues/84
				// "deis logs -a %s", app
				// sess, err = Run("deis logs -a %s", testApp.Name)
				// Expect(err).To(BeNil())
				// Eventually(sess).Should(Say("hi"))
				// Eventually(sess).Should(Exit(0))

				// curl app to make sure everything OK
				curlCmd = Cmd{CommandLineString: fmt.Sprintf(`curl -sL -w "%%{http_code}\\n" "%s" -o /dev/null`, testApp.URL)}
				Eventually(CmdWithRetry(curlCmd, strconv.Itoa(http.StatusOK), cmdRetryTimeout)).Should(BeTrue())

				// can scale cmd down to 0
				sess, err = Run("deis ps:scale cmd=0 -a %s", testApp.Name)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(Say("Scaling processes... but first,"))
				Eventually(sess, DefaultMaxTimeout).Should(Say(`done in \d+s`))
				Eventually(sess).Should(Say("=== %s Processes", testApp.Name))
				Eventually(sess).Should(Exit(0))

				procsListing = ListProcs(testApp).Out.Contents()
				// scrape current processes, should be 1 worker
				Expect(len(ScrapeProcs(testApp.Name, procsListing))).To(Equal(1))

				// with routable 'cmd' process gone, curl should return StatusBadGateway
				curlCmd = Cmd{CommandLineString: fmt.Sprintf(`curl -sL -w "%%{http_code}\\n" "%s" -o /dev/null`, testApp.URL)}
				Eventually(CmdWithRetry(curlCmd, strconv.Itoa(http.StatusBadGateway), cmdRetryTimeout)).Should(BeTrue())
			})
		})
	})
})
