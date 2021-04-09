package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/duration"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/kubectl-operator/internal/cmd/internal/log"
	"github.com/operator-framework/kubectl-operator/pkg/action"
)

func newOperatorListOperandsCmd(cfg *action.Configuration) *cobra.Command {
	l := action.NewOperatorListOperands(cfg)
	output := ""
	validOutputs := []string{"json", "yaml"}

	cmd := &cobra.Command{
		Use:   "list-operands <operator>",
		Short: "List operands of an installed operator",
		Long: `List operands of an installed operator.

This command lists all operands found throughout the cluster for the operator
specified on the command line. Since the scope of an operator is restricted by
its operator group, the output will include namespace-scoped operands from the
operator group's target namespaces and all cluster-scoped operands.

To search for operands for an operator in a different namespace, use the
--namespace flag. By default, the namespace from the current context is used.

Operand kinds are determined from the owned CustomResourceDefinitions listed in
the operator's ClusterServiceVersion.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeOutput := func(io.Writer, *unstructured.UnstructuredList) error { panic("writeOutput was not set") }
			switch output {
			case "json":
				writeOutput = writeJSON
			case "yaml":
				writeOutput = writeYAML
			case "":
				writeOutput = writeTable
			default:
				log.Fatalf("invalid value for flag output %q, expected one of %s", output, strings.Join(validOutputs, "|"))
			}

			operands, err := l.Run(cmd.Context(), args[0])
			if err != nil {
				log.Fatalf("list operands: %v", err)
			}

			if len(operands.Items) == 0 {
				log.Print("No resources found")
				return
			}

			sort.Slice(operands.Items, func(i, j int) bool {
				if operands.Items[i].GetAPIVersion() != operands.Items[j].GetAPIVersion() {
					return operands.Items[i].GetAPIVersion() < operands.Items[j].GetAPIVersion()
				}
				if operands.Items[i].GetKind() != operands.Items[j].GetKind() {
					return operands.Items[i].GetKind() < operands.Items[j].GetKind()
				}
				if operands.Items[i].GetNamespace() != operands.Items[j].GetNamespace() {
					return operands.Items[i].GetNamespace() < operands.Items[j].GetNamespace()
				}
				return operands.Items[i].GetName() < operands.Items[j].GetName()
			})
			if err := writeOutput(os.Stdout, operands); err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", output, fmt.Sprintf("Output format. One of: %s", strings.Join(validOutputs, "|")))
	return cmd
}

func writeTable(w io.Writer, operands *unstructured.UnstructuredList) error {
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 3, 4, 2, ' ', 0)
	if _, err := fmt.Fprintf(tw, "APIVERSION\tKIND\tNAMESPACE\tNAME\tAGE\n"); err != nil {
		return err
	}
	for _, o := range operands.Items {
		age := time.Since(o.GetCreationTimestamp().Time)
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", o.GetAPIVersion(), o.GetKind(), o.GetNamespace(), o.GetName(), duration.HumanDuration(age)); err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return err
	}
	return nil
}

func writeJSON(w io.Writer, operands *unstructured.UnstructuredList) error {
	out, err := json.Marshal(operands)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, out, "", "  "); err != nil {
		return err
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return err
	}
	return nil
}

func writeYAML(w io.Writer, operands *unstructured.UnstructuredList) error {
	var jsonWriter bytes.Buffer
	if err := writeJSON(&jsonWriter, operands); err != nil {
		return err
	}
	out, err := yaml.JSONToYAML(jsonWriter.Bytes())
	if err != nil {
		return err
	}
	if _, err := w.Write(out); err != nil {
		return err
	}
	return nil
}
