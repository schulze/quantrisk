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
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/schulze/quantrisk/risk"
)

var (
	flagFile      = flag.String("file", "", "CSV of scenario name and parameters")
	flagYears     = flag.Int("years", 100000, "Number of years to simulate")
	flagSigDigits = flag.Int("sigdigits", 3, "Number of significant digits in output values")
	flagPlot      = flag.Bool("plot", false, "Print a SVG of loss exceedance curve to stdout")
	flagCurrency  = flag.String("currency", "$", "Currency to use to output monetary values")
)

func main() {
	flag.Parse()
	var in io.Reader
	var out io.Writer = os.Stdout
	if *flagFile != "" {
		f, err := os.Open(*flagFile)
		if err != nil {
			log.Fatal(err)
		}
		in = f
	} else {
		in = os.Stdin
	}
	losses := csvToSimpleLoss(in)
	m := risk.MultiLoss(losses)
	priorities := m.PrioritizedLosses()

	if *flagPlot {
		m.LossExceedanceCurve(*flagYears, "Aggregated Loss Exceedance", out)
		return
	}

	for _, v := range priorities {
		fmt.Fprintf(out, "%v,%v,%v\n", v.Identifier, v.Name,
			sigDigits(v.Loss, *flagSigDigits, *flagCurrency))
	}

}

// csvToSimpleLoss reads CSV (as described in RFC 4180) from the io.Reader in.
// Each record should contain the fields:  label, name, probability, low_loss, high_loss.
func csvToSimpleLoss(in io.Reader) []risk.SimpleLoss {
	records, err := csv.NewReader(in).ReadAll()
	if err != nil {
		panic(err)
	}
	losses := make([]risk.SimpleLoss, 0)
	for _, v := range records {
		p, _ := strconv.ParseFloat(v[2], 64)
		low, _ := strconv.ParseFloat(v[3], 64)
		high, _ := strconv.ParseFloat(v[4], 64)
		s := risk.NewSimpleLoss(v[0], v[1], p, low, high)
		losses = append(losses, s)
	}
	return losses
}

// sigDigits rounds f to a desired number of signigicant digits and returns a formatted currency string.
// For example sigDigits(1234.56, 3, "$") returns "$1,230".
func sigDigits(f float64, digits int, currency string) string {
	f = math.Floor(f)
	l := len(strconv.FormatFloat(f, 'f', 0, 64)) - digits
	factor := math.Pow(10, float64(l))
	f = math.Floor(f/factor) * factor

	p := message.NewPrinter(language.English)
	n := p.Sprintf("%d", int(f))
	return "\"" + *flagCurrency + n + "\""
}
