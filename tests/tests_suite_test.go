package tests

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"
	"time"

	. "github.com/deis/workflow-e2e/utils"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestTests(t *testing.T) {
	RegisterFailHandler(Fail)

	enableJunit := os.Getenv("JUNIT")
	if enableJunit == "true" {
		junitReporter := reporters.NewJUnitReporter(filepath.Join(HomeHome, "junit.xml"))
		RunSpecsWithDefaultAndCustomReporters(t, "Deis Workflow", []Reporter{junitReporter})
	} else {
		RunSpecs(t, "Deis Workflow")
	}
}

var (
	randSuffix        = rand.Intn(1000)
	testUser          = fmt.Sprintf("test-%d", randSuffix)
	testPassword      = "asdf1234"
	testEmail         = fmt.Sprintf("test-%d@deis.io", randSuffix)
	testAdminUser     = "admin"
	testAdminPassword = "admin"
	testAdminEmail    = "admin@example.com"
	keyName           = fmt.Sprintf("deiskey-%v", randSuffix)
)

var testRoot, testHome, keyPath, gitSSH string

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(10 * time.Second)

	// use the "deis" executable in the search $PATH
	output, err := exec.LookPath("deis")
	Expect(err).NotTo(HaveOccurred(), output)

	testHome, err = ioutil.TempDir("", "deis-workflow-home")
	Expect(err).NotTo(HaveOccurred())
	os.Setenv("HOME", testHome)

	// register the test-admin user
	RegisterOrLogin(Url, testAdminUser, testAdminPassword, testAdminEmail)

	// verify this user is an admin by running a privileged command
	sess, err := Run("deis users:list")
	Expect(err).To(BeNil())
	Eventually(sess).Should(Exit(0))

	sshDir := path.Join(testHome, ".ssh")

	// register the test user and add a key
	RegisterOrLogin(Url, testUser, testPassword, testEmail)

	keyPath = CreateKey(keyName, testHome)

	// Write out a git+ssh wrapper file to avoid known_hosts warnings
	gitSSH = path.Join(sshDir, "git-ssh")
	sshFlags := "-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
	if Debug {
		sshFlags = sshFlags + " -v"
	}
	ioutil.WriteFile(gitSSH, []byte(fmt.Sprintf(
		"#!/bin/sh\nSSH_ORIGINAL_COMMAND=\"ssh $@\"\nexec /usr/bin/ssh %s -i %s \"$@\"\n",
		sshFlags, keyPath)), 0777)

	sess, err = Run("deis keys:add %s.pub", keyPath)
	Expect(err).To(BeNil())
	Eventually(sess).Should(Exit(0))
	Eventually(sess).Should(Say("Uploading %s.pub to deis... done", keyName))

	time.Sleep(5 * time.Second) // wait for ssh key to propagate
})

var _ = BeforeEach(func() {
	var err error
	var output string

	testRoot, err = ioutil.TempDir("", "deis-workflow-test")
	Expect(err).NotTo(HaveOccurred())

	os.Chdir(testRoot)
	output, err = Execute(`git clone https://github.com/deis/example-go.git`)
	Expect(err).NotTo(HaveOccurred(), output)
	output, err = Execute(`git clone https://github.com/deis/example-perl.git`)
	Expect(err).NotTo(HaveOccurred(), output)

	Login(Url, testUser, testPassword)
})

var _ = AfterEach(func() {
	err := os.RemoveAll(testRoot)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	os.Chdir(testHome)

	Cancel(Url, testUser, testPassword)
	Cancel(Url, testAdminUser, testAdminPassword)

	os.RemoveAll(fmt.Sprintf("~/.ssh/%s*", keyName))

	err := os.RemoveAll(testHome)
	Expect(err).NotTo(HaveOccurred())

	os.Setenv("HOME", HomeHome)
})
