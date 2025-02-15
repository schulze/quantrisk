# quantrisk

A simple reimplementation of Netflix's [riskquant](https://github.com/Netflix-Skunkworks/riskquant) library in Go.
When Netflix fist published a [blog post](https://netflixtechblog.com/open-sourcing-riskquant-a-library-for-quantifying-risk-6720cc1e4968) that sparked my interest in quantiative loss models.

But riskquant required downloading gigabites of dependencies in TensorFlow for simple probablility distributions,
so at some point for experimentation I rewrote the tool in Go.
It is an experiment on how to implement loss exceedance computation, not a finished tool.

	% go build .
	% ./quantrisk -h
	Usage of ./quantrisk:
	  -currency string
	    	Currency to use to output monetary values (default "$")
	  -file string
	    	CSV of scenario name and parameters
	  -plot
	    	Print a SVG of loss exceedance curve to stdout
	  -sigdigits int
	    	Number of significant digits in output values (default 3)
	  -years int
	    	Number of years to simulate (default 100000)
	% ./quantrisk -file test.csv > got.csv
	% diff got.csv want.csv
	% ./quantrisk -file test.csv -plot > plot.svg


