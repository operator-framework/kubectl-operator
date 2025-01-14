package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/operator-framework/kubectl-operator/pkg/action/v1"
	"io"
	"os"
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

func newExtensionListOperandsCmd(cfg *action.Configuration) *cobra.Command {
	l := v0.NewOperatorListOperands(cfg)
	output := ""
	validOutputs := []string{"json", "yaml"}

	cmd := &cobra.Command{
		Use:   "list-operands <clusterExtensionName>",
		Short: "List operands of an installed cluster extension",
		Long: `List operands of an installed cluster extension.

This command lists all operands found throughout the cluster for the cluster
extension specified on the command line.

Operand kinds are determined from the CustomResourceDefinitions that bear a label
matching the cluster extension's name.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeOutput := func(io.Writer, *unstructured.UnstructuredList) error { panic("writeOutput was not set") } //nolint:staticcheck
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
	if _, err := fmt.Fprintf(tw, "GROUP\tKIND\tNAMESPACE\tNAME\tAGE\n"); err != nil {
		return err
	}
	for _, o := range operands.Items {
		gk := o.GroupVersionKind().GroupKind()
		age := time.Since(o.GetCreationTimestamp().Time)
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", gk.Group, gk.Kind, o.GetNamespace(), o.GetName(), duration.HumanDuration(age)); err != nil {
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
