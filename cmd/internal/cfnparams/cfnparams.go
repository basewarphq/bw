package cfnparams

import (
	"regexp"

	"github.com/cockroachdb/errors"
)

var placeholderRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)

func Resolve(raw map[string]string, ctxValues map[string]string) (map[string]string, error) {
	resolved := make(map[string]string, len(raw))
	for k, v := range raw {
		val, err := interpolate(v, ctxValues)
		if err != nil {
			return nil, errors.Wrapf(err, "parameter %q", k)
		}
		resolved[k] = val
	}
	return resolved, nil
}

func interpolate(val string, ctxValues map[string]string) (string, error) {
	var resolveErr error
	result := placeholderRe.ReplaceAllStringFunc(val, func(match string) string {
		key := placeholderRe.FindStringSubmatch(match)[1]
		v, ok := ctxValues[key]
		if !ok {
			resolveErr = errors.Newf("unknown context key %q", key)
			return match
		}
		return v
	})
	if resolveErr != nil {
		return "", resolveErr
	}
	return result, nil
}
