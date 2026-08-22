// Package cam implements the FAIR Controls Analytics Model (FAIR-CAM) v1.0.
//
// FAIR-CAM describes how risk management controls operate as a system through
// three functional domains:
//
//   - Loss Event Controls (LEC): directly affect the frequency or magnitude of loss events
//   - Variance Management Controls (VMC): affect the operational performance/reliability of controls
//   - Decision Support Controls (DSC): affect risk management decision-making
//
// Each domain is decomposed into high-level functions and sub-functions that
// have defined relationships (Boolean AND/OR) with each other.
//
// See https://www.fairinstitute.org/ for the full FAIR-CAM standard.
package cam

// Domain represents one of the three FAIR-CAM functional domains.
type Domain string

const (
	// DomainLEC is the Loss Event Control domain. Controls in this domain
	// directly affect the frequency or magnitude of loss events.
	DomainLEC Domain = "LEC"

	// DomainVMC is the Variance Management Control domain. Controls in this
	// domain affect the operational performance and reliability of other controls.
	DomainVMC Domain = "VMC"

	// DomainDSC is the Decision Support Control domain. Controls in this
	// domain affect risk management decision-making.
	DomainDSC Domain = "DSC"
)

// Function identifies a control function within a domain.
type Function string

// Loss Event Control (LEC) Functions
//
// LEC decomposes into three high-level functions:
//   Prevention (OR: Avoidance, Deterrence, Resistance)
//   Detection  (AND: Visibility, Monitoring, Recognition)
//   Response   (weak AND: Event Termination, Resilience, Loss Reduction)
//
// Detection AND Response must both exist to mitigate loss event effects.

const (
	// LEC / Prevention (OR relationship between sub-functions)

	// LECAvoidance reduces the frequency of contact between threat agents
	// and the assets they could adversely affect.
	// Unit: % reduction in contact frequency with threat agents.
	LECAvoidance Function = "LEC.Prevention.Avoidance"

	// LECDeterrence reduces the probability of potentially harmful actions
	// after a threat agent has come into contact with an asset.
	// Unit: % reduction in probability that threat actors choose to act harmfully.
	LECDeterrence Function = "LEC.Prevention.Deterrence"

	// LECResistance reduces the likelihood that a threat agent's actions
	// will result in a loss event.
	// Unit: % probability of resisting potentially harmful actions.
	LECResistance Function = "LEC.Prevention.Resistance"

	// LEC / Detection (AND relationship between sub-functions)

	// LECVisibility provides an indication of activity that may be anomalous
	// or illicit (e.g., logs, cameras, sensors).
	// Unit: % probability that the control provides access to necessary information.
	LECVisibility Function = "LEC.Detection.Visibility"

	// LECMonitoring reviews data provided by Visibility controls.
	// Unit: elapsed time between reviews.
	LECMonitoring Function = "LEC.Detection.Monitoring"

	// LECRecognition enables differentiation of normal from abnormal
	// activity/conditions that may indicate a loss event.
	// Unit: % probability of successfully differentiating loss events from normal.
	LECRecognition Function = "LEC.Detection.Recognition"

	// LEC / Response (weak AND relationship between sub-functions)

	// LECEventTermination enables termination of ongoing threat agent
	// activities that could continue to be harmful.
	// Unit: time from recognition to control over the event.
	LECEventTermination Function = "LEC.Response.EventTermination"

	// LECResilience maintains or restores normal operations after a
	// loss event (e.g., backup/recovery, failover).
	// Unit: time operating in a degraded mode.
	LECResilience Function = "LEC.Response.Resilience"

	// LECLossReduction reduces the amount of realized losses from an
	// event (e.g., insurance, legal actions).
	// Unit: reduction of lost economic value.
	LECLossReduction Function = "LEC.Response.LossReduction"
)

// Variance Management Control (VMC) Functions
//
// VMC decomposes into three high-level functions:
//   Prevention     (OR: Reduce Change Frequency, Reduce Variance Probability)
//   Identification (AND with Correction: Threat Intelligence, Control Monitoring)
//   Correction     (AND with Identification: Treatment Selection, Implementation)

