package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	internalaction "github.com/operator-framework/kubectl-operator/internal/pkg/action"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func newOperatorGroupListCmd(cfg *action.Configuration) *cobra.Command {
	var allNamespaces bool
	i := internalaction.NewOperatorGroupList(cfg)
	//i.Logf = log.Printf

	cmd := &cobra.Command{
		Use:   "list-operatorgroups",
		Short: "List all OperatorGroups",
		Example: `
#
# Output a tabular list of OperatorGroups as Tenants:
# 
$ kubectl operator list-operatorgroups -A

# Output a graph of OperatorGroups as Tenants in mermaid format:
#
$ kubectl operator list-operatorgroups -A -g

#
# Output a graph of OperatorGroups as Tenants as a SVG image:
#
$ kubectl operator list-operatorgroups -A -g | \
		docker run --rm -i -v "$PWD":/data ghcr.io/mermaid-js/mermaid-cli/mermaid-cli -o /data/tenants.svg

# Note:  mermaid has a default maxTextSize of 30 000 characters.  To override this, generate a JSON-formatted initialization file for
# mermaid like this (using 300 000 for the limit):
$ cat << EOF > ./mermaid.json
{ "maxTextSize": 300000 }
EOF
# and then pass the file for initialization configuration, via the '-c' option, like:
$ kubectl operator list-operatorgroups -A -g | \
		docker run --rm -i -v "$PWD":/data ghcr.io/mermaid-js/mermaid-cli/mermaid-cli -c /data/mermaid.json -o /data/operatorhubio-catalog.svg


`,

		Run: func(cmd *cobra.Command, args []string) {
			if allNamespaces {
				cfg.Namespace = v1.NamespaceAll
			}

			tenants, err := i.Run(cmd.Context())
			if err != nil {
				log.Fatalf("failed to list operator groups: %v", err)
			}
			if len(tenants) == 0 {
				log.Printf("No Namespaces with OperatorGroups found in the current context.")
			}

			sort.SliceStable(tenants, func(i, j int) bool {
				return strings.Compare(tenants[i].Namespace, tenants[j].Namespace) < 0
			})

			if i.ShowGraph {
				out := createGraph(tenants)
				for _, v := range out {
					fmt.Print(v)
				}
			} else {
				tw := tabwriter.NewWriter(os.Stdout, 3, 4, 2, ' ', 0)
				_, _ = fmt.Fprintf(tw, "TENANT\tTYPE\tSUBTENANTS\tTARGETNAMESPACES\tPARENTTENANTS\n")
				for _, tenant := range tenants {
					if tenant.OperatorGroup != nil {
						targetnss := strings.Join(tenant.OperatorGroup.Spec.TargetNamespaces, ",")

						childTenants := make([]string, 0, len(tenant.ChildrenTenants))
						for _, childTenant := range tenant.ChildrenTenants {
							if childTenant.Namespace != tenant.Namespace {
								childTenants = append(childTenants, childTenant.Namespace)
							}
						}
						childTenantsStr := strings.Join(childTenants, ",")
						parentTenants := make([]string, 0, len(tenant.ParentTenants))
						for _, parentTenant := range tenant.ParentTenants {
							parentTenants = append(parentTenants, parentTenant.Namespace)
						}
						parentTenantsStr := strings.Join(parentTenants, ",")

						fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", tenant.Namespace, effectiveInstallMode(*tenant), childTenantsStr, targetnss, parentTenantsStr)
					}
				}
				_ = tw.Flush()
			}
		},
	}
	cmd.Flags().BoolVarP(&i.ShowGraph, "graph", "g", false, "generate a graph")
	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "list operators in all namespaces")

	return cmd
}

