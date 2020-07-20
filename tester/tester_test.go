package tester

import "testing"

func TestTester(t *testing.T) {
	T(t).Run(EmptyTesterTest)
}

func EmptyTesterTest(tester Tester) {
	tester.Stop()
}
