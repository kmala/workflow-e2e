package tests

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"

	. "github.com/deis/workflow-e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Processes", func() {

	Context("with a deployed app", func() {

		var testApp App
		once := &sync.Once{}

		BeforeEach(func() {
			// Set up the Processes test app only once and assume the suite will clean up.
			once.Do(func() {
				os.Chdir("example-go")
				appName := GetRandAppName()
				CreateApp(appName)
				testApp = DeployApp(appName, gitSSH)
			})
		})

		PDescribeTable("can scale up and down",

			func(scaleTo, respCode int) {
				// TODO: need some way to choose between "web" and "cmd" here!
				// scale the app's processes to the desired number
				sess, err := Run("deis ps:scale web=%d --app=%s", scaleTo, testApp.Name)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(Say("Scaling processes... but first,"))
				Eventually(sess, DefaultMaxTimeout).Should(Say(`done in \d+s`))
				Eventually(sess).Should(Say("=== %s Processes", testApp.Name))
				Eventually(sess).Should(Exit(0))

				// test that there are the right number of processes listed
				procsListing := ListProcs(testApp).Out.Contents()
				procs := ScrapeProcs(testApp.Name, procsListing)
				Expect(len(procs)).To(Equal(scaleTo))

				// curl the app's root URL and print just the HTTP response code
				sess, err = Run(`curl -sL -w "%%{http_code}\\n" "%s" -o /dev/null`, testApp.URL)
				Eventually(sess).Should(Say(strconv.Itoa(respCode)))
				Eventually(sess).Should(Exit(0))
			},

			Entry("scales to 1", 1, 200),
			Entry("scales to 3", 3, 200),
			Entry("scales to 0", 0, 502),
			Entry("scales to 5", 5, 200),
			Entry("scales to 0", 0, 502),
			Entry("scales to 1", 1, 200),
		)

		DescribeTable("can restart processes",

			func(restart string, scaleTo int, respCode int) {
				// TODO: need some way to choose between "web" and "cmd" here!
				// scale the app's processes to the desired number
				sess, err := Run("deis ps:scale web=%d --app=%s", scaleTo, testApp.Name)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(Say("Scaling processes... but first,"))
				Eventually(sess, DefaultMaxTimeout).Should(Say(`done in \d+s`))
				Eventually(sess).Should(Say("=== %s Processes", testApp.Name))
				Eventually(sess).Should(Exit(0))

				// capture the process names
				beforeProcs := ScrapeProcs(testApp.Name, sess.Out.Contents())

				// restart the app's process(es)
				var arg string
				switch restart {
				case "all":
					arg = ""
				case "by type":
					// TODO: need some way to choose between "web" and "cmd" here!
					arg = "web"
				case "by wrong type":
					// TODO: need some way to choose between "web" and "cmd" here!
					arg = "cmd"
				case "one":
					procsLen := len(beforeProcs)
					Expect(procsLen).To(BeNumerically(">", 0))
					arg = beforeProcs[rand.Intn(procsLen)]
				}
				sess, err = Run("deis ps:restart %s --app=%s", arg, testApp.Name)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(Say("Restarting processes... but first,"))
				if scaleTo == 0 || restart == "by wrong type" {
					Eventually(sess).Should(Say("Could not find any processes to restart"))
				} else {
					Eventually(sess, DefaultMaxTimeout).Should(Say(`done in \d+s`))
					Eventually(sess).Should(Say("=== %s Processes", testApp.Name))
				}
				Eventually(sess).Should(Exit(0))

				// capture the process names
				procsListing := ListProcs(testApp).Out.Contents()
				afterProcs := ScrapeProcs(testApp.Name, procsListing)

				// compare the before and after sets of process names
				Expect(len(afterProcs)).To(Equal(scaleTo))
				if scaleTo > 0 && restart != "by wrong type" {
					Expect(beforeProcs).NotTo(Equal(afterProcs))
				}

				// curl the app's root URL and print just the HTTP response code
				maxRetryIterations := 15
				curlCmd := Cmd{CommandLineString: fmt.Sprintf(`curl -sL -w "%%{http_code}\\n" "%s" -o /dev/null`, testApp.URL)}
				Eventually(CmdWithRetry(curlCmd, strconv.Itoa(respCode), maxRetryIterations)).Should(BeTrue())

			},

			Entry("restarts one of 1", "one", 1, 200),
			Entry("restarts all of 1", "all", 1, 200),
			Entry("restarts all of 1 by type", "by type", 1, 200),
			Entry("restarts all of 1 by wrong type", "by wrong type", 1, 200),
			Entry("restarts one of 6", "one", 6, 200),
			Entry("restarts all of 6", "all", 6, 200),
			Entry("restarts all of 6 by type", "by type", 6, 200),
			Entry("restarts all of 6 by wrong type", "by wrong type", 6, 200),
			PEntry("restarts all of 0", "all", 0, 502),
			PEntry("restarts all of 0 by type", "by type", 0, 502),
			PEntry("restarts all of 0 by wrong type", "by wrong type", 0, 502),
		)
	})
})
