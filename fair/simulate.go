package fair

import (
	"errors"
	"fmt"
	"math"

	"golang.org/x/exp/rand"
	"gonum.org/v1/gonum/stat/distuv"
)

// seed is used for all distribution RNGs to make simulation deterministic.
// This means runs are reproducible but distributions are not independent.
// TODO: replace with a single shared RNG to get proper independence across
// distributions while keeping reproducibility.
const seed = 0

// Distribution can draw random samples and report its analytical mean.
type Distribution interface {
	Draw(n int) []float64
	Mean() float64
}

// Distribution implementations

// PoissonDist draws integer-valued event counts.
type PoissonDist struct{ dist distuv.Poisson }

func newPoisson(lambda float64) *PoissonDist {
	if lambda < 0 {
		lambda = 0
	}
	return &PoissonDist{dist: distuv.Poisson{Lambda: lambda, Src: rand.NewSource(seed)}}
}

func (d *PoissonDist) Draw(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = d.dist.Rand()
	}
	return out
}

func (d *PoissonDist) Mean() float64 { return d.dist.Mean() }

// PERTDist is a four-parameter Beta distribution scaled to [min, max] with
// the given mode. Uses the standard PERT kurtosis of 4.
type PERTDist struct {
	min, max, mode float64
	beta           distuv.Beta
}

func newPERT(min, mode, max float64) (*PERTDist, error) {
	if min >= max {
		return nil, errors.New("fair: PERT requires min < max")
	}
	mode = clamp(mode, min, max)

	const kurtosis = 4.0
	alpha := 1 + kurtosis*(mode-min)/(max-min)
	beta := 1 + kurtosis*(max-mode)/(max-min)
	return &PERTDist{
		min: min, max: max, mode: mode,
		beta: distuv.Beta{Alpha: alpha, Beta: beta, Src: rand.NewSource(seed)},
	}, nil
}

func (d *PERTDist) Draw(n int) []float64 {
	out := make([]float64, n)
	span := d.max - d.min
	for i := range out {
		out[i] = d.beta.Rand()*span + d.min
	}
	return out
}

func (d *PERTDist) Mean() float64 { return (d.min + 4*d.mode + d.max) / 6 }

// LogNormalDist fits a log-normal to a 90% confidence interval [low, high].
// low maps to the 5th percentile, high to the 95th.
type LogNormalDist struct{ dist distuv.LogNormal }

func newLogNormal(low, high float64) (*LogNormalDist, error) {
	if low >= high {
		return nil, errors.New("fair: LogNormal requires low < high")
	}
	norm := distuv.Normal{Mu: 0, Sigma: 1, Src: rand.NewSource(seed)}
	factor := -0.5 / norm.Quantile(0.05)
	mu := (math.Log(low) + math.Log(high)) / 2
	sigma := factor * (math.Log(high) - math.Log(low))
	return &LogNormalDist{dist: distuv.LogNormal{Mu: mu, Sigma: sigma, Src: rand.NewSource(seed)}}, nil
}

func (d *LogNormalDist) Draw(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = d.dist.Rand()
	}
	return out
}

func (d *LogNormalDist) Mean() float64 { return d.dist.Mean() }

// BetaDist draws from a Beta distribution on [0,1], parameterized from a
// three-point estimate using PERT-like shape. Used for probabilities.
type BetaDist struct{ dist distuv.Beta }

func newBeta(min, mode, max float64) (*BetaDist, error) {
	min = clamp(min, 0, 1)
	max = clamp(max, 0, 1)
	if min >= max {
		// Degenerate: point-mass at 0.5 via narrow symmetric Beta.
		return &BetaDist{dist: distuv.Beta{Alpha: 100, Beta: 100, Src: rand.NewSource(seed)}}, nil
	}
	mode = clamp(mode, min, max)

	const kurtosis = 4.0
	alpha := 1 + kurtosis*(mode-min)/(max-min)
	beta := 1 + kurtosis*(max-mode)/(max-min)
	return &BetaDist{dist: distuv.Beta{Alpha: alpha, Beta: beta, Src: rand.NewSource(seed)}}, nil
}

func (d *BetaDist) Draw(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = d.dist.Rand()
	}
	return out
}

func (d *BetaDist) Mean() float64 { return d.dist.Mean() }

// clamp restricts v to [lo, hi].
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}


// EstimateRole hints at what an estimate represents so the simulation engine
// can pick an appropriate default distribution when Dist == DistDefault.
type EstimateRole int

const (
	RoleFrequency   EstimateRole = iota // event counts (LEF, TEF)
	RoleProbability                     // 0–1 probabilities (Susceptibility)
	RoleMagnitude                       // monetary loss amounts
)

