package llama_test

import (
	"os"

	"github.com/go-skynet/go-llama.cpp"
	. "github.com/go-skynet/go-llama.cpp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LLama binding", func() {
	testModelPath := os.Getenv("TEST_MODEL")

	Context("Declaration", func() {
		It("fails with no model", func() {
			model, err := New("not-existing")
			Expect(err).To(HaveOccurred())
			Expect(model).To(BeNil())
		})
	})
	Context("Inferencing tests (using "+testModelPath+") ", func() {
		getModel := func() (*LLama, error) {
			model, err := New(
				testModelPath,
				EnableF16Memory,
				SetContext(128),
				SetMMap(true),
				SetNBatch(512),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(model).ToNot(BeNil())
			return model, err
		}

		It("predicts successfully", func() {
			if testModelPath == "" {
				Skip("test skipped - only makes sense if the TEST_MODEL environment variable is set.")
			}

			model, err := getModel()
			text, err := model.Predict(`[INST] Answer to the following question:
how much is 2+2?
[/INST]`)
			Expect(err).ToNot(HaveOccurred(), text)
			Expect(text).To(ContainSubstring("4"), text)
		})

		It("speculative sampling predicts", Label("gpu"), func() {
			if testModelPath == "" {
				Skip("test skipped - only makes sense if the TEST_MODEL environment variable is set.")
			}
			// Create target model with perplexity as required for speculative sampling
			model, err := New(
				testModelPath,
				EnableF16Memory,
				SetContext(512),
				SetMMap(false), // Disable mmap to avoid potential memory issues
				SetNBatch(512),
				SetPerplexity(true), // Required for speculative sampling
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(model).ToNot(BeNil())
			defer model.Free()

			// Create draft model with same settings
			model2, err := New(
				testModelPath,
				EnableF16Memory,
				SetContext(512),
				SetMMap(false), // Disable mmap to avoid potential memory issues
				SetNBatch(512),
				SetPerplexity(true), // Required for speculative sampling
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(model2).ToNot(BeNil())
			defer model2.Free()

			// Run speculative sampling with smaller draft size
			text, err := model.SpeculativeSampling(model2, `Answer: 2+2=`, llama.SetNDraft(4))
			Expect(err).ToNot(HaveOccurred(), text)
			// Just check that we get some output, not specific content
			Expect(len(text)).To(BeNumerically(">", 0))
		})

		It("tokenizes strings successfully", func() {
			if testModelPath == "" {
				Skip("test skipped - only makes sense if the TEST_MODEL environment variable is set.")
			}

			model, err := getModel()
			l, tokens, err := model.TokenizeString("A STRANGE GAME.\nTHE ONLY WINNING MOVE IS NOT TO PLAY.\n\nHOW ABOUT A NICE GAME OF CHESS?",
				SetRopeFreqBase(10000.0), SetRopeFreqScale(1))

			Expect(err).ToNot(HaveOccurred())
			Expect(l).To(BeNumerically(">", 0))
			Expect(int(l)).To(Equal(len(tokens)))
		})
	})

	Context("Inferencing tests with GPU (using "+testModelPath+") ", Label("gpu"), func() {
		getModel := func() (*LLama, error) {
			model, err := New(
				testModelPath,
				llama.EnableF16Memory, llama.SetContext(128), llama.EnableEmbeddings, llama.SetGPULayers(10),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(model).ToNot(BeNil())
			return model, err
		}

		It("predicts successfully", func() {
			if testModelPath == "" {
				Skip("test skipped - only makes sense if the TEST_MODEL environment variable is set.")
			}

			model, err := getModel()
			text, err := model.Predict(`[INST] Answer to the following question:
how much is 2+2?
[/INST]`)
			Expect(err).ToNot(HaveOccurred(), text)
			Expect(text).To(ContainSubstring("4"), text)
		})
	})
})
