package service

import "testing"

func TestLimitForDistanceHitsPageBody(t *testing.T) {
	for distance := 0; distance < 8000; distance++ {
		limit := limitForDistance(distance)
		pageSize := limit + 1
		offset := distance % pageSize
		if offset >= limit {
			t.Fatalf("distance %d limit %d landed on the dropped slot", distance, limit)
		}
	}
}
