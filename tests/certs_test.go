package tests

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	. "github.com/deis/workflow-e2e/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Certs", func() {
	var testApp App
	var domain string
	var certName string
	var certNames []string
	var customSSLEndpoint string
	var exampleRepo = "example-go"

	certPath := path.Join(GetDir(), "files/certs")
	certs := map[string]Cert{
		"www": Cert{
			Name:     "www-foo-com",
			CertPath: fmt.Sprintf("%s/www.foo.com.cert", certPath),
			KeyPath:  fmt.Sprintf("%s/www.foo.com.key", certPath)},
		"wildcard": Cert{
			Name:     "wildcard-foo-com",
			CertPath: fmt.Sprintf("%s/wildcard.foo.com.cert", certPath),
			KeyPath:  fmt.Sprintf("%s/wildcard.foo.com.key", certPath)},
		"foo": Cert{
			Name:     "foo-com",
			CertPath: fmt.Sprintf("%s/foo.com.cert", certPath),
			KeyPath:  fmt.Sprintf("%s/foo.com.key", certPath)},
		"bar": Cert{
			Name:     "bar-com",
			CertPath: fmt.Sprintf("%s/bar.com.cert", certPath),
			KeyPath:  fmt.Sprintf("%s/bar.com.key", certPath)},
	}

	cleanUpCerts := func(certNames []string) {
		certsListing := string(ListCerts().Wait().Out.Contents()[:])
		if !strings.Contains(certsListing, "No certs") {
			RemoveCerts(certNames)
		}
	}

	cleanUpDomains := func(domains []string) {
		for _, domain := range domains {
			RemoveDomain(domain, testApp.Name)
		}
	}

	Context("with an app yet to be deployed", func() {
		BeforeEach(func() {
			GitInit()
			testApp = App{Name: GetRandAppName()}
			CreateApp(testApp.Name)
			domain = getRandDomain()
			certName = strings.Replace(domain, ".", "-", -1)
			certNames = []string{certName}
		})

		AfterEach(func() {
			cleanUpCerts(certNames)
		})

		It("can add, attach, list, and remove certs", func() {
			AddDomain(domain, testApp.Name)

			AddCert(certName, certs["wildcard"].CertPath, certs["wildcard"].KeyPath)

			Eventually(CertsInfo(certName)).Should(Say("No connected domains"))

			AttachCert(certName, domain)

			Eventually(CertsInfo(certName)).Should(Say(domain))
		})
	})

	Context("with a deployed app", func() {
		once := &sync.Once{}

		BeforeEach(func() {
			// Set up the test app only once and assume the suite will clean up.
			once.Do(func() {
				os.Chdir(exampleRepo)
				appName := GetRandAppName()
				CreateApp(appName)
				testApp = DeployApp(appName, gitSSH)
			})
			domain = getRandDomain()
			certName = strings.Replace(domain, ".", "-", -1)
			certNames = []string{certName}

			customSSLEndpoint = strings.Replace(testApp.URL, "http", "https", 1)
			portRegexp := regexp.MustCompile(`:\d+`)
			customSSLEndpoint = portRegexp.ReplaceAllString(customSSLEndpoint, "") // strip port
		})

		AfterEach(func() {
			defer os.Chdir("..")
			cleanUpCerts(certNames)
		})

		It("can specify limit to number of certs returned by certs:list", func() {
			alternateCertName := strings.Replace(getRandDomain(), ".", "-", -1)
			certNames = append(certNames, alternateCertName)
			randDomainRegExp := `my-custom-[0-9]{0,9}-domain-com`

			AddCert(certName, certs["wildcard"].CertPath, certs["wildcard"].KeyPath)
			AddCert(alternateCertName, certs["wildcard"].CertPath, certs["wildcard"].KeyPath)

			sess, err := Run("deis certs:list -l 0")
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say("No certs"))
			Eventually(sess).Should(Exit(0))

			sess, err = Run("deis certs:list --limit=1")
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say(randDomainRegExp))
			Eventually(sess).Should(Not(Say(randDomainRegExp)))
			Eventually(sess).Should(Exit(0))

			sess, err = Run("deis certs:list")
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(Say(randDomainRegExp))
			Eventually(sess).Should(Say(randDomainRegExp))
			Eventually(sess).Should(Exit(0))
		})

		It("can add, attach, list, and remove certs... improperly", func() {
			nonExistentCert := "non-existent.crt"
			nonExistentCertName := "non-existent-cert"

			AddDomain(domain, testApp.Name)

			// attempt to add cert with improper cert name (includes periods)
			sess, err := Run("deis certs:add %s %s %s", domain, certs["wildcard"].CertPath, certs["wildcard"].KeyPath)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("400 Bad Request"))
			Eventually(sess).Should(Exit(1))

			// attempt to add cert with cert and key file swapped
			sess, err = Run("deis certs:add %s %s %s", certName, certs["wildcard"].KeyPath, certs["wildcard"].CertPath)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("400 Bad Request"))
			Eventually(sess).Should(Exit(1))

			// attempt to add cert with non-existent keys
			sess, err = Run("deis certs:add %s %s %s", certName, nonExistentCert, "non-existent.key")
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("open %s: no such file or directory", nonExistentCert))
			Eventually(sess).Should(Exit(1))

			// attempt to remove non-existent cert
			sess, err = Run("deis certs:remove %s", nonExistentCertName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("404 Not Found"))
			Eventually(sess).Should(Exit(1))

			// attempt to get info on non-existent cert
			sess, err = Run("deis certs:info %s", nonExistentCertName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("404 Not Found"))
			Eventually(sess).Should(Exit(1))

			// attempt to attach non-existent cert
			sess, err = Run("deis certs:attach %s %s", nonExistentCertName, domain)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("404 Not Found"))
			Eventually(sess).Should(Exit(1))

			// attempt to detach non-existent cert
			sess, err = Run("deis certs:detach %s %s", nonExistentCertName, domain)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("404 Not Found"))
			Eventually(sess).Should(Exit(1))

			AddCert(certName, certs["wildcard"].CertPath, certs["wildcard"].KeyPath)

			// attempt to attach to non-existent domain
			sess, err = Run("deis certs:attach %s %s", certName, "non-existent-domain")
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("404 Not Found"))
			Eventually(sess).Should(Exit(1))

			// attempt to detach from non-existent domain
			sess, err = Run("deis certs:detach %s %s", certName, "non-existent-domain")
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("404 Not Found"))
			Eventually(sess).Should(Exit(1))

			// attempt to remove non-existent cert
			sess, err = Run("deis certs:remove %s", nonExistentCertName)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess.Err).Should(Say("404 Not Found"))
			Eventually(sess).Should(Exit(1))
		})

		Context("multiple domains and certs", func() {
			domains := map[string]string{
				"wildcard": "*.foo.com",
				"foo":      "foo.com",
				"bar":      "bar.com",
			}
			domainNames := []string{domains["wildcard"], domains["foo"], domains["bar"]}

			AfterEach(func() {
				// need to cleanup domains as they are not named randomly as above
				cleanUpDomains(domainNames)
			})

			It("can attach/detach 2 certs (1 wildcard) to/from 3 domains (1 wildcard)", func() {
				sharedCert := certs["wildcard"]
				certNames = []string{sharedCert.Name, certs["bar"].Name}

				// Add all 3 domains
				for _, domain := range domains {
					AddDomain(domain, testApp.Name)
				}

				// Add 2 certs
				AddCert(sharedCert.Name, sharedCert.CertPath, sharedCert.KeyPath)
				AddCert(certs["bar"].Name, certs["bar"].CertPath, certs["bar"].KeyPath)

				// Share wildcard cert betwtixt two domains, attach the other
				for _, domain := range []string{domains["wildcard"], domains["foo"]} {
					AttachCert(sharedCert.Name, domain)
				}
				AttachCert(certs["bar"].Name, domains["bar"])

				// With multiple strings to check, use substrings as ordering is non-deterministic
				// (Should(Say()) enforces strict ordering)
				bothDomains := fmt.Sprintf("%s,%s", domains["wildcard"], domains["foo"])
				Eventually(CertsInfo(sharedCert.Name).Wait().Out.Contents()).Should(ContainSubstring(bothDomains))
				Eventually(CertsInfo(certs["bar"].Name)).Should(Say(domains["bar"]))

				// All SSL endpoints should be good to go
				for _, domain := range domains {
					VerifySSLEndpoint(customSSLEndpoint, domain, http.StatusOK)
				}

				// Detach shared cert from one domain and re-check endpoints
				DetachCert(sharedCert.Name, domains["wildcard"])
				Eventually(CertsInfo(sharedCert.Name)).Should(Say(domains["foo"]))
				VerifySSLEndpoint(customSSLEndpoint, domains["wildcard"], http.StatusNotFound)
				VerifySSLEndpoint(customSSLEndpoint, domains["foo"], http.StatusOK)

				DetachCert(certs["bar"].Name, domains["bar"])
				VerifySSLEndpoint(customSSLEndpoint, domains["bar"], http.StatusNotFound)
			})

			getOtherDomains := func(myDomain string, domains map[string]string) []string {
				otherDomains := make([]string, 0, len(domains)-1)

				for _, domain := range domains {
					if domain != myDomain {
						otherDomains = append(otherDomains, domain)
					}
				}
				return otherDomains
			}

			DescribeTable("3 certs (no wildcards), 3 domains (1 wildcard)",

				func(domain, certName, cert, key string) {
					certNames = []string{certName}
					domainNames = []string{domain}

					AddDomain(domain, testApp.Name)

					AddCert(certName, cert, key)

					AttachCert(certName, domain)

					Eventually(CertsInfo(certName).Wait().Out.Contents()).Should(ContainSubstring(domain))

					VerifySSLEndpoint(customSSLEndpoint, domain, http.StatusOK)
					for _, otherDomain := range getOtherDomains(domain, domains) {
						VerifySSLEndpoint(customSSLEndpoint, otherDomain, http.StatusNotFound)
					}

					DetachCert(certName, domain)

					Eventually(CertsInfo(certName)).Should(Say("No connected domains"))

					VerifySSLEndpoint(customSSLEndpoint, domain, http.StatusNotFound)
				},

				Entry("a non-wildcard cert to a wildcard domain",
					domains["wildcard"], certs["www"].Name, certs["www"].CertPath, certs["www"].KeyPath),
				Entry("a non-wildcard cert to a non-wildcard domain",
					domains["foo"], certs["foo"].Name, certs["foo"].CertPath, certs["foo"].KeyPath),
				Entry("a non-wildcard cert to a non-wildcard domain",
					domains["bar"], certs["bar"].Name, certs["bar"].CertPath, certs["bar"].KeyPath),
			)
		})
	})
})
