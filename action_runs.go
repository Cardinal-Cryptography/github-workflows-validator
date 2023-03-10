package main

type ActionRuns struct {
	Using string        `yaml:"using"`
	Steps []*ActionStep `yaml:"steps"`
}
