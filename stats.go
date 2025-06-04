package main

import (
	"math"
	"sort"
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
