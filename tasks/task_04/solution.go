package main

import "math"

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) == 0 {
		return Stats{}
	} else if len(nums) == 1 {
		return Stats{0, 0, 0, 0}
	}
	sum := int64(0)
	min := int64(math.MaxInt64)
	max := int64(math.MinInt64)
	count := 0
	for i := 0; i < len(nums)-1; i++ {
		d := nums[i+1] - nums[i]
		sum += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
		count++
	}
	return Stats{count, sum, min, max}
}
