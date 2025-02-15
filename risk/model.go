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
	"errors"
	"math"

	"golang.org/x/exp/rand"

	"gonum.org/v1/gonum/stat/distuv"
)

const seed = 0 // TODO

type FrequencyDist interface {
	Draw(int) []int64
	Mean() float64
}

type MagnitudeDist interface {
	Draw(int) []float64
	Mean() float64
}

// Poisson frequency distribution
type Poisson struct {
	distuv.Poisson
}

// NewPoisson returns a new Poisson model with a mean rate per interval of frequency.
func NewPoisson(frequency float64) (*Poisson, error) {
	if frequency < 0 {
		return nil, errors.New("Frequency must be non-negative.")
	}
	return &Poisson{distuv.Poisson{Lambda: frequency, Src: rand.NewSource(seed)}}, nil
}

func (dist *Poisson) Draw(n int) []int64 {
	d := make([]int64, n)
	for i := range d {
		d[i] = int64(dist.Rand())
	}
	return d
}

func (dist *Poisson) Mean() float64 {
	return dist.Poisson.Mean() // Lambda
}

// The code for the PERT distribution below was adapted from the implementation
// in https://github.com/goinvest/distuvx by the distuvx developers.

// PERT represents a PERT distribution, which is a four parameter Beta
// distribution described by the parameters min, max, and mode, as well as the
// requirement that the mean = (max + 4 * mode + min) / 6.
// (https://en.wikipedia.org/wiki/PERT_distribution)
type PERT struct {
	min  float64
	max  float64
	mode float64
	distuv.Beta
}

// NewPERT constructs a new modified PERT distribution using the given min, max, mode and kurtosis.
// Constraints are min < max and min ≤ mode ≤ max.
func NewPERT(min, max, mode, kurtosis float64) PERT {
	checkPERTParameters(min, max, mode)
	alpha := 1 + kurtosis*(mode-min)/(max-min)
	beta := 1 + kurtosis*(max-mode)/(max-min)
	return PERT{
		min:  min,
		max:  max,
		mode: mode,
		Beta: distuv.Beta{
			Alpha: alpha,
			Beta:  beta,
			Src:   rand.NewSource(seed),
		},
	}
}

func checkPERTParameters(min, max, mode float64) {
	// TODO: Remove panics, use errors.
	if min >= max {
		panic("pert: constraint of min < max violated")
	}
	if min > mode {
		panic("pert: constraint of min <= mode violated")
	}
	if mode > max {
		panic("pert: constraint of mode <= max violated")
	}
}

// Mean returns the mean of the PERT probability distribution.
func (p PERT) Mean() float64 {
	return (p.min + 4*p.mode + p.max) / 6
}

// Rand implements the Rander interface for the PERT distribution.
func (p PERT) Rand() float64 {
	return p.Beta.Rand()*(p.max-p.min) + p.min
}

func (dist PERT) Draw(n int) []float64 {
	d := make([]float64, n)
	for i := range d {
		d[i] = dist.Rand()
	}
	return d
}

// Lognormal magnitude distribution based on a 90% confidence interval.
type LogNormal struct {
	Low, High float64
	distuv.LogNormal
}

// NewLognormal returns a LogNormal model with a low low-loss estimate and a high high-loss estimate.
// The range from low to high represents the 90% confidence interval, that the loss will fall in that range.
func NewLogNormal(low, high float64) (*LogNormal, error) {
	if low >= high {
		return nil, errors.New("Low must be less than high.")
	}
	// Low and high are fitted to a lognormal distribution, so that they fall at the 5% and 95%
	// cumulative probability points.
	norm := distuv.Normal{Mu: 0, Sigma: 1, Src: rand.NewSource(seed)}
	factor := -0.5 / norm.Quantile(0.05)
	mu := (math.Log(low) + math.Log(high)) / 2         // average of the logarithm of low/high
	shape := factor * (math.Log(high) - math.Log(low)) // standard deviation
	distrib := distuv.LogNormal{Mu: mu, Sigma: shape, Src: rand.NewSource(seed)}
	return &LogNormal{Low: low, High: high, LogNormal: distrib}, nil
}

func (dist *LogNormal) Draw(n int) []float64 {
	d := make([]float64, n)
	for i := range d {
		d[i] = dist.Rand()
	}
	return d
}

func (dist LogNormal) Mean() float64 {
	return dist.LogNormal.Mean()
}
