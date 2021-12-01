//go:generate mockgen -source=filter.go -destination=mock_filter_test.go -package=pkg_test Filter

package pkg

import "regexp"

type Filter interface {
	Match(file string) bool
}

type FilterRegexp struct {
	re *regexp.Regexp
}

func (f FilterRegexp) Match(file string) bool {
	return f.re.MatchString(file)
}

func NewFilterRegexp(re string) FilterRegexp {
	return FilterRegexp{re: regexp.MustCompile(re)}
}