// NewDistribution creates a Distribution from an Estimate. If e.Dist is
// DistDefault, the role-based default is used:
//
//   - RoleFrequency   → Poisson (lambda = PERT mean)
//   - RoleProbability  → Beta
//   - RoleMagnitude    → PERT
//
// Invalid estimates are clamped to safe values rather than rejected, so
// that partially-filled forms still produce usable (if imprecise) results.
func NewDistribution(e Estimate, role EstimateRole) (Distribution, error) {
	dist := e.Dist
	if dist == DistDefault {
		switch role {
		case RoleFrequency:
			dist = DistPoisson
		case RoleProbability:
			dist = DistBeta
		case RoleMagnitude:
			dist = DistPERT
		}
	}

	switch dist {
	case DistPoisson:
		lambda := e.PERTMean()
		if lambda <= 0 {
			lambda = 0.001 // near-zero avoids zero-lambda panic
		}
		return newPoisson(lambda), nil

	case DistPERT:
		min, mode, max := e.Min, e.ML, e.Max
		if min < 0 {
			min = 0
		}
		if mode <= min {
			mode = min + 1
		}
		if max <= mode {
			max = mode + 1
		}
		return newPERT(min, mode, max)

	case DistLogNormal:
		low, high := e.Min, e.Max
		if low <= 0 {
			low = 1
		}
		if high <= low {
			high = low * 10
		}
		return newLogNormal(low, high)

	case DistBeta:
		return newBeta(e.Min, e.ML, e.Max)

	default:
		return nil, fmt.Errorf("fair: unsupported distribution type: %v", dist)
	}
}

// Simulation results

// SimulationResult holds per-year aggregate losses from a Monte Carlo run.
type SimulationResult struct {
	YearlyLosses []float64
}

// Mean returns the average annual loss across simulated years.
func (sr SimulationResult) Mean() float64 {
	if len(sr.YearlyLosses) == 0 {
		return 0
	}
	var sum float64
	for _, v := range sr.YearlyLosses {
		sum += v
	}
	return sum / float64(len(sr.YearlyLosses))
}

// Scenario

// Scenario is a named FAIR loss event used in multi-scenario analyses.
type Scenario struct {
	Identifier string
	Name       string
	LossEvent
}

// Label returns "Identifier: Name" or just Identifier if Name is empty.
func (s Scenario) Label() string {
	if s.Name == "" {
		return s.Identifier
	}
	return s.Identifier + ": " + s.Name
}

// AnnualLoss is a scenario's expected annualized loss.
type AnnualLoss struct {
	Identifier string
	Name       string
	Loss       float64
}

// PrioritizedLosses computes the expected annualized loss for each scenario
// and returns them sorted descending by loss amount.
// Uses insertion sort — adequate for the expected number of scenarios (<100).
func PrioritizedLosses(scenarios []Scenario) ([]AnnualLoss, error) {
	result := make([]AnnualLoss, 0, len(scenarios))
	for _, s := range scenarios {
		al, err := AnnualizedLoss(s.LossEvent)
		if err != nil {
			return nil, err
		}
		result = append(result, AnnualLoss{
			Identifier: s.Identifier,
			Name:       s.Name,
			Loss:       al,
		})
	}
	// Sort descending by loss (NaN sorts last).
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && less(result[j], result[j-1]); j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}
	return result, nil
}

// less returns true if a should sort before b (higher loss first, NaN last).
func less(a, b AnnualLoss) bool {
	if math.IsNaN(b.Loss) && !math.IsNaN(a.Loss) {
		return true
	}
	return a.Loss > b.Loss
}


// SimulateMulti runs a Monte Carlo simulation for each scenario and returns
// per-scenario results plus an aggregate (yearly sum across all scenarios).
func SimulateMulti(scenarios []Scenario, years int) (perScenario []SimulationResult, aggregate SimulationResult, err error) {
	perScenario = make([]SimulationResult, len(scenarios))
	aggregate = SimulationResult{YearlyLosses: make([]float64, years)}

	for i, s := range scenarios {
		sr, err := SimulateLossEvent(s.LossEvent, years)
		if err != nil {
			return nil, SimulationResult{}, err
		}
		perScenario[i] = sr
		for j := range sr.YearlyLosses {
			aggregate.YearlyLosses[j] += sr.YearlyLosses[j]
		}
	}
	return perScenario, aggregate, nil
}


