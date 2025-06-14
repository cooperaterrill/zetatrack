package main

import (
	"math"
	"sort"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func median(times []int64) int64 {
	sort.Slice(times, func(i int, j int) bool {
		return times[i] < times[j]
	})
	var med int64
	if len(times)%2 == 0 {
		med = (times[len(times)/2-1] + times[len(times)/2]) / 2
	} else {
		med = times[len(times)/2]
	}
	return med
}

func iqr(times []int64) int64 {
	sort.Slice(times, func(i int, j int) bool {
		return times[i] < times[j]
	})
	n := len(times) / 2
	var lowerTimes []int64
	var upperTimes []int64
	for i := 0; i < n; i++ {
		lowerTimes = append(lowerTimes, times[i])
		upperTimes = append(upperTimes, times[len(times)-i-1])
	}

	return median(upperTimes) - median(lowerTimes)
}

func mean(times []int64) int64 {
	var mean int64
	for _, time := range times {
		mean += time
	}
	return mean / int64(len(times))
}

func stdev(times []int64) int64 {
	mean := mean(times)
	var res int64
	for _, time := range times {
		res += (mean - time) * (mean - time)
	}
	res /= int64(len(times) - 1)

	return int64(math.Sqrt(float64(res)))
}

func MeanAndStdev(times []int64) (int64, int64) {
	return mean(times), stdev(times)
}

func MedianAndIqr(times []int64) (int64, int64) {
	return median(times), iqr(times)
}

func GraphScoreOverTime(filepath string) *charts.Line {
	logs := ParseLogFile(filepath)

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Time played",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Min:          0,
			Max:          "dataMax",
			Name:         "Time (milliseconds)",
			NameLocation: "center",
			NameGap:      50,
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Type:         "time",
			Min:          "dataMin",
			Max:          "dataMax",
			Name:         "Run #",
			NameGap:      30,
			NameLocation: "center",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "axis",
		}),
	)

	scores := GetScoreList(logs)
	times := GetLogtimeList(logs)
	items := make([]opts.LineData, len(scores))
	for i := 0; i < len(scores); i++ {
		if scores[i] == 0 {
			continue
		}
		items = append(items, opts.LineData{Value: []interface{}{times[i], scores[i]}})
	}

	line.AddSeries("Performance", items)
	return line
}

func GraphTimePerProblemOverTime(filepath string) *charts.Line {
	logs := ParseLogFile(filepath)
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Time played",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Min:          0,
			Max:          "dataMax",
			Name:         "Time (milliseconds)",
			NameLocation: "center",
			NameGap:      50,
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Type:         "time",
			Min:          "dataMin",
			Max:          "dataMax",
			Name:         "Run #",
			NameGap:      30,
			NameLocation: "center",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "axis",
		}),
	)

	avgs := GetAverageTimePerProblemList(logs)
	times := GetLogtimeList(logs)
	items := make([]opts.LineData, 0)
	for i := 0; i < len(avgs); i++ {
		if avgs[i] == 0 {
			continue
		}
		items = append(items, opts.LineData{Value: []interface{}{times[i], avgs[i]}})
	}

	line.AddSeries("Average time per problem", items)
	return line
}
