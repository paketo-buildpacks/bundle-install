package bundleinstall_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

const GEMFILE_TEMPLATE = `source 'https://rubygems.org'

ruby %s`

func testGemfileParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path   string
		parser bundleinstall.GemfileParser
	)

	it.Before(func() {
		file, err := ioutil.TempFile("", "Gemfile")
		Expect(err).NotTo(HaveOccurred())
		defer file.Close()

		path = file.Name()

		parser = bundleinstall.NewGemfileParser()
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	context("ParseVersion", func() {
		context("when given different types of versions", func() {
			it("parses the versions correctly", func() {
				versions := []string{
					`"2.6.0"`,
					`"~> 2.6.0"`,
					`"~> 2.7.0"`,
					`'~> 2.7.0'`,
					`'~> 2.10.0'`,
					`'~> 2.10.10'`,
					`'~> 10.0.0'`,
					`'~>10.0.0'`,
					`'~>  10.0.0'`,
					`'~>	10.0.0'`,
					`"< 2.10.10"`,
					`"> 2.10.10"`,
					`"<= 2.10.10"`,
					`">= 2.10.10"`,
					`"= 2.10.10"`,
				}

				for _, v := range versions {
					expectedVersion := strings.Trim(v, `"'`)

					Expect(ioutil.WriteFile(path, []byte(fmt.Sprintf(GEMFILE_TEMPLATE, v)), 0644)).To(Succeed())

					version, err := parser.ParseVersion(path)
					Expect(err).NotTo(HaveOccurred())
					Expect(version).To(Equal(expectedVersion))
				}
			})
		})

		context("when the Gemfile file does not exist", func() {
			it.Before(func() {
				Expect(os.Remove(path)).To(Succeed())
			})

			it("returns an ErrNotExist error", func() {
				_, err := parser.ParseVersion(path)
				Expect(errors.Is(err, os.ErrNotExist)).To(BeTrue())
			})
		})

		context("failure cases", func() {
			context("when the Gemfile cannot be opened", func() {
				it.Before(func() {
					Expect(os.Chmod(path, 0000)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := parser.ParseVersion(path)
					Expect(err).To(MatchError(ContainSubstring("failed to parse Gemfile:")))
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
