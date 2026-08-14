package inventory

import (
	"slices"
	"strings"

	"github.com/go-json-experiment/json"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

type builder struct {
	compositeFunc MappingFunc
	inputFunc     MappingFunc
	mapping       Mapping
	// if enabled, the json conversion will be graceful without rejecting unknown members
	gracefulConversion bool
	conversionIssues   []string
}

func BuildInventory(
	log logging.Logger,
	req *fnv1.RunFunctionRequest,
	opts ...Option,
) (Reader, error) {
	inv, _, err := BuildGracefulInventory(log, req, false, opts...)
	return inv, err
}

func BuildGracefulInventory(
	log logging.Logger,
	req *fnv1.RunFunctionRequest,
	gracefulConversion bool,
	opts ...Option,
) (r Reader, conversionIssues []string, err error) {
	if req == nil {
		return nil, nil, errors.New("request cannot be nil")
	}

	// Initialize options with default values
	bldr := &builder{
		gracefulConversion: gracefulConversion,
	}

	// Apply all provided options
	for _, opt := range opts {
		if err := opt(bldr); err != nil {
			return nil, nil, err
		}
	}

	inventory := &inventory{
		request: req,
	}

	// Handle composite resources if XRD mapping is provided
	if err := bldr.buildComposite(log, req, bldr.compositeFunc, inventory); err != nil {
		return nil, nil, err
	}
	// Handle input if input mapping is provided
	if err := bldr.buildInput(log, req, bldr.inputFunc, inventory); err != nil {
		return nil, nil, err
	}

	// Handle composed resources
	if inventory.desiredComposed, err = bldr.buildInventoryResourceMap(
		log,
		req.GetDesired().GetResources(),
		bldr.mapping,
	); err != nil {
		return nil, nil, err
	}

	if inventory.observedComposed, err = bldr.buildInventoryResourceMap(
		log,
		req.GetObserved().GetResources(),
		bldr.mapping,
	); err != nil {
		return nil, nil, err
	}

	if inventory.requirements, err = bldr.buildInventoryResourcesMap(
		log,
		req.GetRequiredResources(),
		bldr.mapping,
	); err != nil {
		return nil, nil, err
	}

	slices.Sort(bldr.conversionIssues)
	bldr.conversionIssues = slices.Compact(bldr.conversionIssues)

	return inventory, bldr.conversionIssues, nil
}

func (bldr *builder) buildComposite(
	log logging.Logger,
	req *fnv1.RunFunctionRequest,
	xrdMapping MappingFunc,
	inventory *inventory,
) error {
	if xrdMapping == nil {
		return nil
	}

	// Handle desired composite
	if desired := req.GetDesired(); desired != nil && desired.GetComposite() != nil {
		res, err := bldr.convertTo(desired.GetComposite().GetResource(), xrdMapping)
		if err != nil {
			return err
		}
		inventory.desiredComposite = &Resource{
			ro:         res,
			ready:      desired.GetComposite().GetReady(),
			connection: desired.GetComposite().GetConnectionDetails(),
		}
		log.Debug("Got desired composite present in request")
	}

	// Handle observed composite
	if observed := req.GetObserved(); observed != nil && observed.GetComposite() != nil {
		res, err := bldr.convertTo(observed.GetComposite().GetResource(), xrdMapping)
		if err != nil {
			return err
		}
		inventory.observedComposite = &Resource{
			ro:         res,
			ready:      observed.GetComposite().GetReady(),
			connection: observed.GetComposite().GetConnectionDetails(),
		}
		log.Debug("Got observed composite present in request")
	}
	return nil
}

func (bldr *builder) buildInput(
	log logging.Logger,
	req *fnv1.RunFunctionRequest,
	inputMapping MappingFunc,
	inventory *inventory,
) error {
	if inputMapping == nil {
		return nil
	}

	var err error

	// Handle input
	if input := req.GetInput(); input != nil {
		if inventory.input, err = bldr.convertTo(input, inputMapping); err != nil {
			return err
		}
	} else {
		log.Debug("No input present in request")
	}

	return nil
}

func (bldr *builder) buildInventoryResourceMap(
	log logging.Logger,
	res map[string]*fnv1.Resource,
	mapping Mapping,
) (ResourceMap, error) {
	im := ResourceMap{}
	for name, res := range res {
		m := mappingFor(mapping, name)
		if m != nil {
			ro, err := bldr.convertTo(res.GetResource(), m)
			if err != nil {
				return im, err
			}
			im[name] = &Resource{
				ro:         ro,
				ready:      res.GetReady(),
				connection: res.GetConnectionDetails(),
			}
		} else {
			log.Debug("No inventory mapper defined for resource", "name", name)
		}
	}

	return im, nil
}

func (bldr *builder) convertTo(s *structpb.Struct, mf MappingFunc) (runtime.Object, error) {
	if s == nil || mf == nil {
		return nil, nil
	}
	ro, err := mf()
	if err != nil {
		return nil, err
	}

	if err := bldr.asObject(s, ro); err != nil {
		return nil, err
	}
	return ro, nil
}

// asObject gets the supplied Kubernetes object from the supplied struct.
// from: https://github.com/crossplane/function-sdk-go/blob/main/resource/resource.go
func (bldr *builder) asObject(s *structpb.Struct, o runtime.Object) error {
	// We try to avoid a JSON round-trip if o is backed by unstructured data.
	// Any type that is or embeds *unstructured.Unstructured has this method.
	if u, ok := o.(interface{ SetUnstructuredContent(_ map[string]any) }); ok {
		u.SetUnstructuredContent(s.AsMap())
		return nil
	}

	b, err := protojson.Marshal(s)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal %T to JSON", s)
	}

	err = json.Unmarshal(b, o, json.RejectUnknownMembers(true))
	if err == nil {
		return nil
	}

	var semanticErr *json.SemanticError
	if !bldr.gracefulConversion || !errors.As(err, &semanticErr) {
		return errors.Wrapf(err, "cannot marshal %T to JSON", s)
	}

	if bldr.gracefulConversion {
		bldr.conversionIssues = append(bldr.conversionIssues, semanticErr.Error())
		// re-try with graceful conversion
		return errors.Wrapf(
			json.Unmarshal(b, o, json.RejectUnknownMembers(false)),
			"cannot unmarshal JSON from %T into %T",
			s,
			o,
		)
	}

	return nil
}

