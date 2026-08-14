package require

import (
	"github.com/pkg/errors"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

type Requirements struct {
	selectors map[string]*fnv1.ResourceSelector
}

// Requires returns a new Requirements instance that requires the given
// extra resources.
func Requires(requirements ...Requirement) (*Requirements, error) {
	r := &Requirements{selectors: map[string]*fnv1.ResourceSelector{}}
	for _, req := range requirements {
		name, sel, err := req.Get()
		if err != nil {
			return nil, err
		}
		r.selectors[name] = sel
	}
	return r, nil
}

// Assure checks if the requirements are met in the given request. If not, it returns false.
func (r *Requirements) Assure(req *fnv1.RunFunctionRequest) (provided bool, err error) {
	if len(r.selectors) == 0 {
		return true, nil
	}
	rr := req.GetRequiredResources()
	if rr == nil {
		// required resources not yet provided - return and wait for the next iteration
		return false, nil
	}
	for name, sel := range r.selectors {
		extra, ok := rr[name]
		if !ok {
			return true, errors.Errorf("required resource %q not found", name)
		}
		switch sel.GetMatch().(type) {
		case *fnv1.ResourceSelector_MatchName:
			if l := len(extra.GetItems()); l != 1 {
				return true, errors.Errorf(
					"expected 1 required resource for name %q, but got %d",
					name,
					l,
				)
			}
		case *fnv1.ResourceSelector_MatchLabels:
			if l := len(extra.GetItems()); l == 0 {
				return true, errors.Errorf(
					"expected > 1 required resources for name %q, but got %d",
					name,
					l,
				)
			}
		}
	}
	return true, nil
}

// Register registers the requirements in the given response.
func (r *Requirements) Register(rsp *fnv1.RunFunctionResponse) {
	if len(r.selectors) > 0 {
		rsp.Requirements = &fnv1.Requirements{
			Resources: r.selectors,
		}
	}
}