func createGraph(tenants []*internalaction.Tenant) []string {
	out := make([]string, 0)
	in := "  "

	// Create a Mermaid graph of "Tenants"

	// Header and formatting
	out = append(out,
		"flowchart LR\n",
		in+"classDef tenant fill:#eFe,stroke:#000;\n",
		in+"classDef bad fill:#f99,stroke:#000;\n",
		in+"classDef ns fill:#fff,stroke:#000,stroke-width:1px;\n",
		in+fmt.Sprintf("classDef im-%s fill:#eef,stroke:#000,stroke-width:2px;\n", v1alpha1.InstallModeTypeOwnNamespace),
		in+fmt.Sprintf("classDef im-%s fill:#ddf,stroke:#000,stroke-width:2px;\n", v1alpha1.InstallModeTypeSingleNamespace),
		in+fmt.Sprintf("classDef im-%s fill:#ccf,stroke:#000,stroke-width:2px;\n", v1alpha1.InstallModeTypeMultiNamespace),
		"\n",
	)

	// Legend
	out = append(out,
		in+"subgraph Legend [Legend]\n",
		in+"direction TB",
		in+"LNS[Namespace]:::ns\n",
		in+fmt.Sprintf("LOGOWN(%s):::im-%s\n", v1alpha1.InstallModeTypeOwnNamespace, v1alpha1.InstallModeTypeOwnNamespace),
		in+fmt.Sprintf("LOGSINGLE(%s):::im-%s\n", v1alpha1.InstallModeTypeSingleNamespace, v1alpha1.InstallModeTypeSingleNamespace),
		in+fmt.Sprintf("LOGMULTI(%s):::im-%s\n", v1alpha1.InstallModeTypeMultiNamespace, v1alpha1.InstallModeTypeMultiNamespace),
		in+"LOGNS[Operand Namespace]:::ns\n",
		in+"end",
	)

	// The Node in a graph
	type Node struct {
		ID     string                 // The node id.
		Tenant *internalaction.Tenant // The Tenant represented by the Node
		Class  string                 // The style class
		Name   string                 // The display name
		im     string                 // Effective installmode or ""
	}

	// Convert to Nodes by namespace/id name
	nodeMap := make(map[string]Node, len(tenants))
	for i := range tenants {
		t := tenants[i]
		n := Node{
			ID:     t.Namespace,
			Tenant: t,
		}

		// No OG means an operand namespace
		if t.OperatorGroup == nil {
			n.Name = "[" + n.ID + "]"
			n.Class = "ns"
		} else {
			n.Name = "(" + n.ID + ")"
			n.im = effectiveInstallMode(*t)
			n.Class = "im-" + n.im
		}

		if len(t.ParentTenants) > 1 {
			n.Class = "bad"
		}
		nodeMap[t.Namespace] = n
	}

	// Avoid cycles by recording our edges.
	processed := mapset.NewSet[string]()

	for _, node := range nodeMap {
		level := 0

		// Ignore AllNamespace mode Tenants and Namespace-only tenants
		if node.Tenant.OperatorGroup != nil && node.im != (string)(v1alpha1.InstallModeTypeAllNamespaces) {

			var writeTenant func(leftTenant *internalaction.Tenant, rtTenant *internalaction.Tenant)
			writeTenant = func(leftTenant *internalaction.Tenant, rtTenant *internalaction.Tenant) {
				level++
				space := strings.Repeat(" ", level*2)

				leftNode := nodeMap[leftTenant.Namespace]
				rtNode := nodeMap[rtTenant.Namespace]

				edge := leftNode.ID + ">" + rtNode.ID
				if !processed.Contains(edge) {
					processed.Add(edge)

					out = append(out,
						space+fmt.Sprintf("%s%s:::%s --> %s%s:::%s\n", leftNode.ID, leftNode.Name, leftNode.Class, rtNode.ID, rtNode.Name, rtNode.Class),
					)

					for j := range rtTenant.ChildrenTenants {
						var childTenant *internalaction.Tenant = rtTenant.ChildrenTenants[j]
						if childTenant == rtTenant {
							// Write out self-references
							writeTenant(rtTenant, rtTenant)
						} else {
							writeTenant(rtTenant, childTenant)
						}
					}
				}
				level--
			}

			// Any tenant that has no parents, is the Root of the tenant tree.
			if len(node.Tenant.ParentTenants) == 0 {
				// Linked Tenants will be embedded in the parent Tenant subgraph
				out = append(out,
					in+fmt.Sprintf("t-%s:::tenant\n", node.ID),
					in+fmt.Sprintf("subgraph t-%s[%s]\n", node.ID, node.ID),
					in+"direction LR\n",
				)

				for _, childTenant := range node.Tenant.ChildrenTenants {
					writeTenant(node.Tenant, childTenant)
				}

				out = append(out,
					in+"end\n\n",
				)
			}
		}
	}

	return out
}

func effectiveInstallMode(t internalaction.Tenant) string {
	switch len(t.OperatorGroup.Spec.TargetNamespaces) {
	case 0:
		return (string)(v1alpha1.InstallModeTypeAllNamespaces)
	case 1:
		switch t.OperatorGroup.Spec.TargetNamespaces[0] {
		case "":
			return (string)(v1alpha1.InstallModeTypeAllNamespaces)
		case t.OperatorGroup.Namespace:
			return (string)(v1alpha1.InstallModeTypeOwnNamespace)
		default:
			return string(v1alpha1.InstallModeTypeSingleNamespace)
		}
	default:
		return string(v1alpha1.InstallModeTypeMultiNamespace)
	}
}