func (bldr *builder) buildInventoryResourcesMap(
	_ logging.Logger,
	resMap map[string]*fnv1.Resources,
	mapping Mapping,
) (ResourcesMap, error) {
	im := ResourcesMap{}
	for name, res := range resMap {
		m := mappingFor(mapping, name)
		if m != nil {
			ros := make([]*Resource, len(res.GetItems()))
			for i, item := range res.GetItems() {
				ro, err := bldr.convertTo(item.GetResource(), m)
				if err != nil {
					return im, err
				}

				ros[i] = &Resource{
					ro:         ro,
					ready:      item.GetReady(),
					connection: item.GetConnectionDetails(),
				}
			}

			im[name] = ros
		}
	}

	return im, nil
}

func mappingFor(im Mapping, name string) MappingFunc {
	if im == nil {
		return nil
	}
	for prefix, mf := range im {
		if strings.HasPrefix(name, prefix) {
			return mf
		}
	}
	return nil
}

// hasSharedPrefix reports whether either string is a prefix of the other.
func hasSharedPrefix(a, b string) bool {
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}

// FindConflictingPrefix reports whether the provided prefix matches any key in the mapping-by-prefix
// relationship (in either direction). It returns (ok, matchedKey).
func FindConflictingPrefix[T any](m map[string]T, prefix string) string {
	for k := range m {
		if k != prefix && hasSharedPrefix(prefix, k) {
			return k
		}
	}
	return ""
}
