package schema

import "github.com/infobloxopen/apx/internal/validator"

// ExtractCRD summarizes a Kubernetes CustomResourceDefinition for the catalog
// site detail panel: its GVK, scope, and served/storage versions. The full
// structural schema is available via the raw source viewer.
func ExtractCRD(filePath string) (*CRDSchema, error) {
	info, err := validator.ExtractCRDInfo(filePath)
	if err != nil {
		return nil, err
	}
	return &CRDSchema{
		Group:          info.Group,
		Kind:           info.Kind,
		Plural:         info.Plural,
		Scope:          info.Scope,
		ServedVersions: info.ServedVersions,
		StorageVersion: info.StorageVersion,
	}, nil
}
