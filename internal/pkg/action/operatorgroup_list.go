package action

import (
	"context"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/kubectl-operator/pkg/action"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/**
Plan:
- Show the Operator Group Tenancy model by namespace
- Show namespaces that have an OperatorGroup in them.
- Show sub-namespaces
- NAMESPACE, TYPE, TARGETS
*/

// A Tenant is some Namespace that may or may not have an OperatorGroup.
// The ParentTenants and ChildrenTenants describe the relationship between this
// Tenant namespace and how the OperatorGroup ties them together.
type Tenant struct {
	Namespace       string
	OperatorGroup   *v1.OperatorGroup // The optional OperatorGroup associated with the Tenant
	ParentTenants   []*Tenant         // Tenants that watch this tenant.
	ChildrenTenants []*Tenant         // Tenants that this tenant watches, possibly including self.
}
type OperatorGroupList struct {
	config *action.Configuration

	ShowGraph bool
}

func NewOperatorGroupList(cfg *action.Configuration) *OperatorGroupList {
	return &OperatorGroupList{
		config: cfg,
	}
}

// Retrieve a list of Tenants that represent a Namespace with and without an OperatorGroup
func (l *OperatorGroupList) Run(ctx context.Context) ([]*Tenant, error) {
	ogs := v1.OperatorGroupList{}
	options := client.ListOptions{Namespace: l.config.Namespace}
	if err := l.config.Client.List(ctx, &ogs, &options); err != nil {
		return nil, err
	}
	// ObjectGroups by Namespace name.
	ogMap := make(map[string]*v1.OperatorGroup, len(ogs.Items))
	for _, og := range ogs.Items {
		og := og
		ogMap[og.Namespace] = &og
	}

	// Graph of Tenants
	// Tenant has children, including (optionally) self.
	// Tenant has parents, not including self.
	tenantMap := make(map[string]*Tenant)

	// Convert each ObjectGroup to a Tenant, decorating each Tenant
	coreTenants := make([]*Tenant, len(ogs.Items))
	for i, og := range ogs.Items {
		og := og
		t := Tenant{
			OperatorGroup: &og,
			Namespace:     og.Namespace,
		}
		coreTenants[i] = &t
		tenantMap[og.Namespace] = &t
	}

	// Process each core Tenant's ObjectGroup's target namespace, linking them up in a graph.
	for _, t := range coreTenants {
		for _, ns := range t.OperatorGroup.Spec.TargetNamespaces {
			childTenant := tenantMap[ns]
			if childTenant == nil {
				childTenant = &Tenant{Namespace: ns}
				tenantMap[ns] = childTenant
			}
			t.ChildrenTenants = append(t.ChildrenTenants, childTenant)

			if childTenant.Namespace != t.Namespace {
				// Don't add self as a parent
				childTenant.ParentTenants = append(childTenant.ParentTenants, t)
			}
		}
	}
	tenants := make([]*Tenant, 0, len(tenantMap))
	for _, t := range tenantMap {
		tenants = append(tenants, t)
	}
	return tenants, nil
}
