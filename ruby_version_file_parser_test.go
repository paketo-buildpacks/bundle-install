package bundleinstall_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	bundleinstall "github.com/paketo-buildpacks/bundle-install"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testRubyVersionFileParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path   string
		parser bundleinstall.RubyVersionFileParser
	)

	it.Before(func() {
		path = filepath.Join(t.TempDir(), ".ruby-version")
		_, err := os.Create(path)
		Expect(err).NotTo(HaveOccurred())
		parser = bundleinstall.NewRubyVersionFileParser()
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	context("ParseVersion", func() {
		context("when the version is just a single Major value", func() {
			it.Before(func() {
				Expect(os.WriteFile(path, []byte("2\n"), 0644)).To(Succeed())
			})

			it("parses correctly", func() {
				version, err := parser.ParseVersion(path)
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("2"))
			})
		})

		context("when the version is a Major.Minor value", func() {
			it.Before(func() {
				Expect(os.WriteFile(path, []byte("2.6\n"), 0644)).To(Succeed())
			})

			it("parses correctly", func() {
				version, err := parser.ParseVersion(path)
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("2.6"))
			})
		})

		context("when the version is a Major.Minor.Patch value", func() {
			it.Before(func() {
				Expect(os.WriteFile(path, []byte("2.6.3\n"), 0644)).To(Succeed())
			})

			it("parses correctly", func() {
				version, err := parser.ParseVersion(path)
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("2.6.3"))
			})
		})

		context("when the .ruby-version file does not exist", func() {
			it.Before(func() {
				Expect(os.Remove(path)).To(Succeed())
			})

			it("returns an ErrNotExist error", func() {
				_, err := parser.ParseVersion(path)
				Expect(errors.Is(err, os.ErrNotExist)).To(BeTrue())
			})
		})

		context("failure cases", func() {
			context("when the .ruby-version file cannot be opened", func() {
				it.Before(func() {
					Expect(os.Chmod(path, 0000)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := parser.ParseVersion(path)
					Expect(err).To(MatchError(ContainSubstring("failed to read .ruby-version file:")))
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the .ruby-version file contains an invalid version", func() {
				it.Before(func() {
					Expect(os.WriteFile(path, []byte("invalid.version\n"), 0644)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := parser.ParseVersion(path)
					Expect(err).To(MatchError(ContainSubstring("no valid ruby version found in .ruby-version file:")))
				})
			})

		})
	})
}
