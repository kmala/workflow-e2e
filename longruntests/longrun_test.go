package longruntests_test

import (
	"os"
	"time"

	. "github.com/deis/workflow-e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

func pingApp(pingChan chan<- bool, testApp App) {
	defer GinkgoRecover()
	defer func() { pingChan <- true }()
	startTime := time.Now()
	duration := time.Duration(1) * time.Minute
	for time.Since(startTime) <= duration {
		sess, err := Run(`curl -sL -w "%%{http_code}\\n" "%s" -o /dev/null`, testApp.URL)
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(Say("200"))
		Eventually(sess).Should(Exit(0))
	}
}

func scaleApp(scaleChan chan<- bool, testApp App) {
	scaleTo := 5
	defer GinkgoRecover()
	defer func() { scaleChan <- true }()
	startTime := time.Now()
	duration := time.Duration(1) * time.Minute
	for time.Since(startTime) <= duration {
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
		if scaleTo == 5 {
			scaleTo = 1
		} else {
			scaleTo = 5
		}
		time.Sleep(5 * time.Second)
	}
}

var _ = Describe("Longrun", func() {

	DescribeTable("Run example app", func(exampleApp string) {
		var testApp App
		os.Chdir(exampleApp)
		for {
			appName := GetRandAppName()
			CreateApp(appName)
			testApp = DeployApp(appName, gitSSH)

			sess, err := Run("deis apps:create %s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("App with this id already exists."))
			Eventually(sess).ShouldNot(Exit(0))

			VerifyAppInfo(testApp, testUser)

			VerifyAppOpen(testApp)

			sess, err = Run("deis apps:run echo Hello, 世界")
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Out, (1 * time.Minute)).Should(Say("Hello, 世界"))
			numUsers := 100
			pingChan := make(chan bool, numUsers)
			for i := 1; i <= numUsers; i++ {
				go pingApp(pingChan, testApp)
			}
			scaleChan := make(chan bool, 1)
			go scaleApp(scaleChan, testApp)
			<-scaleChan
			for i := 1; i <= numUsers; i++ {
				<-pingChan
			}
			DestroyApp(testApp)
			sess, err = Run("git remote rm deis")
			time.Sleep(5 * time.Second)
		}
	},
		Entry("with example-go", "example-go"),
		Entry("with example-php", "example-php"),
		Entry("with example-python-django", "example-python-django"),
		Entry("with example-dockerfile-http", "example-dockerfile-http"),
	)
})
