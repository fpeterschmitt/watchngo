//go:generate mockgen -source=filter.go -destination=mock_filter_test.go -package=pkg_test Filter

package pkg

type Filter interface {
	MatchString(file string) bool
}
