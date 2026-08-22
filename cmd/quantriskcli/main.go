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
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	_ "modernc.org/sqlite"

	"encoding/csv"

	"github.com/schulze/quantrisk/chart"
	"github.com/schulze/quantrisk/fair"
	"github.com/schulze/quantrisk/internal/model"
	"github.com/schulze/quantrisk/internal/store"
)

var (
	flagDB = flag.String("db", "quantrisk.db", "SQLite database path")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: quantriskcli [flags] <command> [command-flags]\n\n")
		fmt.Fprintf(os.Stderr, "Administration tool for quantriskd.\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  create-user     Create a user (for automation/seeding)\n")
		fmt.Fprintf(os.Stderr, "  create-session  Create a session token for a user\n")
		fmt.Fprintf(os.Stderr, "  migrate         Run database migrations\n")
		fmt.Fprintf(os.Stderr, "  seed            Load realistic fixture data\n")
		fmt.Fprintf(os.Stderr, "  import          Import CSV data into the database\n")
		fmt.Fprintf(os.Stderr, "  simulate        Run Monte Carlo simulation from CSV\n")
		fmt.Fprintf(os.Stderr, "\nGlobal flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	switch flag.Args()[0] {
	case "create-user":
		runCreateUser(flag.Args()[1:])
	case "create-session":
		runCreateSession(flag.Args()[1:])
	case "migrate":
		runMigrate(flag.Args()[1:])
	case "seed":
		runSeed(flag.Args()[1:])
	case "import":
		runImport(flag.Args()[1:])
	case "simulate":
		runSimulate(flag.Args()[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", flag.Args()[0])
		flag.Usage()
		os.Exit(1)
	}
}

// openDB opens the SQLite database with WAL mode, foreign keys, and a
// busy timeout so CLI writes don't conflict with the running daemon.
func openDB() (*sql.DB, error) {
	dsn := *flagDB + "?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

// create-user

func runCreateUser(args []string) {
	fs := flag.NewFlagSet("create-user", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: quantriskcli -db <path> create-user <username> [display-name]\n")
		fmt.Fprintf(os.Stderr, "\nCreate a user directly in the database (no passkey).\n")
		fmt.Fprintf(os.Stderr, "Useful for seeding/automation. Prints the user ID.\n")
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}
	username := fs.Arg(0)
	displayName := username
	if fs.NArg() >= 2 {
		displayName = fs.Arg(1)
	}

	s, err := store.New(*flagDB)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	// Idempotent: if user already exists, print their ID.
	if u, err := s.GetUserByUsername(username); err == nil {
		fmt.Println(u.ID)
		return
	}

	u, err := s.CreateUser(username, displayName)
	if err != nil {
		log.Fatalf("create user: %v", err)
	}
	fmt.Println(u.ID)
}

// create-session

func runCreateSession(args []string) {
	fs := flag.NewFlagSet("create-session", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: quantriskcli -db <path> create-session <username>\n")
		fmt.Fprintf(os.Stderr, "\nCreate a session token for the given user.\n")
		fmt.Fprintf(os.Stderr, "Prints the session token (use as Cookie: qr_session=<token>).\n")
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}
	username := fs.Arg(0)

	s, err := store.New(*flagDB)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	u, err := s.GetUserByUsername(username)
	if err != nil {
		log.Fatalf("user %q not found: %v", username, err)
	}

	token, err := s.CreateSession(u.ID, 24*time.Hour)
	if err != nil {
		log.Fatalf("create session: %v", err)
	}
	fmt.Println(token)
}

// migrate

func runMigrate(args []string) {
	migrateFlags := flag.NewFlagSet("migrate", flag.ExitOnError)
	to := migrateFlags.Int("to", 0, "Target migration version (0 = latest)")
	status := migrateFlags.Bool("status", false, "Show migration status")
	migrateFlags.Parse(args)

	db, err := openDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if *status {
		st, err := store.GetMigrationStatus(db)
		if err != nil {
			log.Fatalf("get migration status: %v", err)
		}
		fmt.Printf("Current version: %d\n", st.Current)
		fmt.Printf("Latest available: %d\n", st.Available)
		if len(st.Applied) > 0 {
			fmt.Println("\nApplied migrations:")
			for _, a := range st.Applied {
				fmt.Printf("  %s\n", a)
			}
		}
		if len(st.Pending) > 0 {
			fmt.Println("\nPending migrations:")
			for _, p := range st.Pending {
				fmt.Printf("  %s\n", p)
			}
		} else {
			fmt.Println("\nNo pending migrations.")
		}
		return
	}

	if err := store.Migrate(db, *to); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	st, err := store.GetMigrationStatus(db)
	if err != nil {
		log.Fatalf("get migration status: %v", err)
	}
	fmt.Printf("Database at version %d.\n", st.Current)
	if len(st.Pending) == 0 {
		fmt.Println("All migrations applied.")
	} else {
		fmt.Printf("%d migration(s) still pending.\n", len(st.Pending))
	}
}

// import

func runImport(args []string) {
	importFlags := flag.NewFlagSet("import", flag.ExitOnError)
	risksFile := importFlags.String("risks", "", "CSV file of risks (identifier, name, probability, low_loss, high_loss)")
	importFlags.Parse(args)

	if *risksFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: quantriskcli -db <path> import -risks <file.csv>\n")
		os.Exit(1)
	}

	s, err := store.New(*flagDB)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer s.Close()

	importRisks(s, *risksFile)
}

func importRisks(s *store.Store, path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	scenarios := csvToScenarios(f)

	var created, updated int
	for _, sc := range scenarios {
		r := &model.Risk{
			Identifier: sc.Identifier,
			Scenario:   sc.Name,
			LossEvent:  sc.LossEvent,
		}

		existing, err := s.GetRiskByIdentifier(r.Identifier)
		if err == nil {
			r.ID = existing.ID
			if err := s.UpdateRisk(r); err != nil {
				log.Fatalf("update risk %s: %v", r.Identifier, err)
			}
			updated++
		} else {
			if err := s.CreateRisk(r); err != nil {
				log.Fatalf("create risk %s: %v", r.Identifier, err)
			}
			created++
		}
	}

	fmt.Printf("Imported %d risk(s): %d created, %d updated.\n", created+updated, created, updated)
}

// simulate (CSV analysis, original behavior)

func runSimulate(args []string) {
	simFlags := flag.NewFlagSet("simulate", flag.ExitOnError)
	file := simFlags.String("file", "", "CSV of scenario name and parameters (stdin if omitted)")
	years := simFlags.Int("years", 100000, "Number of years to simulate")
	sigdigits := simFlags.Int("sigdigits", 3, "Number of significant digits")
	plot := simFlags.Bool("plot", false, "Output SVG loss exceedance curve")
	currency := simFlags.String("currency", "$", "Currency symbol")
	simFlags.Parse(args)

	var in io.Reader
	var out io.Writer = os.Stdout
	if *file != "" {
		f, err := os.Open(*file)
		if err != nil {
			log.Fatal(err)
		}
		in = f
	} else {
		in = os.Stdin
	}
	scenarios := csvToScenarios(in)
	priorities, err := fair.PrioritizedLosses(scenarios)
	if err != nil {
		log.Fatal(err)
	}

	if *plot {
		renderLEC(scenarios, *years, "Aggregated Loss Exceedance", out)
		return
	}

	for _, v := range priorities {
		fmt.Fprintf(out, "%v,%v,%v\n", v.Identifier, v.Name,
			sigDigits(v.Loss, *sigdigits, *currency))
	}
}

func renderLEC(scenarios []fair.Scenario, years int, title string, w io.Writer) {
	perScenario, aggregate, err := fair.SimulateMulti(scenarios, 10000)
	if err != nil {
		log.Fatal(err)
	}

	var curves []chart.NamedCurve
	for i, s := range scenarios {
		points := chart.ExceedancePointsFrom(perScenario[i].YearlyLosses, 99)
		curves = append(curves, chart.NamedCurve{
			Label:  s.Label(),
			Points: points,
		})
	}
	if len(scenarios) > 1 {
		points := chart.ExceedancePointsFrom(aggregate.YearlyLosses, 99)
		curves = append(curves, chart.NamedCurve{
			Label:  "Aggregate",
			Points: points,
		})
	}
	chart.RenderLEC(curves, title, w)
}

// csvToScenarios reads CSV records (identifier, name, frequency, low_loss, high_loss)
// and returns fair.Scenario values using Poisson frequency + LogNormal magnitude.
func csvToScenarios(in io.Reader) []fair.Scenario {
	records, err := csv.NewReader(in).ReadAll()
	if err != nil {
		log.Fatalf("read CSV: %v", err)
	}
	scenarios := make([]fair.Scenario, 0, len(records))
	for _, v := range records {
		p, _ := strconv.ParseFloat(v[2], 64)
		low, _ := strconv.ParseFloat(v[3], 64)
		high, _ := strconv.ParseFloat(v[4], 64)
		scenarios = append(scenarios, fair.NewSimpleScenario(v[0], v[1], p, low, high))
	}
	return scenarios
}

func sigDigits(f float64, digits int, currency string) string {
	f = math.Floor(f)
	l := len(strconv.FormatFloat(f, 'f', 0, 64)) - digits
	factor := math.Pow(10, float64(l))
	f = math.Floor(f/factor) * factor

	p := message.NewPrinter(language.English)
	n := p.Sprintf("%d", int(f))
	return "\"" + currency + n + "\""
}
