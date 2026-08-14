package inventory

import (
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
)

// BuildFromRequest converts a given requests data to the inventory.
func BuildFromRequest(req *fnv1.RunFunctionRequest, log logging.Logger, targets ...any) error {
	opts, err := BuildInventoryOptions(targets...)
	if err != nil {
		return err
	}
	inv, err := BuildInventory(log, req, opts...)
	if err != nil {
		return err
	}

	return Inject(inv, targets...)
}

// BuildGracefulFromRequest converts a given requests data to the inventory.
func BuildGracefulFromRequest(
	req *fnv1.RunFunctionRequest,
	log logging.Logger,
	targets ...any,
) (conversionIssues []string, err error) {
	opts, err := BuildInventoryOptions(targets...)
	if err != nil {
		return nil, err
	}
	inv, conversionIssues, err := BuildGracefulInventory(log, req, true, opts...)
	if err != nil {
		return conversionIssues, err
	}

	err = Inject(inv, targets...)
	return conversionIssues, err
}
