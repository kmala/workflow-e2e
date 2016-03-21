package longruntests_test

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

func listProcs(testApp App) *Session {
	sess, err := start("deis ps:list --app=%s", testApp.Name)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess).Should(Say("=== %s Processes", testApp.Name))
	Eventually(sess).Should(Exit(0))
	return sess
}

// scrapeProcs returns the sorted process names for an app from the given output.
// It matches the current "deis ps" output for a healthy container:
//   earthy-vocalist-v2-cmd-1d73e up (v2)
//   myapp-v16-web-bujlq up (v16)
func scrapeProcs(app string, output []byte) []string {
	re := regexp.MustCompile(fmt.Sprintf(procsRegexp, app))
	found := re.FindAllSubmatch(output, -1)
	procs := make([]string, len(found))
	for i := range found {
		procs[i] = string(found[i][1])
	}
	sort.Strings(procs)
	return procs
}

func pingApp(pingChan chan<- bool, testApp App) {
	defer GinkgoRecover()
	defer func() { pingChan <- true }()
	startTime := time.Now()
	duration := time.Duration(1) * time.Minute
	for time.Since(startTime) <= duration {
		sess, err := start(`curl -sL -w "%%{http_code}\\n" "%s" -o /dev/null`, testApp.URL)
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
		sess, err := start("deis ps:scale web=%d --app=%s", scaleTo, testApp.Name)
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess).Should(Say("Scaling processes... but first,"))
		Eventually(sess, defaultMaxTimeout).Should(Say(`done in \d+s`))
		Eventually(sess).Should(Say("=== %s Processes", testApp.Name))
		Eventually(sess).Should(Exit(0))

		// test that there are the right number of processes listed
		procsListing := listProcs(testApp).Out.Contents()
		procs := scrapeProcs(testApp.Name, procsListing)
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

	It("Run example Go app", func() {
		var testApp App
		os.Chdir("example-go")
		for {
			appName := getRandAppName()
			createApp(appName)
			testApp = deployApp(appName)

			sess, err := start("deis apps:create %s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("App with this id already exists."))
			Eventually(sess).ShouldNot(Exit(0))

			verifyAppInfo(testApp)

			verifyAppOpen(testApp)

			sess, err = start("deis apps:run echo Hello, 世界")
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
			destroyApp(testApp)
			sess, err = start("git remote rm deis")
			time.Sleep(5 * time.Second)
		}
	})
	It("Run example php app", func() {
		var testApp App
		os.Chdir("example-php")
		for {
			appName := getRandAppName()
			createApp(appName)
			testApp = deployApp(appName)

			sess, err := start("deis apps:create %s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("App with this id already exists."))
			Eventually(sess).ShouldNot(Exit(0))

			verifyAppInfo(testApp)

			verifyAppOpen(testApp)

			sess, err = start("deis apps:run echo Hello, 世界")
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
			destroyApp(testApp)
			sess, err = start("git remote rm deis")
			time.Sleep(5 * time.Second)
		}
	})
	It("Run example python-django app", func() {
		var testApp App
		os.Chdir("example-python-django")
		for {
			appName := getRandAppName()
			createApp(appName)
			testApp = deployApp(appName)

			sess, err := start("deis apps:create %s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("App with this id already exists."))
			Eventually(sess).ShouldNot(Exit(0))

			verifyAppInfo(testApp)

			verifyAppOpen(testApp)

			sess, err = start("deis apps:run echo Hello, 世界")
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
			destroyApp(testApp)
			sess, err = start("git remote rm deis")
			time.Sleep(5 * time.Second)
		}
	})
	It("Run example dockerfile app", func() {
		var testApp App
		os.Chdir("example-dockerfile-http")
		for {
			appName := getRandAppName()
			createApp(appName)
			testApp = deployApp(appName)

			sess, err := start("deis apps:create %s", testApp.Name)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("App with this id already exists."))
			Eventually(sess).ShouldNot(Exit(0))

			verifyAppInfo(testApp)

			verifyAppOpen(testApp)

			sess, err = start("deis apps:run echo Hello, 世界")
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
			destroyApp(testApp)
			sess, err = start("git remote rm deis")
			time.Sleep(5 * time.Second)
		}
	})
})
