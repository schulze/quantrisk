/*
Copyright 2019-2020 Netflix, Inc.
Copyright 2022 Frithjof Schulze

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package risk

import (
	"testing"
)

func TestLogNormalDistribution(t *testing.T) {
	logn, _ := NewLogNormal(1, 10)
	if cdf := logn.CDF(1); cdf <= 0.049 || cdf >= 0.051 {
		t.Fatal("CDF at 1 should be around 0.5.")
	}
	if cdf := logn.CDF(10); cdf <= 0.949 || cdf >= 0.951 {
		t.Fatal("CDF at 10 should be around 0.95.")
	}
}

func TestLogNormalHardParameters(t *testing.T) {
	hard, _ := NewLogNormal(635000, 19000000)
	expectedMean := 5922706.83351131
	if found := hard.Mean(); found-expectedMean > 1e-7 {
		t.Fatalf("Mean value mismatch: Expected %v, Found %v", expectedMean, found)
	}
}

func TestPERT(t *testing.T) {
	min := 0.
	max := 10.
	mostLikely := 5.
	kurtosis := 1.
	pertFreq := NewPERT(min, max, mostLikely, kurtosis)

	values := pertFreq.Draw(10000)
	sum := 0.
	for _, v := range values {
		sum += v
	}
	total := sum / 10000
	if total < 4.7 || total > 5.3 {
		t.Fatalf("Expected total aound 5, got %v", total)
	}

	pertFreq = NewPERT(0, 0.01, 0.005, 1)
	values = pertFreq.Draw(100000)
	sum = 0.
	for _, v := range values {
		sum += v
	}
	total = sum / 100000
	if total < 0.0040 || total > 0.006 {
		t.Fatalf("Expected sum aound 0.005, got %v", total)
	}
}
