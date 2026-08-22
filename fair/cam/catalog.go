package cam

// Relationship describes the Boolean relationship between sibling functions.
type Relationship string

const (
	// OR means any one functioning control achieves the objective;
	// multiple controls have cumulative effect.
	OR Relationship = "OR"

	// AND means all sub-functions must be present for the objective
	// to be achieved.
	AND Relationship = "AND"

	// WeakAND means deficiency in one sub-function diminishes overall
	// efficacy but doesn't necessarily inhibit it entirely.
	WeakAND Relationship = "WeakAND"
)

// FunctionInfo describes a control function's metadata.
type FunctionInfo struct {
	// Function is the canonical identifier.
	Function Function

	// Name is the human-readable name.
	Name string

	// Description summarizes what the function does.
	Description string

	// Domain is the functional domain.
	Domain Domain

	// Parent is the high-level function this belongs to (e.g., "LEC.Prevention").
	// Empty for top-level groupings.
	Parent string

	// SiblingRelationship describes the Boolean relationship among siblings
	// under the same parent.
	SiblingRelationship Relationship

	// Unit describes how the function's effectiveness is measured.
	Unit string
}

// Catalog is the complete set of FAIR-CAM control functions.
var Catalog = []FunctionInfo{
	// LEC / Prevention (OR)
	{
		Function:            LECAvoidance,
		Name:                "Avoidance",
		Description:         "Reduce the frequency of contact between threat agents and the assets they could adversely affect.",
		Domain:              DomainLEC,
		Parent:              "LEC.Prevention",
		SiblingRelationship: OR,
		Unit:                "% reduction in contact frequency with threat agents",
	},
	{
		Function:            LECDeterrence,
		Name:                "Deterrence",
		Description:         "Reduce the probability of potentially harmful actions after a threat agent has come into contact with an asset.",
		Domain:              DomainLEC,
		Parent:              "LEC.Prevention",
		SiblingRelationship: OR,
		Unit:                "% reduction in probability that threat actors choose to act harmfully",
	},
	{
		Function:            LECResistance,
		Name:                "Resistance",
		Description:         "Reduce the likelihood that a threat agent's actions will result in a loss event.",
		Domain:              DomainLEC,
		Parent:              "LEC.Prevention",
		SiblingRelationship: OR,
		Unit:                "% probability of resisting potentially harmful actions",
	},

	// LEC / Detection (AND)
	{
		Function:            LECVisibility,
		Name:                "Visibility",
		Description:         "Provide an indication of activity that may be anomalous or illicit.",
		Domain:              DomainLEC,
		Parent:              "LEC.Detection",
		SiblingRelationship: AND,
		Unit:                "% probability that the control provides access to necessary information",
	},
	{
		Function:            LECMonitoring,
		Name:                "Monitoring",
		Description:         "Review data provided by Visibility controls.",
		Domain:              DomainLEC,
		Parent:              "LEC.Detection",
		SiblingRelationship: AND,
		Unit:                "elapsed time between reviews",
	},
	{
		Function:            LECRecognition,
		Name:                "Recognition",
		Description:         "Enable differentiation of normal from abnormal activity/conditions that may indicate a loss event.",
		Domain:              DomainLEC,
		Parent:              "LEC.Detection",
		SiblingRelationship: AND,
		Unit:                "% probability of successful differentiation",
	},

	// LEC / Response (WeakAND)
	{
		Function:            LECEventTermination,
		Name:                "Event Termination",
		Description:         "Enable termination of threat agent activities that could continue to be harmful.",
		Domain:              DomainLEC,
		Parent:              "LEC.Response",
		SiblingRelationship: WeakAND,
		Unit:                "time from recognition to control over the event",
	},
	{
		Function:            LECResilience,
		Name:                "Resilience",
		Description:         "Maintain or restore normal operations.",
		Domain:              DomainLEC,
		Parent:              "LEC.Response",
		SiblingRelationship: WeakAND,
		Unit:                "time operating in a degraded mode",
	},
	{
		Function:            LECLossReduction,
		Name:                "Loss Reduction",
		Description:         "Reduce the amount of realized losses from an event.",
		Domain:              DomainLEC,
		Parent:              "LEC.Response",
		SiblingRelationship: WeakAND,
		Unit:                "reduction of lost economic value",
	},

	// VMC / Prevention (OR)
	{
		Function:            VMCReduceChangeFreq,
		Name:                "Reduce Change Frequency",
		Description:         "Reduce the frequency of changes that could introduce variance.",
		Domain:              DomainVMC,
		Parent:              "VMC.Prevention",
		SiblingRelationship: OR,
		Unit:                "% reduction in frequency of changes",
	},
	{
		Function:            VMCReduceVarianceProb,
		Name:                "Reduce Variance Probability",
		Description:         "Reduce the probability that changes will result in control degradation or failure.",
		Domain:              DomainVMC,
		Parent:              "VMC.Prevention",
		SiblingRelationship: OR,
		Unit:                "% reduction in variance",
	},

	// VMC / Identification (AND with Correction)
	{
		Function:            VMCThreatIntel,
		Name:                "Threat Intelligence",
		Description:         "Identify changes in the threat landscape that diminish the efficacy of controls.",
		Domain:              DomainVMC,
		Parent:              "VMC.Identification",
		SiblingRelationship: AND,
		Unit:                "elapsed time between threat landscape changes and awareness",
	},
	{
		Function:            VMCControlMonitoring,
		Name:                "Control Monitoring",
		Description:         "Identify variance in control conditions.",
		Domain:              DomainVMC,
		Parent:              "VMC.Identification",
		SiblingRelationship: AND,
		Unit:                "elapsed time between control changes and recognition",
	},

	// VMC / Correction (AND with Identification)
	{
		Function:            VMCTreatmentSelection,
		Name:                "Treatment Selection & Prioritization",
		Description:         "Select and prioritize control variance corrections.",
		Domain:              DomainVMC,
		Parent:              "VMC.Correction",
		SiblingRelationship: AND,
		Unit:                "elapsed time from identification until corrective actions begin",
	},
	{
		Function:            VMCImplementation,
		Name:                "Implementation",
		Description:         "Correct variant conditions.",
		Domain:              DomainVMC,
		Parent:              "VMC.Correction",
		SiblingRelationship: AND,
		Unit:                "elapsed time from initiation until completion",
	},

	// DSC / Prevention (AND)
	{
		Function:            DSCDefinedExpectations,
		Name:                "Defined Expectations",
		Description:         "Clearly define expectations and/or objectives.",
		Domain:              DomainDSC,
		Parent:              "DSC.Prevention",
		SiblingRelationship: AND,
		Unit:                "probability that clear expectations have been defined",
	},
	{
		Function:            DSCCommunication,
		Name:                "Communication of Expectations",
		Description:         "Communicate expectations to responsible personnel.",
		Domain:              DomainDSC,
		Parent:              "DSC.Prevention",
		SiblingRelationship: AND,
		Unit:                "probability that expectations have been communicated",
	},
	{
		Function:            DSCSituationalAwareness,
		Name:                "Situational Awareness",
		Description:         "Provide decision-makers with understanding of the risk landscape and implications of decisions. Sub-functions: Data (Asset, Threat, Controls), Analysis, Reporting.",
		Domain:              DomainDSC,
		Parent:              "DSC.Prevention",
		SiblingRelationship: AND,
		Unit:                "composite of data/analysis/reporting quality",
	},
	{
		Function:            DSCEnsureCapability,
		Name:                "Ensure Capability",
		Description:         "Ensure decision-makers have the skills, authority, and resources for aligned decisions.",
		Domain:              DomainDSC,
		Parent:              "DSC.Prevention",
		SiblingRelationship: AND,
		Unit:                "probability that responsible persons have necessary skills/resources",
	},
	{
		Function:            DSCIncentives,
		Name:                "Incentives",
		Description:         "Motivate personnel to make decisions aligned with expectations and objectives.",
		Domain:              DomainDSC,
		Parent:              "DSC.Prevention",
		SiblingRelationship: AND,
		Unit:                "probability that appropriate incentives are in place",
	},

	// DSC / Identification
	{
		Function:            DSCIdentification,
		Name:                "Identify Misaligned Decisions",
		Description:         "Enable identification of decisions not aligned with organizational expectations.",
		Domain:              DomainDSC,
		Parent:              "DSC.Identification",
		SiblingRelationship: AND,
		Unit:                "elapsed time from misaligned decision to identification",
	},

	// DSC / Correction
	{
		Function:            DSCCorrection,
		Name:                "Correct Misaligned Decisions",
		Description:         "Correct causes and outcomes of misaligned decisions. Fulfilled by controls within other functions.",
		Domain:              DomainDSC,
		Parent:              "DSC.Correction",
		SiblingRelationship: AND,
		Unit:                "elapsed time from recognition to correction",
	},
}

// catalogIndex is built at init time for fast lookup.
var catalogIndex map[Function]FunctionInfo

func init() {
	catalogIndex = make(map[Function]FunctionInfo, len(Catalog))
	for _, fi := range Catalog {
		catalogIndex[fi.Function] = fi
	}
}

// Lookup returns the FunctionInfo for a given function, and whether it exists.
func Lookup(f Function) (FunctionInfo, bool) {
	fi, ok := catalogIndex[f]
	return fi, ok
}

// Functions returns all functions in the given domain.
// If domain is empty, all functions are returned.
func Functions(domain Domain) []FunctionInfo {
	if domain == "" {
		out := make([]FunctionInfo, len(Catalog))
		copy(out, Catalog)
		return out
	}
	var out []FunctionInfo
	for _, fi := range Catalog {
		if fi.Domain == domain {
			out = append(out, fi)
		}
	}
	return out
}

// Parents returns the distinct parent groupings for a domain, in catalog order.
func Parents(domain Domain) []string {
	seen := map[string]bool{}
	var out []string
	for _, fi := range Catalog {
		if fi.Domain == domain && !seen[fi.Parent] {
			seen[fi.Parent] = true
			out = append(out, fi.Parent)
		}
	}
	return out
}
