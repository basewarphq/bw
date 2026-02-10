package tool

import "fmt"

type Step int

const (
	StepFmt Step = iota
	StepGen
	StepLint
	StepCompiles
	StepUnitTest
)

var stepNames = [...]string{
	StepFmt:      "fmt",
	StepGen:      "gen",
	StepLint:     "lint",
	StepCompiles: "compiles",
	StepUnitTest: "unit-test",
}

func (s Step) String() string {
	if int(s) < len(stepNames) {
		return stepNames[s]
	}
	return fmt.Sprintf("step(%d)", int(s))
}

var DevSteps = []Step{StepGen, StepFmt}

var CheckSteps = []Step{StepLint, StepCompiles, StepUnitTest}

var AllDevCheckSteps = []Step{StepGen, StepFmt, StepLint, StepCompiles, StepUnitTest}

func StepOrder() []Step {
	return AllDevCheckSteps
}
