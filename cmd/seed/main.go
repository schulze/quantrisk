// Command seed populates a quantrisk database with realistic security
// risk scenarios, controls, requirements, gaps, and their
// relationships. All data is inserted directly via the store package.
//
// Usage:
//
//	seed -db quantrisk.db
//	seed -db quantrisk.db -user admin "Admin User"   # also create a login user
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/schulze/quantrisk/fair"
	"github.com/schulze/quantrisk/fair/cam"
	"github.com/schulze/quantrisk/internal/model"
	"github.com/schulze/quantrisk/internal/store"
)

var (
	flagDB   = flag.String("db", "quantrisk.db", "SQLite database path")
	flagUser = flag.String("user", "", "Create a login user (username)")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: seed [flags] [display-name]\n\n")
		fmt.Fprintf(os.Stderr, "Populate a quantrisk database with realistic fixture data.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	s, err := store.New(*flagDB)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer s.Close()

	// Optional: create user
	if *flagUser != "" {
		displayName := *flagUser
		if flag.NArg() > 0 {
			displayName = flag.Arg(0)
		}
		if u, err := s.GetUserByUsername(*flagUser); err == nil {
			log.Printf("user %q already exists (id=%d)", *flagUser, u.ID)
		} else {
			u, err := s.CreateUser(*flagUser, displayName)
			if err != nil {
				log.Fatalf("create user: %v", err)
			}
			log.Printf("created user %q (id=%d)", *flagUser, u.ID)
		}
	}

	seedAll(s)
	log.Println("seed complete")
}

// Fixtures

func seedAll(s *store.Store) {
	risks := seedRisks(s)
	reqs := seedRequirements(s)
	ctrls := seedControls(s)
	gaps := seedGaps(s, ctrls, reqs)

	seedControlFunctions(s, ctrls)
	seedLinks(s, risks, reqs, ctrls, gaps)
}

// Risk scenarios

func seedRisks(s *store.Store) map[string]int64 {
	scenarios := []model.Risk{
		{
			Identifier: "RISK-001",
			Scenario:   "Ransomware encrypts production systems causing extended outage and extortion payment",
			LossEvent: fair.LossEvent{
				LEFMode: fair.LEFDecomposed,
				TEF:     fair.Estimate{Min: 1, ML: 3, Max: 8, Rationale: "Based on industry rates for mid-size orgs; Verizon DBIR shows 1-8 ransomware attempts/year"},
				Susc:    fair.Estimate{Min: 0.05, ML: 0.15, Max: 0.35, Rationale: "Endpoint protection and segmentation reduce but don't eliminate susceptibility"},
				PL: fair.LossForm{
					ProdL: fair.Estimate{Min: 500_000, ML: 2_000_000, Max: 8_000_000, Rationale: "5-30 day outage at $100K-$250K/day revenue impact"},
					RespC: fair.Estimate{Min: 200_000, ML: 750_000, Max: 2_000_000, Rationale: "IR firm, forensics, rebuild; Ponemon avg IR cost $750K"},
					ReplC: fair.Estimate{Min: 50_000, ML: 200_000, Max: 500_000, Rationale: "Hardware replacement, re-imaging, new licenses"},
				},
				SL: fair.LossForm{
					FinJu: fair.Estimate{Min: 0, ML: 100_000, Max: 1_000_000, Rationale: "Potential regulatory fines if PII exposed during attack"},
					RepuD: fair.Estimate{Min: 100_000, ML: 500_000, Max: 3_000_000, Rationale: "Customer churn, brand damage; 2-5% revenue impact over 12 months"},
				},
			},
		},
		{
			Identifier: "RISK-002",
			Scenario:   "Credential stuffing attack compromises customer accounts leading to data theft and fraud",
			LossEvent: fair.LossEvent{
				LEFMode:   fair.LEFDirect,
				DirectLEF: fair.Estimate{Min: 2, ML: 6, Max: 15, Rationale: "Continuous credential stuffing attempts; 2-15 successful campaigns/year per Akamai data"},
				PL: fair.LossForm{
					ProdL: fair.Estimate{Min: 10_000, ML: 50_000, Max: 200_000, Rationale: "Account lockout procedures, forced password resets reduce platform availability"},
					RespC: fair.Estimate{Min: 25_000, ML: 100_000, Max: 400_000, Rationale: "Investigation, customer notification, credit monitoring per breach notification laws"},
				},
				SL: fair.LossForm{
					FinJu: fair.Estimate{Min: 50_000, ML: 250_000, Max: 2_000_000, Rationale: "GDPR/CCPA fines, FTC enforcement for inadequate credential protection"},
					RepuD: fair.Estimate{Min: 50_000, ML: 200_000, Max: 1_000_000, Rationale: "Customer trust erosion; higher churn in financial/health sectors"},
				},
			},
		},
		{
			Identifier: "RISK-003",
			Scenario:   "Insider threat exfiltrates proprietary source code or trade secrets to competitor",
			LossEvent: fair.LossEvent{
				LEFMode: fair.LEFDecomposed,
				TEF:     fair.Estimate{Min: 0.5, ML: 2, Max: 5, Rationale: "Privileged insiders with access; turnover creates 0.5-5 threat events/year"},
				Susc:    fair.Estimate{Min: 0.10, ML: 0.25, Max: 0.50, Rationale: "DLP and access controls exist but motivated insiders can bypass"},
				PL: fair.LossForm{
					CAdvL: fair.Estimate{Min: 1_000_000, ML: 5_000_000, Max: 25_000_000, Rationale: "Lost competitive advantage; R&D investment written off; depends on IP value"},
				},
				SL: fair.LossForm{
					FinJu: fair.Estimate{Min: 100_000, ML: 500_000, Max: 5_000_000, Rationale: "Litigation costs, trade secret misappropriation claims"},
					RepuD: fair.Estimate{Min: 50_000, ML: 250_000, Max: 1_000_000, Rationale: "Signal of weak culture; investor/partner concern"},
				},
			},
		},
		{
			Identifier: "RISK-004",
			Scenario:   "Supply chain compromise injects malicious code via third-party dependency",
			LossEvent: fair.LossEvent{
				LEFMode:   fair.LEFDirect,
				DirectLEF: fair.Estimate{Min: 0.3, ML: 1, Max: 3, Rationale: "SolarWinds/log4j-class events; Sonatype reports 700% increase in supply chain attacks"},
				PL: fair.LossForm{
					ProdL: fair.Estimate{Min: 200_000, ML: 1_000_000, Max: 5_000_000, Rationale: "Emergency patching, service degradation during remediation window"},
					RespC: fair.Estimate{Min: 100_000, ML: 500_000, Max: 3_000_000, Rationale: "Forensic analysis of blast radius, vendor coordination, customer communication"},
					ReplC: fair.Estimate{Min: 50_000, ML: 150_000, Max: 500_000, Rationale: "Dependency replacement, code audit, build pipeline rebuilds"},
				},
				SL: fair.LossForm{
					FinJu: fair.Estimate{Min: 0, ML: 200_000, Max: 5_000_000, Rationale: "Regulatory scrutiny for inadequate vendor risk management"},
					RepuD: fair.Estimate{Min: 100_000, ML: 500_000, Max: 2_000_000, Rationale: "Downstream customer impact erodes trust"},
				},
			},
		},
		{
			Identifier: "RISK-005",
			Scenario:   "Cloud misconfiguration exposes sensitive data in public storage bucket",
			LossEvent: fair.LossEvent{
				LEFMode: fair.LEFDecomposed,
				TEF:     fair.Estimate{Min: 2, ML: 6, Max: 15, Rationale: "Configuration changes are frequent; each is a potential misconfig event"},
				Susc:    fair.Estimate{Min: 0.02, ML: 0.08, Max: 0.20, Rationale: "IaC templates and CSPM reduce but don't eliminate misconfiguration risk"},
				PL: fair.LossForm{
					RespC: fair.Estimate{Min: 50_000, ML: 200_000, Max: 800_000, Rationale: "Breach notification, forensics, legal review; depends on data volume"},
				},
				SL: fair.LossForm{
					FinJu: fair.Estimate{Min: 100_000, ML: 500_000, Max: 5_000_000, Rationale: "Per-record fines under GDPR ($150/record × exposed records)"},
					RepuD: fair.Estimate{Min: 50_000, ML: 300_000, Max: 2_000_000, Rationale: "Public disclosure of preventable misconfiguration"},
				},
			},
		},
		{
			Identifier: "RISK-006",
			Scenario:   "Exploitation of unpatched critical vulnerability in internet-facing application",
			LossEvent: fair.LossEvent{
				LEFMode: fair.LEFDecomposed,
				TEF:     fair.Estimate{Min: 5, ML: 15, Max: 50, Rationale: "Automated scanning + weaponized exploits within days of CVE publication"},
				Susc:    fair.Estimate{Min: 0.03, ML: 0.10, Max: 0.25, Rationale: "WAF, segmentation, and rapid patching reduce susceptibility"},
				PL: fair.LossForm{
					ProdL: fair.Estimate{Min: 100_000, ML: 500_000, Max: 2_000_000, Rationale: "Emergency patching windows, potential service disruption"},
					RespC: fair.Estimate{Min: 50_000, ML: 300_000, Max: 1_500_000, Rationale: "Incident response, compromise assessment, remediation"},
				},
				SL: fair.LossForm{
					FinJu: fair.Estimate{Min: 0, ML: 100_000, Max: 2_000_000, Rationale: "Regulatory fines if breach results from known, unpatched vuln"},
				},
			},
		},
		{
			Identifier: "RISK-007",
			Scenario:   "Business email compromise (BEC) leads to fraudulent wire transfer",
			LossEvent: fair.LossEvent{
				LEFMode:   fair.LEFDirect,
				DirectLEF: fair.Estimate{Min: 1, ML: 4, Max: 12, Rationale: "FBI IC3 data: BEC is #1 loss category; mid-size orgs see 1-12 attempts/year"},
				PL: fair.LossForm{
					ProdL: fair.Estimate{Min: 5_000, ML: 25_000, Max: 100_000, Rationale: "Staff time investigating, process disruption"},
					RespC: fair.Estimate{Min: 10_000, ML: 50_000, Max: 200_000, Rationale: "Legal, bank recovery attempts, law enforcement coordination"},
				},
				SL: fair.LossForm{
					FinJu: fair.Estimate{Min: 25_000, ML: 150_000, Max: 1_000_000, Rationale: "Direct financial loss from fraudulent transfer; FBI avg BEC loss $150K"},
				},
			},
		},
		{
			Identifier: "RISK-008",
			Scenario:   "DDoS attack overwhelms internet-facing services causing prolonged outage",
			LossEvent: fair.LossEvent{
				LEFMode:   fair.LEFDirect,
				DirectLEF: fair.Estimate{Min: 2, ML: 6, Max: 20, Rationale: "Cloudflare/Akamai report 2-20 significant DDoS events/year for mid-tier targets"},
				PL: fair.LossForm{
					ProdL: fair.Estimate{Min: 50_000, ML: 250_000, Max: 1_000_000, Rationale: "Revenue loss during 1-48 hour outage depending on mitigation speed"},
					RespC: fair.Estimate{Min: 10_000, ML: 50_000, Max: 200_000, Rationale: "DDoS mitigation service activation, engineering escalation"},
				},
				SL: fair.LossForm{
					RepuD: fair.Estimate{Min: 10_000, ML: 50_000, Max: 300_000, Rationale: "Customer frustration, SLA breach penalties"},
				},
			},
		},
	}

	ids := make(map[string]int64, len(scenarios))
	for i := range scenarios {
		r := &scenarios[i]
		if err := s.CreateRisk(r); err != nil {
			log.Fatalf("create risk %s: %v", r.Identifier, err)
		}
		ids[r.Identifier] = r.ID
		log.Printf("risk  %s (id=%d): %s", r.Identifier, r.ID, truncate(r.Scenario, 70))
	}
	return ids
}

// Requirements (regulatory / framework controls)

func seedRequirements(s *store.Store) map[string]int64 {
	reqs := []model.Requirement{
		{Identifier: "REQ-001", Name: "Access Control", Source: "NIST SP 800-53 AC-2",
			Description: "Manage system accounts including establishing, activating, modifying, reviewing, disabling, and removing accounts"},
		{Identifier: "REQ-002", Name: "Incident Response Plan", Source: "NIST SP 800-53 IR-1",
			Description: "Establish and maintain an incident response capability including preparation, detection, analysis, containment, eradication, and recovery"},
		{Identifier: "REQ-003", Name: "Vulnerability Management", Source: "NIST SP 800-53 RA-5",
			Description: "Scan for vulnerabilities in systems and applications, remediate findings within defined timeframes based on severity"},
		{Identifier: "REQ-004", Name: "Configuration Management", Source: "NIST SP 800-53 CM-2",
			Description: "Develop, document, and maintain baseline configurations and inventories of organizational systems"},
		{Identifier: "REQ-005", Name: "Audit and Accountability", Source: "NIST SP 800-53 AU-2",
			Description: "Identify events that the system must be capable of logging and ensure audit records contain sufficient information"},
		{Identifier: "REQ-006", Name: "System and Communications Protection", Source: "NIST SP 800-53 SC-7",
			Description: "Observe and control communications at external and key internal boundaries using boundary protection devices"},
		{Identifier: "REQ-007", Name: "Supply Chain Risk Management", Source: "NIST SP 800-53 SR-3",
			Description: "Employ supply chain controls and safeguards against supply chain risks including assessment of third-party components"},
		{Identifier: "REQ-008", Name: "Security Awareness Training", Source: "NIST SP 800-53 AT-2",
			Description: "Provide security awareness training to system users including recognizing and reporting indicators of insider threat and social engineering"},
		{Identifier: "REQ-009", Name: "Data Protection", Source: "GDPR Article 32",
			Description: "Implement appropriate technical and organizational measures to ensure a level of security appropriate to the risk of processing personal data"},
		{Identifier: "REQ-010", Name: "Patch Management", Source: "PCI DSS 6.3.3",
			Description: "Install critical security patches within one month of release; establish a process for identifying and ranking vulnerabilities"},
	}

	ids := make(map[string]int64, len(reqs))
	for i := range reqs {
		r := &reqs[i]
		if err := s.CreateRequirement(r); err != nil {
			log.Fatalf("create requirement %s: %v", r.Identifier, err)
		}
		ids[r.Identifier] = r.ID
		log.Printf("req   %s (id=%d): %s [%s]", r.Identifier, r.ID, r.Name, r.Source)
	}
	return ids
}

// Controls

func seedControls(s *store.Store) map[string]int64 {
	ctrls := []model.Control{
		{Identifier: "CTL-001", Name: "Endpoint Detection and Response", Status: "implemented",
			Description: "CrowdStrike Falcon deployed on all endpoints and servers; automated prevention, detection, and response to malware, ransomware, and fileless attacks"},
		{Identifier: "CTL-002", Name: "Multi-Factor Authentication", Status: "implemented",
			Description: "Phishing-resistant MFA (FIDO2/WebAuthn) required for all user authentication to production systems and administrative interfaces"},
		{Identifier: "CTL-003", Name: "Network Segmentation", Status: "implemented",
			Description: "Production, staging, and corporate networks segmented with zero-trust micro-segmentation; east-west traffic filtered by workload identity"},
		{Identifier: "CTL-004", Name: "SIEM and Security Monitoring", Status: "implemented",
			Description: "Centralized SIEM (Splunk) with 24/7 SOC monitoring; automated correlation rules and alert triage for security events across all infrastructure"},
		{Identifier: "CTL-005", Name: "Vulnerability Scanning and Patching", Status: "implemented",
			Description: "Automated weekly vulnerability scanning (Qualys) with SLA-driven patching: critical ≤7d, high ≤30d, medium ≤90d; emergency patching process for zero-days"},
		{Identifier: "CTL-006", Name: "Data Loss Prevention", Status: "implemented",
			Description: "DLP policies on email, cloud storage, and endpoints monitoring for PII, source code, and classified data exfiltration; alerts route to SOC"},
		{Identifier: "CTL-007", Name: "Cloud Security Posture Management", Status: "implemented",
			Description: "Wiz CSPM continuously scanning AWS/GCP for misconfigurations, public exposure, and IAM policy drift; auto-remediation for critical findings"},
		{Identifier: "CTL-008", Name: "Software Composition Analysis", Status: "implemented",
			Description: "Snyk integrated into CI/CD pipeline scanning all dependencies for known vulnerabilities; builds blocked on critical/high findings"},
		{Identifier: "CTL-009", Name: "Security Awareness Program", Status: "verified",
			Description: "Quarterly phishing simulations, annual security training, targeted BEC awareness for finance; metrics tracked and reported to CISO"},
		{Identifier: "CTL-010", Name: "Incident Response Procedures", Status: "verified",
			Description: "Documented IR playbooks for ransomware, data breach, BEC, and DDoS; tabletop exercises quarterly; retainer with external IR firm (CrowdStrike Services)"},
		{Identifier: "CTL-011", Name: "Backup and Recovery", Status: "implemented",
			Description: "Immutable backups with 3-2-1 strategy; daily incrementals, weekly fulls; tested restores monthly; air-gapped copies for ransomware resilience"},
		{Identifier: "CTL-012", Name: "DDoS Mitigation", Status: "implemented",
			Description: "Cloudflare enterprise DDoS protection with always-on L3/L4 mitigation and on-demand L7 scrubbing; automatic failover within 10 seconds"},
		{Identifier: "CTL-013", Name: "Branch Protection and Code Review", Status: "implemented",
			Description: "All production branches require pull request with minimum 2 approvals, CI pass, and CODEOWNERS review; force push disabled; admin enforcement enabled"},
		{Identifier: "CTL-014", Name: "Wire Transfer Controls", Status: "implemented",
			Description: "Dual authorization for wire transfers >$10K; out-of-band verification via phone callback for new payees; daily reconciliation by finance team"},
	}

	ids := make(map[string]int64, len(ctrls))
	for i := range ctrls {
		c := &ctrls[i]
		if err := s.CreateControl(c); err != nil {
			log.Fatalf("create control %s: %v", c.Identifier, err)
		}
		ids[c.Identifier] = c.ID
		log.Printf("ctrl  %s (id=%d): %s [%s]", c.Identifier, c.ID, c.Name, c.Status)
	}
	return ids
}

// Gaps

func seedGaps(s *store.Store, ctrls, reqs map[string]int64) map[string]int64 {
	ctlType := "control"
	reqType := "requirement"
	ctlID := func(id string) *int64 { v := ctrls[id]; return &v }
	reqID := func(id string) *int64 { v := reqs[id]; return &v }

	gaps := []model.Gap{
		{Identifier: "GAP-001", Name: "Incomplete endpoint coverage", Severity: "high", Status: "open",
			Description: "31 endpoints (1.3%) lack EDR agent — primarily legacy OT systems and contractor laptops. Compensating controls (network isolation) partially mitigate but don't provide equivalent detection capability.",
			ParentType:  &ctlType, ParentID: ctlID("CTL-001")},
		{Identifier: "GAP-002", Name: "Service account MFA exceptions", Severity: "medium", Status: "mitigated",
			Description: "3 service accounts use certificate-based authentication instead of FIDO2. Mitigated by short-lived certificates (8h), restricted network access, and enhanced monitoring.",
			ParentType:  &ctlType, ParentID: ctlID("CTL-002")},
		{Identifier: "GAP-003", Name: "Legacy application patching delays", Severity: "high", Status: "open",
			Description: "2 legacy applications cannot be patched within SLA due to vendor dependencies; currently 45 days overdue on 3 high-severity CVEs. Virtual patching via WAF applied as interim measure.",
			ParentType:  &ctlType, ParentID: ctlID("CTL-005")},
		{Identifier: "GAP-004", Name: "No SBOM for legacy services", Severity: "medium", Status: "open",
			Description: "12 legacy microservices predate SCA pipeline integration; no software bill of materials exists. Manual audit planned for Q3.",
			ParentType:  &ctlType, ParentID: ctlID("CTL-008")},
		{Identifier: "GAP-005", Name: "Third-party risk assessment backlog", Severity: "high", Status: "open",
			Description: "Supply chain risk management requirement not fully met: 8 of 23 critical vendors have not completed annual security assessment. Due to staffing gap in vendor risk team.",
			ParentType:  &reqType, ParentID: reqID("REQ-007")},
		{Identifier: "GAP-006", Name: "Insufficient DLP coverage for API egress", Severity: "medium", Status: "open",
			Description: "DLP policies cover email and cloud storage but lack coverage for API-based data egress paths. Engineering team scoped a 6-week project to extend DLP to API gateway.",
			ParentType:  &ctlType, ParentID: ctlID("CTL-006")},
	}

	ids := make(map[string]int64, len(gaps))
	for i := range gaps {
		g := &gaps[i]
		if err := s.CreateGap(g); err != nil {
			log.Fatalf("create gap %s: %v", g.Identifier, err)
		}
		ids[g.Identifier] = g.ID
		log.Printf("gap   %s (id=%d): %s [%s/%s]", g.Identifier, g.ID, g.Name, g.Severity, g.Status)
	}
	return ids
}

// FAIR-CAM control function assignments

func seedControlFunctions(s *store.Store, ctrls map[string]int64) {
	assignments := []model.ControlFunction{
		// CTL-001: EDR — detection + response
		{ControlID: ctrls["CTL-001"], Function: cam.LECVisibility,
			Effectiveness: cam.Effectiveness{
				Capability:  fair.Estimate{Min: 0.85, ML: 0.92, Max: 0.97},
				Coverage:    fair.Estimate{Min: 0.95, ML: 0.987, Max: 1.0},
				Reliability: fair.Estimate{Min: 0.90, ML: 0.95, Max: 0.99},
			},
			Notes: "Falcon sensor provides kernel-level visibility; 98.7% deployment coverage"},
		{ControlID: ctrls["CTL-001"], Function: cam.LECRecognition,
			Effectiveness: cam.Effectiveness{
				Capability:  fair.Estimate{Min: 0.80, ML: 0.88, Max: 0.95},
				Coverage:    fair.Estimate{Min: 0.85, ML: 0.92, Max: 0.97},
				Reliability: fair.Estimate{Min: 0.88, ML: 0.93, Max: 0.98},
			},
			Notes: "ML-based detection with behavioral IOAs; tested against MITRE ATT&CK"},
		{ControlID: ctrls["CTL-001"], Function: cam.LECEventTermination,
			Effectiveness: cam.Effectiveness{
				Capability:  fair.Estimate{Min: 0.75, ML: 0.85, Max: 0.93},
				Coverage:    fair.Estimate{Min: 0.80, ML: 0.90, Max: 0.97},
				Reliability: fair.Estimate{Min: 0.85, ML: 0.92, Max: 0.97},
			},
			Notes: "Automated process kill, network isolation; response within seconds"},

		// CTL-002: MFA — prevention (resistance)
		{ControlID: ctrls["CTL-002"], Function: cam.LECResistance,
			Effectiveness: cam.Effectiveness{
				Capability:  fair.Estimate{Min: 0.90, ML: 0.97, Max: 0.99},
				Coverage:    fair.Estimate{Min: 0.96, ML: 0.99, Max: 1.0},
				Reliability: fair.Estimate{Min: 0.92, ML: 0.96, Max: 0.99},
			},
			Notes: "FIDO2 is phishing-resistant; near-complete enrollment"},

		// CTL-003: Network segmentation — prevention (avoidance)
		{ControlID: ctrls["CTL-003"], Function: cam.LECAvoidance,
			Effectiveness: cam.Effectiveness{
				Capability:  fair.Estimate{Min: 0.70, ML: 0.82, Max: 0.92},
				Coverage:    fair.Estimate{Min: 0.75, ML: 0.85, Max: 0.95},
				Reliability: fair.Estimate{Min: 0.80, ML: 0.88, Max: 0.95},
			},
			Notes: "Micro-segmentation limits lateral movement; reduces contact frequency between threat agents and assets"},

		// CTL-004: SIEM — detection
		{ControlID: ctrls["CTL-004"], Function: cam.LECMonitoring,
			Effectiveness: cam.Effectiveness{
				Capability:  fair.Estimate{Min: 0.80, ML: 0.88, Max: 0.95},
				Coverage:    fair.Estimate{Min: 0.85, ML: 0.92, Max: 0.98},
				Reliability: fair.Estimate{Min: 0.90, ML: 0.95, Max: 0.99},
			},
			Notes: "24/7 SOC monitoring with automated triage; average alert-to-review time 4 minutes"},

		// CTL-005: Vulnerability scanning — VMC identification
		{ControlID: ctrls["CTL-005"], Function: cam.VMCControlMonitoring,
			Effectiveness: cam.Effectiveness{
				Capability:  fair.Estimate{Min: 0.82, ML: 0.90, Max: 0.96},
				Coverage:    fair.Estimate{Min: 0.88, ML: 0.93, Max: 0.98},
				Reliability: fair.Estimate{Min: 0.85, ML: 0.92, Max: 0.97},
			},
			Notes: "Weekly automated scans; SLA compliance at 96%"},

		// CTL-007: CSPM — VMC identification + prevention
		{ControlID: ctrls["CTL-007"], Function: cam.VMCControlMonitoring,
			Effectiveness: cam.Effectiveness{
				Capability:  fair.Estimate{Min: 0.85, ML: 0.92, Max: 0.97},
				Coverage:    fair.Estimate{Min: 0.90, ML: 0.95, Max: 0.99},
				Reliability: fair.Estimate{Min: 0.88, ML: 0.94, Max: 0.98},
			},
			Notes: "Continuous CSPM scanning across AWS/GCP; auto-remediation for critical drift"},

		// CTL-009: Security awareness — DSC prevention
		{ControlID: ctrls["CTL-009"], Function: cam.DSCDefinedExpectations,
			Effectiveness: cam.Effectiveness{
				Capability:  fair.Estimate{Min: 0.60, ML: 0.72, Max: 0.85},
				Coverage:    fair.Estimate{Min: 0.90, ML: 0.95, Max: 0.99},
				Reliability: fair.Estimate{Min: 0.55, ML: 0.68, Max: 0.80},
			},
			Notes: "Phishing click rate improving but human factor inherently variable"},

		// CTL-011: Backup — response (resilience)
		{ControlID: ctrls["CTL-011"], Function: cam.LECResilience,
			Effectiveness: cam.Effectiveness{
				Capability:  fair.Estimate{Min: 0.85, ML: 0.92, Max: 0.97},
				Coverage:    fair.Estimate{Min: 0.88, ML: 0.94, Max: 0.99},
				Reliability: fair.Estimate{Min: 0.82, ML: 0.90, Max: 0.96},
			},
			Notes: "Immutable backups with monthly restore testing; RTO 2h14m achieved"},

		// CTL-012: DDoS mitigation — response (event termination)
		{ControlID: ctrls["CTL-012"], Function: cam.LECEventTermination,
			Effectiveness: cam.Effectiveness{
				Capability:  fair.Estimate{Min: 0.90, ML: 0.96, Max: 0.99},
				Coverage:    fair.Estimate{Min: 0.85, ML: 0.93, Max: 0.98},
				Reliability: fair.Estimate{Min: 0.92, ML: 0.97, Max: 0.99},
			},
			Notes: "Always-on L3/L4 + on-demand L7; auto-mitigation <10s"},

		// CTL-013: Branch protection — prevention (resistance)
		{ControlID: ctrls["CTL-013"], Function: cam.LECResistance,
			Effectiveness: cam.Effectiveness{
				Capability:  fair.Estimate{Min: 0.88, ML: 0.94, Max: 0.98},
				Coverage:    fair.Estimate{Min: 0.92, ML: 0.96, Max: 1.0},
				Reliability: fair.Estimate{Min: 0.90, ML: 0.95, Max: 0.99},
			},
			Notes: "Enforced via GitHub branch protection; admin enforcement enabled"},

		// CTL-014: Wire transfer controls — prevention (resistance)
		{ControlID: ctrls["CTL-014"], Function: cam.LECResistance,
			Effectiveness: cam.Effectiveness{
				Capability:  fair.Estimate{Min: 0.82, ML: 0.90, Max: 0.96},
				Coverage:    fair.Estimate{Min: 0.90, ML: 0.95, Max: 0.99},
				Reliability: fair.Estimate{Min: 0.78, ML: 0.87, Max: 0.94},
			},
			Notes: "Dual authorization + phone callback; human process reliability is inherently lower"},
	}

	for i := range assignments {
		cf := &assignments[i]
		if err := s.CreateControlFunction(cf); err != nil {
			log.Fatalf("create control function: %v", err)
		}
		log.Printf("cfunc %s on control id=%d", cf.Function, cf.ControlID)
	}
}

// Relationships

func seedLinks(s *store.Store, risks, reqs, ctrls, gaps map[string]int64) {
	// Controls ↔ Risks
	controlRisks := [][2]string{
		{"CTL-001", "RISK-001"}, // EDR ↔ ransomware
		{"CTL-001", "RISK-006"}, // EDR ↔ unpatched vuln exploitation
		{"CTL-002", "RISK-002"}, // MFA ↔ credential stuffing
		{"CTL-002", "RISK-003"}, // MFA ↔ insider threat
		{"CTL-003", "RISK-001"}, // segmentation ↔ ransomware
		{"CTL-003", "RISK-003"}, // segmentation ↔ insider threat
		{"CTL-004", "RISK-001"}, // SIEM ↔ ransomware
		{"CTL-004", "RISK-003"}, // SIEM ↔ insider threat
		{"CTL-004", "RISK-006"}, // SIEM ↔ unpatched vuln
		{"CTL-005", "RISK-006"}, // vuln scanning ↔ unpatched vuln
		{"CTL-005", "RISK-004"}, // vuln scanning ↔ supply chain
		{"CTL-006", "RISK-003"}, // DLP ↔ insider threat
		{"CTL-007", "RISK-005"}, // CSPM ↔ cloud misconfig
		{"CTL-008", "RISK-004"}, // SCA ↔ supply chain
		{"CTL-009", "RISK-002"}, // awareness ↔ credential stuffing
		{"CTL-009", "RISK-007"}, // awareness ↔ BEC
		{"CTL-010", "RISK-001"}, // IR procedures ↔ ransomware
		{"CTL-011", "RISK-001"}, // backup ↔ ransomware
		{"CTL-012", "RISK-008"}, // DDoS mitigation ↔ DDoS
		{"CTL-013", "RISK-004"}, // branch protection ↔ supply chain
		{"CTL-014", "RISK-007"}, // wire transfer controls ↔ BEC
	}
	for _, link := range controlRisks {
		if err := s.LinkControlRisk(ctrls[link[0]], risks[link[1]]); err != nil {
			log.Printf("link %s↔%s: %v", link[0], link[1], err)
		}
	}
	log.Printf("linked %d control↔risk relationships", len(controlRisks))

	// Controls ↔ Requirements
	controlReqs := [][2]string{
		{"CTL-002", "REQ-001"}, // MFA ↔ access control
		{"CTL-003", "REQ-006"}, // segmentation ↔ system/comm protection
		{"CTL-004", "REQ-005"}, // SIEM ↔ audit and accountability
		{"CTL-005", "REQ-003"}, // vuln scanning ↔ vulnerability mgmt
		{"CTL-005", "REQ-010"}, // vuln scanning ↔ patch management
		{"CTL-007", "REQ-004"}, // CSPM ↔ configuration management
		{"CTL-008", "REQ-007"}, // SCA ↔ supply chain risk mgmt
		{"CTL-009", "REQ-008"}, // awareness ↔ security training
		{"CTL-010", "REQ-002"}, // IR procedures ↔ incident response
		{"CTL-006", "REQ-009"}, // DLP ↔ data protection (GDPR)
	}
	for _, link := range controlReqs {
		if err := s.LinkControlRequirement(ctrls[link[0]], reqs[link[1]]); err != nil {
			log.Printf("link %s↔%s: %v", link[0], link[1], err)
		}
	}
	log.Printf("linked %d control↔requirement relationships", len(controlReqs))

}

// helpers

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
