package tool

import "fmt"

type Step int

const (
	StepInit Step = iota
	StepDoctor
	StepFmt
	StepGen
	StepLint
	StepBuild
	StepUnitTest
	StepRelease
	StepBootstrap
	StepDiff
	StepDeploy
	StepInspect
)

var stepNames = [...]string{
	StepInit:      "init",
	StepDoctor:    "doctor",
	StepFmt:       "fmt",
	StepGen:       "gen",
	StepLint:      "lint",
	StepBuild:     "build",
	StepUnitTest:  "unit-test",
	StepRelease:   "release",
	StepBootstrap: "bootstrap",
	StepDiff:      "diff",
	StepDeploy:    "deploy",
	StepInspect:   "inspect",
}

func (s Step) String() string {
	if int(s) < len(stepNames) {
		return stepNames[s]
	}
	return fmt.Sprintf("step(%d)", int(s))
}

var InitSteps = []Step{StepInit}

var DoctorSteps = []Step{StepDoctor}

var DevSteps = []Step{StepGen, StepFmt}

var CheckSteps = []Step{StepLint, StepUnitTest}

var ReleaseSteps = []Step{StepRelease}

var PreflightSteps = []Step{StepDoctor, StepGen, StepFmt, StepLint, StepBuild, StepUnitTest}

var InfraSteps = []Step{StepBootstrap, StepDiff, StepDeploy, StepInspect}

var AllSteps = []Step{
	StepInit, StepDoctor, StepGen, StepFmt, StepLint, StepBuild, StepUnitTest,
	StepRelease, StepBootstrap, StepDiff, StepDeploy, StepInspect,
}

func StepOrder() []Step {
	return PreflightSteps
}
