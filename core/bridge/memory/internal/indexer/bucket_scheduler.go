package indexer

import "sort"

type BucketScheduler struct{}

func (BucketScheduler) Order(buckets []Bucket) []Bucket {
	ordered := append([]Bucket(nil), buckets...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left := normalizeBucket(ordered[i])
		right := normalizeBucket(ordered[j])
		if left.Month != right.Month {
			return left.Month < right.Month
		}
		if left.Namespace != right.Namespace {
			return left.Namespace < right.Namespace
		}
		if left.Workspace != right.Workspace {
			return left.Workspace < right.Workspace
		}
		return left.SegmentPath < right.SegmentPath
	})
	return ordered
}
