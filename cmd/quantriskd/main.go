package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/schulze/quantrisk/internal/server"
	"github.com/schulze/quantrisk/internal/store"
)

var (
	flagAddr   = flag.String("addr", ":8080", "HTTP listen address")
	flagDB     = flag.String("db", "quantrisk.db", "SQLite database path")
	flagYears  = flag.Int("years", 100000, "Default number of years to simulate")
	flagOrigin = flag.String("origin", "http://localhost:8080", "WebAuthn origin(s), comma-separated (e.g. https://example.com,https://example.com:8000)")
)

func main() {
	flag.Parse()

	s, err := store.New(*flagDB)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer s.Close()

	// Build origin list from comma-separated -origin flag.
	// This allows accepting requests from multiple origins, e.g. both
	// https://host (reverse proxy) and https://host:8000 (direct port).
	var origins []string
	for _, o := range strings.Split(*flagOrigin, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}

	// Derive RPID (hostname only) from the first origin.
	rpID := origins[0]
	rpID = strings.TrimPrefix(rpID, "https://")
	rpID = strings.TrimPrefix(rpID, "http://")
	if idx := strings.IndexByte(rpID, ':'); idx != -1 {
		rpID = rpID[:idx]
	}
	if idx := strings.IndexByte(rpID, '/'); idx != -1 {
		rpID = rpID[:idx]
	}

	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "secriskquant",
		RPID:          rpID,
		RPOrigins:     origins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationPreferred,
		},
	})
	if err != nil {
		log.Fatalf("webauthn config: %v", err)
	}

	srv := server.New(s, *flagYears, wa)

	log.Printf("listening on %s (origins=%v rpID=%s)", *flagAddr, origins, rpID)
	if err := http.ListenAndServe(*flagAddr, srv); err != nil {
		log.Fatal(err)
	}
}