// SimulateLossEvent runs a Monte Carlo simulation of a single FAIR LossEvent
// for n years. Each simulated year:
//
//  1. Draw event count from the frequency distribution.
//  2. (Decomposed mode only) For each threat event, draw a susceptibility
//     probability and run a Bernoulli trial to decide if it becomes a loss event.
//  3. For each loss event, draw a magnitude from every non-zero loss form
//     and sum them.
//
// Known limitation: the Bernoulli trials in step 2 use a per-event RNG
// seeded from (year, event index). This makes the simulation deterministic
// but couples the susceptibility filter to the event ordering. A single
// shared RNG would be preferable; see the TODO on the seed constant.
func SimulateLossEvent(le LossEvent, n int) (SimulationResult, error) {
	// Build frequency distribution
	var freqDist Distribution
	var suscDist Distribution

	if le.LEFMode == LEFDirect {
		d, err := NewDistribution(le.DirectLEF, RoleFrequency)
		if err != nil {
			return SimulationResult{}, err
		}
		freqDist = d
	} else {
		d, err := NewDistribution(le.TEF, RoleFrequency)
		if err != nil {
			return SimulationResult{}, err
		}
		freqDist = d

		sd, err := NewDistribution(le.Susc, RoleProbability)
		if err != nil {
			return SimulationResult{}, err
		}
		suscDist = sd
	}

	// Collect non-zero magnitude distributions from PL and SL
	var magDists []Distribution
	for _, e := range append(le.PL.Estimates(), le.SL.Estimates()...) {
		if e.IsZero() {
			continue
		}
		d, err := NewDistribution(e, RoleMagnitude)
		if err != nil {
			return SimulationResult{}, err
		}
		magDists = append(magDists, d)
	}

	// Run simulation
	yearly := make([]float64, n)
	freqDraws := freqDist.Draw(n)

	// Pre-draw susceptibility pool for decomposed mode.
	var suscDraws []float64
	if suscDist != nil {
		var totalEvents int
		for _, f := range freqDraws {
			totalEvents += int(math.Round(f))
		}
		if totalEvents > 0 {
			suscDraws = suscDist.Draw(totalEvents)
		}
	}

	suscIdx := 0
	for yr := 0; yr < n; yr++ {
		numThreat := int(math.Round(freqDraws[yr]))
		if numThreat <= 0 {
			continue
		}

		// Determine actual loss events (after susceptibility filter).
		numLoss := numThreat
		if suscDist != nil {
			actual := 0
			for i := 0; i < numThreat; i++ {
				if suscIdx < len(suscDraws) {
					p := suscDraws[suscIdx]
					suscIdx++
					// Bernoulli trial: uniform draw vs. susceptibility.
					rng := rand.New(rand.NewSource(uint64(yr*1000 + i)))
					if rng.Float64() < p {
						actual++
					}
				}
			}
			numLoss = actual
		}

		if numLoss <= 0 || len(magDists) == 0 {
			continue
		}

		// Draw magnitude for each loss event from each loss form.
		for _, md := range magDists {
			for _, v := range md.Draw(numLoss) {
				yearly[yr] += v
			}
		}
	}

	return SimulationResult{YearlyLosses: yearly}, nil
}


// AnnualizedLoss computes E[annual loss] = E[freq] × E[susc] × Σ E[mag_i]
// using the analytical means of each distribution. No simulation is run.
func AnnualizedLoss(le LossEvent) (float64, error) {
	var freqMean float64
	var suscMean float64 = 1.0

	if le.LEFMode == LEFDirect {
		d, err := NewDistribution(le.DirectLEF, RoleFrequency)
		if err != nil {
			return 0, err
		}
		freqMean = d.Mean()
	} else {
		d, err := NewDistribution(le.TEF, RoleFrequency)
		if err != nil {
			return 0, err
		}
		freqMean = d.Mean()

		sd, err := NewDistribution(le.Susc, RoleProbability)
		if err != nil {
			return 0, err
		}
		suscMean = sd.Mean()
	}

	var totalMagMean float64
	for _, e := range append(le.PL.Estimates(), le.SL.Estimates()...) {
		if e.IsZero() {
			continue
		}
		d, err := NewDistribution(e, RoleMagnitude)
		if err != nil {
			return 0, err
		}
		totalMagMean += d.Mean()
	}

	return freqMean * suscMean * totalMagMean, nil
}

// CSV bridge

// NewSimpleScenario creates a Scenario from simple CSV-style parameters:
// frequency (Poisson lambda), lowLoss/highLoss (LogNormal 90% CI).
//
// This is a convenience for the legacy CSV format and quantriskcli.
// TODO: consider removing once we have proper CSV import with full FAIR fields.
func NewSimpleScenario(identifier, name string, frequency, lowLoss, highLoss float64) Scenario {
	return Scenario{
		Identifier: identifier,
		Name:       name,
		LossEvent: LossEvent{
			LEFMode: LEFDirect,
			DirectLEF: Estimate{
				Min:  frequency,
				ML:   frequency,
				Max:  frequency,
				Dist: DistPoisson,
			},
			PL: LossForm{
				ProdL: Estimate{
					Min:  lowLoss,
					ML:   (lowLoss + highLoss) / 2,
					Max:  highLoss,
					Dist: DistLogNormal,
				},
			},
		},
	}
}
