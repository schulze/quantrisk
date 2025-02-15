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

const (
	frequency = 0.1
	low       = 1
	high      = 10
)

type FixedFreq int64

func (f FixedFreq) Draw(n int) []int64 {
	a := make([]int64, n)
	for i := range a {
		a[i] = int64(f)
	}
	return a
}

func (f FixedFreq) Mean() float64 {
	return float64(f)
}

type FixedMagn float64

func (f FixedMagn) Draw(n int) []float64 {
	a := make([]float64, n)
	for i := range a {
		a[i] = float64(f)
	}
	return a
}

func (f FixedMagn) Mean() float64 {
	return float64(f)
}

// A Loss to use for various tests.
var loss SimpleLoss = SimpleLoss{Loss{FixedFreq(1), FixedMagn(0.5)}, "L1", "loss_name", frequency, low, high}

func TestVariables(t *testing.T) {
	// TODO
}

func TestAnnualized(t *testing.T) {
	// TODO
}

func TestLargeFrequency(t *testing.T) {
}

// TODO More tests.
