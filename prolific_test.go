package main_test

import (
	"bytes"
	"encoding/csv"
	"os"
	"os/exec"
	"fmt"
  "io"
	"io/ioutil"

	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/gbytes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Prolific", func() {
	const max_tasks = 8
	var session *gexec.Session
	var err error

	AfterEach(func() {
		os.Remove("stories.prolific")
	})

	Describe("prolific help", func() {
		Context("given a 'help' argument", func() {
			It("emits usage information", func() {
				cmd := exec.Command(prolific, "help")
				session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Ω(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Ω(session.Out).Should(gbytes.Say("Usage:"))
			})
		})

		Context("given no argument", func() {
			It("emits usage information", func() {
				cmd := exec.Command(prolific)
				session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Ω(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Ω(session.Out).Should(gbytes.Say("Usage:"))
			})
		})
	})

	Describe("prolific template", func() {
		BeforeEach(func() {
			cmd := exec.Command(prolific, "template")
			session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Ω(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
		})

		It("should generate a template file", func() {
			_, err := os.Stat("stories.prolific")
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("generating csv output", func() {
		BeforeEach(func() {
			cmd := exec.Command(prolific, "template")
			session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Ω(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
		})

		Describe("reading from file", func() {
			BeforeEach(func() {
				cmd := exec.Command(prolific, "stories.prolific")
				session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Ω(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
			})

			It("should convert the passed-in prolific file", func() {
				reader := csv.NewReader(bytes.NewReader(session.Out.Contents()))
				records, err := reader.ReadAll()
				Ω(err).ShouldNot(HaveOccurred())

				By("emitting a header line")
				Ω(records[0]).Should(Equal([]string{"Title", "Type", "Description", "Labels", "Task", "Task", "Task", "Task", "Task", "Task", "Task", "Task"}))

				By("parsing all entries")
				Ω(records).Should(HaveLen(7))

				var TITLE, TYPE, DESCRIPTION, LABELS, TASK1 = 0, 1, 2, 3, 4

				By("parsing all relevant fields")
				Ω(records[1][TITLE]).Should(Equal("As a user I can toast a bagel"))
				Ω(records[1][TYPE]).Should(Equal("feature"))
				Ω(records[1][DESCRIPTION]).Should(Equal("When I insert a bagel into toaster and press the on button, I should get a toasted bagel"))
				Ω(records[1][LABELS]).Should(Equal("mvp,toasting"))

				By("handling types correctly")
				Ω(records[3][TYPE]).Should(Equal("feature"))
				Ω(records[4][TYPE]).Should(Equal("bug"))
				Ω(records[5][TYPE]).Should(Equal("chore"))
				Ω(records[6][TYPE]).Should(Equal("release"))

				By("handling empty descriptions correctly")
				Ω(records[4][DESCRIPTION]).Should(BeEmpty())

				By("handling labels correctly")
				Ω(records[3][LABELS]).Should(Equal("mvp,clean-up"))
				Ω(records[5][LABELS]).Should(BeEmpty())
				Ω(records[6][LABELS]).Should(Equal("mvp"))

				By("handling absence of tasks correctly")
				for j := 0 ; j < max_tasks ; j++ {
					Ω(records[1][TASK1 + j]).Should(Equal(""))
				}

				By("handling some tasks correctly")
				for j := 0 ; j < 3 ; j++ {
					Ω(records[5][TASK1 + j]).Should(Equal(fmt.Sprintf("task %d", j+1)))
				}
				for j := 3 ; j < max_tasks ; j++ {
					Ω(records[5][TASK1 + j]).Should(Equal(""))
				}
			})
		})

		Describe("reading from stdin", func() {
			BeforeEach(func() {
				cmd := exec.Command(prolific)
				stdin, err := cmd.StdinPipe()
				Ω(err).ShouldNot(HaveOccurred())

				prolific_content, err := ioutil.ReadFile("stories.prolific")
				Ω(err).ShouldNot(HaveOccurred())
				_, err = stdin.Write(prolific_content)
				Ω(err).ShouldNot(HaveOccurred())

				session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Ω(err).ShouldNot(HaveOccurred())

				err = stdin.Close()
				Ω(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
			})

			It("should convert the passed-in prolific content", func() {
				reader := csv.NewReader(bytes.NewReader(session.Out.Contents()))
				records, err := reader.ReadAll()
				Ω(err).ShouldNot(HaveOccurred())

				By("emitting a header line")
				Ω(records[0]).Should(Equal([]string{"Title", "Type", "Description", "Labels", "Task", "Task", "Task", "Task", "Task", "Task", "Task", "Task"}))

				By("parsing all entries")
				Ω(records).Should(HaveLen(7))
			})
		})

		Describe("tasks", func() {
			var cmd *exec.Cmd
			var stdin io.WriteCloser

			BeforeEach(func() {
				cmd = exec.Command(prolific)
				stdin, err = cmd.StdinPipe()
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("max tasks", func() {
				const story = `As a user I can toast a bagel

When I insert a bagel into toaster and press the on button, I should get a toasted bagel

- [ ] task 1
* [ ] task 2
- [ ] task 3
* [ ] task 4
- [ ] task 5
* [ ] task 6
- [ ] task 7
* [ ] task 8
`
				It("populates all task columns", func() {
					_, err = stdin.Write([]byte(story))
					Ω(err).ShouldNot(HaveOccurred())

					session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Ω(err).ShouldNot(HaveOccurred())

					err = stdin.Close()
					Ω(err).ShouldNot(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0))

					reader := csv.NewReader(bytes.NewReader(session.Out.Contents()))
					records, err := reader.ReadAll()
					Ω(err).ShouldNot(HaveOccurred())

					Ω(records[0]).Should(Equal([]string{"Title", "Type", "Description", "Labels", "Task", "Task", "Task", "Task", "Task", "Task", "Task", "Task"}))
					Ω(records).Should(HaveLen(2))

					var TASK1 = 4
					for j := 0 ; j < 8 ; j++ {
						Ω(records[1][TASK1 + j]).Should(Equal(fmt.Sprintf("task %d", j+1)))
					}
				})
			})

			Context("greater than max tasks", func() {
				const story = `As a user I can toast a bagel

When I insert a bagel into toaster and press the on button, I should get a toasted bagel

- [ ] task 1
* [ ] task 2
- [ ] task 3
* [ ] task 4
- [ ] task 5
* [ ] task 6
- [ ] task 7
* [ ] task 8
* [ ] task 9
`

				It("raises an error", func() {
					_, err = stdin.Write([]byte(story))
					Ω(err).ShouldNot(HaveOccurred())

					session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Ω(err).ShouldNot(HaveOccurred())

					err = stdin.Close()
					Ω(err).ShouldNot(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0))

					Ω(session.Err.Contents()).Should(ContainSubstring("There were errors parsing your file"))
					Ω(session.Err.Contents()).Should(ContainSubstring("Story has more than 8 tasks; consider breaking it into smaller stories:"))
				})
			})
		})
	})
})