const (
	// VMC / Prevention (OR relationship between sub-functions)

	// VMCReduceChangeFreq reduces the frequency of changes that could
	// introduce variance in control performance.
	// Unit: % reduction in frequency of changes that could introduce variance.
	VMCReduceChangeFreq Function = "VMC.Prevention.ReduceChangeFrequency"

	// VMCReduceVarianceProb reduces the probability that changes will
	// result in control degradation or failure.
	// Unit: % reduction in variance.
	VMCReduceVarianceProb Function = "VMC.Prevention.ReduceVarianceProbability"

	// VMC / Identification (AND with Correction)

	// VMCThreatIntel identifies changes in the threat landscape that
	// diminish the efficacy of controls.
	// Unit: elapsed time between threat landscape changes and awareness.
	VMCThreatIntel Function = "VMC.Identification.ThreatIntelligence"

	// VMCControlMonitoring identifies variance in control conditions.
	// Unit: elapsed time between control condition changes and recognition.
	VMCControlMonitoring Function = "VMC.Identification.ControlMonitoring"

	// VMC / Correction (AND with Identification)

	// VMCTreatmentSelection selects and prioritizes control variance
	// corrections.
	// Unit: elapsed time from identification until corrective actions begin.
	VMCTreatmentSelection Function = "VMC.Correction.TreatmentSelection"

	// VMCImplementation corrects variant conditions.
	// Unit: elapsed time from initiation of corrective actions until completion.
	VMCImplementation Function = "VMC.Correction.Implementation"
)

// Decision Support Control (DSC) Functions
//
// DSC decomposes into three high-level functions:
//   Prevention    (AND: Defined Expectations, Communication, Situational Awareness,
//                       Ensure Capability, Incentives)
//   Identification (identify misaligned decisions)
//   Correction     (correct misaligned decisions; fulfilled by other functions)

const (
	// DSC / Prevention (AND relationship between sub-functions)

	// DSCDefinedExpectations clearly defines expectations and objectives
	// (e.g., risk appetite, policies, configuration standards).
	// Unit: probability that clear expectations have been defined.
	DSCDefinedExpectations Function = "DSC.Prevention.DefinedExpectations"

	// DSCCommunication communicates expectations to responsible personnel
	// (e.g., education & awareness training, policy updates).
	// Unit: probability that expectations have been communicated.
	DSCCommunication Function = "DSC.Prevention.Communication"

	// DSCSituationalAwareness provides decision-makers with understanding
	// of the current risk landscape and implications of their decisions.
	// Composed of sub-functions: Data (Asset, Threat, Controls), Analysis, Reporting.
	// Unit: composite of sub-function measurements.
	DSCSituationalAwareness Function = "DSC.Prevention.SituationalAwareness"

	// DSCEnsureCapability ensures decision-makers have the skills,
	// authority, and resources for aligned decisions.
	// Unit: probability that responsible persons have necessary skills/resources.
	DSCEnsureCapability Function = "DSC.Prevention.EnsureCapability"

	// DSCIncentives motivates personnel to make decisions aligned with
	// the organization's expectations and objectives.
	// Unit: probability that appropriate incentives are in place.
	DSCIncentives Function = "DSC.Prevention.Incentives"

	// DSC / Identification

	// DSCIdentification enables identification of decisions not aligned
	// with organizational expectations and objectives.
	// Unit: elapsed time from misaligned decision to identification.
	DSCIdentification Function = "DSC.Identification"

	// DSC / Correction

	// DSCCorrection corrects causes and outcomes of misaligned decisions.
	// Fulfilled by controls within other functions (LEC Response, VMC Correction, etc.).
	// Unit: elapsed time from recognition to correction.
	DSCCorrection Function = "DSC.Correction"
)

// Domain returns the functional domain for a given function.
func (f Function) Domain() Domain {
	if len(f) < 3 {
		return ""
	}
	switch f[:3] {
	case "LEC":
		return DomainLEC
	case "VMC":
		return DomainVMC
	case "DSC":
		return DomainDSC
	}
	return ""
}
