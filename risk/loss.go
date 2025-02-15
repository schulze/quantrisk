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
	"io"
	"math"
	"sort"

	"gonum.org/v1/gonum/stat"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
)

type Loss struct {
	Frequency FrequencyDist
	Magnitude MagnitudeDist
}

func (l Loss) AnnualizedLoss() float64 {
	return l.Frequency.Mean() * l.Magnitude.Mean()
}

// SimulateLossesOneYear returns a slice of zero or more loss magnitudes for a single simulated year.
func (l Loss) SimulateLossedOneYear() []float64 {
	numEvents := int(l.Frequency.Draw(1)[0])
	losses := l.Magnitude.Draw(numEvents)
	return losses
}

func sumInt(s []int64) int64 {
	t := int64(0)
	for _, v := range s {
		t += v
	}
	return t
}

func sumFloat(s []float64) float64 {
	t := float64(0)
	for _, v := range s {
		t += v
	}
	return t
}

// SimulateYears returns a slice of length n, each entry is the sum of losses for that simulated year.
func (l Loss) SimulateYears(n int) []float64 {
	numLosses := l.Frequency.Draw(n)
	lossValues := l.Magnitude.Draw(int(sumInt(numLosses)))
	losses := make([]float64, n)
	seenLosses := 0
	for i := range losses {
		newLosses := int(numLosses[i]) // number of new losses this years
		losses[i] = sumFloat(lossValues[seenLosses : seenLosses+newLosses])
		seenLosses += newLosses
	}
	return losses
}

type LossSummary struct {
	minimum             float64
	tenthPercentile     float64
	mode                float64
	median              float64
	ninetiethPercentile float64
	maximum             float64
}

func SummarizeLoss(losses []float64) LossSummary {
	return LossSummary{}
}

type SimpleLoss struct {
	Loss
	label     string
	name      string
	frequency float64
	lowLoss   float64
	highLoss  float64
}

func NewSimpleLoss(label, name string, frequency, lowLoss, highLoss float64) SimpleLoss {
	poissonModel, _ := NewPoisson(frequency)
	logNormalModel, _ := NewLogNormal(lowLoss, highLoss)
	return SimpleLoss{Loss{poissonModel, logNormalModel}, label, name, frequency, lowLoss, highLoss}
}

// MultiLoss is a container for a slice of loss objects and methods for generating summaries of them.
// TODO This should probably just be a set of functions.
// TODO SimpleLoss should be an interface. We don't just want simple losses...
type MultiLoss []SimpleLoss

type AnnualLoss struct {
	Identifier string
	Name       string
	Loss       float64
}

type annualLosses []AnnualLoss

func (al annualLosses) Len() int {
	return len(al)
}
func (al annualLosses) Less(i, j int) bool {
	return al[i].Loss < al[j].Loss || (math.IsNaN(al[i].Loss) && !math.IsNaN(al[j].Loss))
}

func (al annualLosses) Swap(i, j int) {
	t := al[i]
	al[i] = al[j]
	al[j] = t
}

func (m MultiLoss) PrioritizedLosses() []AnnualLoss {
	losses := make([]AnnualLoss, 0)
	for _, v := range m {
		losses = append(losses, AnnualLoss{v.label, v.name, v.AnnualizedLoss()})
	}
	sort.Sort(annualLosses(losses))
	for i, j := 0, len(losses)-1; i < j; i, j = i+1, j-1 {
		losses[i], losses[j] = losses[j], losses[i]
	}
	return losses
}

// SimulateYears simulates n years across all the losses in the list and returns a slice
// [ year1_loss, year2_loss, ...] of losses per year.
func (m MultiLoss) SimulateYears(n int) []float64 {
	losses := make([][]float64, len(m)) // losses[i][j] is loss of i-th event in j-th year.
	for i, v := range m {
		losses[i] = v.SimulateYears(n)
	}
	yearlyLosses := make([]float64, n)
	for j := range yearlyLosses {
		yearlySum := float64(0)
		for i := range losses {
			yearlySum += losses[i][j]
		}
		yearlyLosses[j] = yearlySum
	}
	return yearlyLosses
}

// LossExceedance generates the Loss Exceedance Curve (LEC) for n years unsing SimulateYears.
func (m MultiLoss) LossExceedanceCurve(n int, title string, out io.Writer) {
	// TODO: This is translated from the Python code, but probably the wrong way to calculate the LEC.
	// We should use the CDF instead of percentiles, so we can just plot a function.
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = "Loss"
	p.Y.Label.Text = "Probability"
	p.X.Scale = plot.LogScale{}
	p.X.Tick.Marker = plot.LogTicks{}

	vals := make(plotter.XYs, 99)
	losses := m.SimulateYears(n)
	sort.Float64s(losses)

	for i := range vals {
		x := float64(i + 1)
		vals[i].X = stat.Quantile(x/100, stat.LinInterp, losses, nil) + 0.1
		vals[i].Y = (100 - x) / 100
	}
	err := plotutil.AddLinePoints(p, "First", vals)
	if err != nil {
		panic(err)
	}

	cdf := func(x float64) float64 { return 1 - stat.CDF(x, stat.Empirical, losses, nil) }
	f := plotter.NewFunction(cdf)
	f.XMin = 1000000
	f.XMax = 10000000000

	p.Add(f, plotter.NewGrid())
	p.Legend.Add("CDF", f)

	p.X.Min = f.XMin
	p.X.Max = f.XMax
	p.Y.Min = 0.0
	p.Y.Max = 0.3
	wt, err := p.WriterTo(600, 600, "svg")
	if err != nil {
		panic(err)
	}
	_, err = wt.WriteTo(out)
	if err != nil {
		panic(err)
	}
}
