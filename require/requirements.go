package require

import (
	"maps"

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

// Assure checks if the requirements are met in the given request. If not, it
// returns false. Per the function protocol, Crossplane sets the map key to an
// empty Resources message when it tried to fetch a resource that does not exist.
// So:
//
//   - key missing → not requested yet → wait for the next iteration
//   - key present, wrong item count → real error
//   - all keys present and valid → continue
//
// Errors still return provided=true so that existing callers following the
// README pattern (checking !provided before err != nil) keep surfacing errors
// instead of silently waiting.
func (r *Requirements) Assure(req *fnv1.RunFunctionRequest) (provided bool, err error) {
	if len(r.selectors) == 0 {
		return true, nil
	}
	rr := req.GetRequiredResources()
	provided = true
	for name, sel := range r.selectors {
		extra, ok := rr[name]
		if !ok {
			provided = false // not fetched yet, wait for next iteration
			continue
		}
		l := len(extra.GetItems())
		switch sel.GetMatch().(type) {
		case *fnv1.ResourceSelector_MatchName:
			if l != 1 {
				return true, errors.Errorf(
					"expected 1 required resource for name %q, but got %d",
					name,
					l,
				)
			}
		case *fnv1.ResourceSelector_MatchLabels:
			if l == 0 {
				return true, errors.Errorf(
					"expected >= 1 required resources for name %q, but got %d",
					name,
					l,
				)
			}
		}
	}
	return provided, nil
}

// Register registers the requirements in the given response, merging with any
// existing requirements rather than overwriting them.
func (r *Requirements) Register(rsp *fnv1.RunFunctionResponse) {
	if len(r.selectors) == 0 {
		return
	}
	if rsp.GetRequirements() == nil {
		rsp.Requirements = &fnv1.Requirements{
			Resources: map[string]*fnv1.ResourceSelector{},
		}
	}
	maps.Copy(rsp.GetRequirements().GetResources(), r.selectors)
}
