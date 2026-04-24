package main

// Scratch test program to verify environment override of the prefix for
// subject names made using `MakeSubject()`.
//
// Some example executions:
//
// 1. Without specifying the environment variable
//
// $ SNX_NATS_ALLOW_DEFAULT_SUBJECTS_PREFIX=true go run lib/db/nats/scratch/test_env_subjects.go Sub1 Sub2 MySpecialSubject
// SNX_NATS_SUBJECTS_PREFIX: (not set)
// Input: 'Sub1' -> Full subject: 'snx-v1.Sub1'
// Input: 'Sub2' -> Full subject: 'snx-v1.Sub2'
// Input: 'MySpecialSubject' -> Full subject: 'snx-v1.MySpecialSubject'
//
// 2. With the environment variable set
//
// $ SNX_NATS_ALLOW_DEFAULT_SUBJECTS_PREFIX=false SNX_NATS_SUBJECTS_PREFIX=XXX go run lib/db/nats/scratch/test_env_subjects.go Sub1 Sub2 MySpecialSubject
// SNX_NATS_SUBJECTS_PREFIX: 'XXX'
// Input: 'Sub1' -> Full subject: 'XXX.Sub1'
// Input: 'Sub2' -> Full subject: 'XXX.Sub2'
// Input: 'MySpecialSubject' -> Full subject: 'XXX.MySpecialSubject'
//
// 3. With the environment variable set to an empty string
//
// $ SNX_NATS_ALLOW_DEFAULT_SUBJECTS_PREFIX=false SNX_NATS_SUBJECTS_PREFIX= go run lib/db/nats/scratch/test_env_subjects.go Sub1 Sub2 MySpecialSubject
// SNX_NATS_SUBJECTS_PREFIX: ''
// Input: 'Sub1' -> Full subject: 'Sub1'
// Input: 'Sub2' -> Full subject: 'Sub2'
// Input: 'MySpecialSubject' -> Full subject: 'MySpecialSubject'

import (
	"fmt"
	"os"

	libCLImate "github.com/synesissoftware/libCLImate.Go"

	snx_lib_db_nats "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/nats"
)

func main() {

	// Initialize climate with DSL-style configuration
	climate, _ := libCLImate.Init(func(cl *libCLImate.Climate) error {

		cl.Version = []int{0, 0, 1}
		cl.ValuesString = "subject-1 [ . . . <subject-N> ]"
		cl.InfoLines = []string{
			"Synthetix Development Tools",
			"Test program to verify environment override of the prefix for subject names made using `MakeSubject()`",
			":version:",
			"",
		}

		// Set value constraints (1 or more required)
		cl.ValuesConstraint = []int{1, -1} // minimum 1, no maximum
		cl.ValueNames = []string{
			"subject-1",
		}

		return nil
	}, libCLImate.InitFlag_PanicOnFailure)

	r, _ := climate.ParseAndVerify(os.Args, libCLImate.ParseFlag_PanicOnFailure)

	// Process each subject argument
	for _, subject := range r.Values {
		partialSubject := subject.Value

		fullSubject := snx_lib_db_nats.MakeSubject(partialSubject)

		fmt.Printf("Input: '%s' -> Full subject: '%s'\n", partialSubject, fullSubject)
	}
}
