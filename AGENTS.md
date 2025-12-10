# AGENTS.md

This document provides guidance for AI agents working with the kubectl-operator repository.

---

## Repository Overview

**kubectl-operator** is a kubectl plugin that functions as a package manager for OLM Operators. It provides a CLI interface for managing both OLMv0 operators (legacy, through [Subscriptions](https://github.com/operator-framework/api/blob/master/crds/operators.coreos.com_subscriptions.yaml)) and OLMv1 extensions (next-generation, through ClusterExtensions).

---

## Architecture

### Directory Structure

```
kubectl-operator/
├── main.go                    # Entry point - delegates to internal/cmd
├── internal/
│   ├── cmd/                   # Command definitions
│   │   ├── root.go            # Root command with global flags
│   │   ├── catalog.go         # OLMv0 catalog commands
│   │   ├── operator.go        # OLMv0 operator commands
│   │   ├── olmv1.go           # OLMv1 root command, references the internal/olmv1 for command definitions.
│   │   └── internal/
│   │       ├── log/           # Logging utilities
│   │       └── olmv1/         # OLMv1 commands
│   ├── pkg/                   # Internal business logic packages
│   │   ├── action/            # OLMv0 command action implementations
│   │   ├── v1/
│   │   │   ├── action/        # OLMv1 command action implementations
│   │   │   └── client/        # Port-forwarding catalog client for OLMv1 search command
│   │   ├── operator/          # PackageManifest wrappers
│   │   ├── subscription/      # Subscription wrappers
│   │   ├── catalogsource/     # CatalogSource wrappers
│   │   └── operand/           # Operand strategy logic
│   └── version/               # Version information
├── pkg/
│   └── action/                # Configuration and client setup
├── .github/workflows/         # CI/CD pipelines
├── .bingo/                    # Bingo-managed tool dependencies
└── vendor/                    # Vendored dependencies
```

### Command Structure

```
kubectl operator
├── [OLMv0: Global Flags: --namespace, --timeout]
│
├── catalog                    # OLMv0 catalog management
│   ├── add                    # OLMv1: Add a catalogsource
│   ├── list                   # OLMv1: List catalogsources
│   └── remove                 # OLMv1: Remove a catalogsource
│
├── install                    # OLMv0: Install operator
├── upgrade                    # OLMv0: Upgrade operator
├── uninstall                  # OLMv0: Uninstall operator
├── list                       # OLMv0: List installed operators
├── list-available             # OLMv0: List available operators
├── list-operands              # OLMv0: List operands
├── describe                   # OLMv0: Describe operator
│
├── olmv1                      # OLMv1 resource management
│   ├── get
│   │   ├── extension         # OLMv1: Display clustrerextensions
│   │   └── catalog           # OLMv1: Display clustercatalogs
│   ├── create
│   │   └── catalog           # OLMv1: Create a clustercatalog
│   ├── delete
│   │   ├── extension         # OLMv1: Delete clustrerextensions
│   │   └── catalog           # OLMv1: Delete clustercatalogs
│   ├── update
│   │   ├── extension         # OLMv1: Update mutable fields on a clustrerextension
│   │   └── catalog           # OLMv1: Update mutable fields on a clustercatalog
│   ├── install
│   │   └── extension         # OLMv1: Create a clusterextension to install a package from a catalog
│   └── search
│       └── catalog           # OLMv1: Search packages in catalogs
│
└── version                    # Show version info
```

**Command Implementation Pattern:**
1. Command file in `internal/cmd/` creates Cobra command
2. Flags bound to action struct fields
3. Action instance created from `internal/pkg/action/` or `internal/pkg/v1/action/`
4. `Run(ctx context.Context)` method executes business logic
5. Results formatted and returned to user

### OLM Resources Reference

This section describes the Kubernetes resources that kubectl-operator interacts with. Note that kubectl-operator **does not define** these CRDs - they are defined by OLM itself.

#### OLMv0 Resources

| Resource | API Group | Purpose | kubectl-operator Interaction |
|----------|-----------|---------|------------------------------|
| **ClusterServiceVersion (CSV)** | operators.coreos.com/v1alpha1 | Defines operator metadata, installation strategy, permissions, and owned/required APIs | Created during `install`, monitored during installation, queried by `list`, deleted during `uninstall` |
| **Subscription** | operators.coreos.com/v1alpha1 | Tracks operator updates from a catalog channel; drives automatic upgrades | Created during `install`, updated during `upgrade`, queried by `list`, deleted during `uninstall` |
| **InstallPlan** | operators.coreos.com/v1alpha1 | Calculated list of resources to install/upgrade; requires approval | Created by OLM when Subscription is created, approved during `install` if auto-approval enabled |
| **CatalogSource** | operators.coreos.com/v1alpha1 | Repository of operators and metadata served via gRPC | Created by `catalog add`, listed by `catalog list`, deleted by `catalog remove` |
| **OperatorGroup** | operators.coreos.com/v1 | Groups namespaces for operator installation scope; enables multi-tenancy | Created automatically during `install` if not present and `--create-operator-group` flag is set |
| **PackageManifest** | packages.operators.coreos.com/v1 | Read-only view of available packages from catalogs | Queried by `list-available` to show packages, used during `install` to validate package/channel |

**Resource short names for kubectl:**
```bash
kubectl get csv              # ClusterServiceVersions
kubectl get sub              # Subscriptions
kubectl get ip               # InstallPlans
kubectl get catsrc           # CatalogSources
kubectl get og               # OperatorGroups
kubectl get packagemanifest  # PackageManifests (no short name)
```

#### OLMv1 Resources

| Resource | API Group | Purpose | kubectl-operator Interaction |
|----------|-----------|---------|------------------------------|
| **ClusterExtension** | olm.operatorframework.io/v1alpha1 | Desired state of an installed operator/extension on the cluster | Created by `olmv1 install extension`, queried by `olmv1 get extensions`, updated by `olmv1 update extension`, deleted by `olmv1 delete extension` |
| **ClusterCatalog** | olm.operatorframework.io/v1 | Catalog definition pointing to FBC (File-Based Catalog) image | Created by `olmv1 create catalog`, queried by `olmv1 get catalogs` and `olmv1 search catalog`, updated by `olmv1 update catalog`, deleted by `olmv1 delete catalog` |

**Resource short names for kubectl:**
```bash
kubectl get clusterextensions       # ClusterExtensions
kubectl get clustercatalogs         # ClusterCatalogs
```

---

## Contribution Guidance

For complete local development workflow, build instructions, testing procedures, and PR submission requirements, refer to **[README.md](./README.md)**.

### Development Environment
Project Requirements:
- **Go 1.23.4**: Minimum Go version required, specified in `go.mod`
- **container runtime** such as `docker` or `podman`
- **Kubernetes**: Compatible with Kubernetes 1.28+
- **OLM**:
   - **Operator Lifecycle Manager**: For the legacy OLM commands, this plugin requires Operator Lifecycle Manager (OLM) to be installed in your cluster
      - Install OLM: https://olm.operatorframework.io/docs/getting-started/
   - **Operator Controller**: For the OLMv1 commands, the plugin required operator-controller to be installed in your cluster
      - Install operator-controller: https://operator-framework.github.io/operator-controller/getting-started/olmv1_getting_started/#installation

### Tool Management
Tools are managed via **bingo** (`.bingo/Variables.mk`) for reproducible builds. All tools are version-pinned.

---

## Initial Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/operator-framework/kubectl-operator.git
   cd kubectl-operator
   ```

2. **Install dependencies:**
   Dependencies are vendored, so they're already present. To update:
   ```bash
   go mod download
   go mod vendor
   ```

## Common Workflows

### Quick reference

- Build: 
   - `make build`: build the kubectl-operator binary to a bin directory at the repository root
   - `make install`: build the kubectl-operator binary and install it to the `$GOPATH`/bin directory
- Unit tests: 
   - `make test`: run unit tests
- Dependency management:
   - `go mod tidy`: tidy project dependencies in go.mod
   - `go mod vendor`: sync project dependencies to local vendor directory
   - `make vet`: vet project dependencies
- Code quality:
   - `make fmt`: fix code formatting
   - `make lint`: run linter on codebase (uses golangci-lint)
      - `make fix-lint-issues`: fix common lint issues
- Release:
   - `make release`: run goreleaser to generate a snapshot

### Creating Commits

**Standard commit workflow:**

1. Ensure the following run without errors:
   - `make fmt`
   - `go mod tidy`
   - `go mod vendor`
   - `make vet`
   - `make lint`
   - `make build`
   - `make test`
2. Stage changes:
   ```bash
   git add <files>
   ```
3. Create commit with descriptive message:
   ```bash
   git commit -s -m "Short description of change"
   ```
4. Ensure branch is up to date with the remote main branch, rebase otherwise
   ```bash
   $ git fetch git@github.com:operator-framework/kubectl-operator.git main
   From github.com:operator-framework/kubectl-operator
   * branch            main       -> FETCH_HEAD
   $ git rebase FETCH_HEAD
   ```

**Commit message conventions:**
- Use conventional commit format
- Include context and reasoning
- Reference issues when applicable
- Signoff on commits. See [DCO](DCO) for Developer Certificate of Origin requirements.

### Creating Pull Requests

* Fork the repository and make changes to a branch
* Run `make test`, `make lint`, and `make vet`
* Update documentation if needed
* Squash commits to a single commit with a comprehensive summary of changes
* Push branch to forked repository
* Submit PR from fork to `operator-framework/kubectl-operator`, generally targeting `main` branch
* Ask contributors listed in `OWNERS` file for `Approved` and `lgtm` tags
* Wait for all checks on the PR to pass

---

## Development Conventions

### Code Style
- Use `gofmt` for formatting (enforced by `make build`)
- Run `go vet` to catch common mistakes (enforced by `make build`)
- Write comprehensive tests
- Document public APIs

**Naming conventions:**
- Packages: lowercase, single word when possible
- Exported types: PascalCase
- Unexported types: camelCase
- Interfaces: noun or adjective (e.g., `Reader`, `Runnable`)
- Functions: PascalCase (exported) or camelCase (unexported)

**File organization:**
- One primary type per file
- File name matches primary type (lowercase with underscores)
- Test files: `*_test.go`
- Interface definitions: often in separate files or with primary implementation

### Import Organization

**Enforced by `gci` linter with custom order:**

```go
import (
    // 1. Standard library
    "context"
    "fmt"

    // 2. Dot imports (rare, avoid if possible)

    // 3. Default third-party imports
    "github.com/spf13/cobra"

    // 4. Operator Framework packages
    "github.com/operator-framework/api/pkg/operators/v1alpha1"

    // 5. This repository (local module)
    "github.com/operator-framework/kubectl-operator/internal/pkg/action"
)
```

### Package Aliases

**Enforced by `importas` linter:**

```go
import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    apierrors "k8s.io/apimachinery/pkg/api/errors"
    apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
    utilruntime "k8s.io/apimachinery/pkg/util/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    bsemver "github.com/blang/semver/v4"

    // Kubernetes API groups: <group><version>
    corev1 "k8s.io/api/core/v1"
    rbacv1 "k8s.io/api/rbac/v1"
)
```

**Always use these exact aliases** - the linter will enforce them.

### Error Handling

**Standard patterns:**

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create subscription: %v", err)
}

// Use apierrors for Kubernetes errors
if apierrors.IsNotFound(err) {
    // Handle not found
}

// Check for specific error types
var statusErr *apierrors.StatusError
if errors.As(err, &statusErr) {
    // Handle status error
}
```

**Error messages:**
- Start with lowercase (will be wrapped)
- Provide context about what operation failed
- Include relevant identifiers (names, namespaces)

### Testing Conventions

**Use Ginkgo/Gomega:**

```go
var _ = Describe("OperatorInstall", func() {
    Context("when installing an operator", func() {
        It("should create a subscription", func() {
            // Test logic
            Expect(err).ToNot(HaveOccurred())
        })
    })
})
```

**Test file structure:**
- `*_suite_test.go` - Test suite setup
- `*_test.go` - Individual test files

**Use table-driven tests for multiple cases:**
```go
DescribeTable("version validation",
    func(version string, expected bool) {
        result := validateVersion(version)
        Expect(result).To(Equal(expected))
    },
    Entry("valid semver", "1.2.3", true),
    Entry("invalid semver", "invalid", false),
)
```

### Code Quality Standards

**Anti-Patterns to Avoid:**
1. Don't over-engineer solutions
2. Don't create premature abstractions
3. Don't bypass the Action pattern
4. Don't skip cleanup
5. Don't ignore context cancellation
5. Don't silence errors

**Best Practices:**

- **Security first**: Validate inputs at system boundaries
- **Simple solutions**: Choose minimum complexity for the task
- **Test coverage**: Add tests for new functionality
- **Error context**: Wrap errors with meaningful context
- **Idempotency**: Make operations safe to retry
- **Documentation**: Update docs when behavior changes

---

## Safety and Security

### Security Considerations

1. **Input Validation:**
   - Validate all user inputs (package names, versions, namespaces)
   - Sanitize inputs used in Kubernetes resource names
   - Validate catalog URLs and sources

2. **RBAC Awareness:**
   - Operations respect Kubernetes RBAC permissions
   - Fail gracefully when permissions are insufficient
   - Document required permissions for each operation

3. **Network Security:**
   - Port-forwarding uses secure connections
   - TLS certificates validated when accessing catalogd
   - No plaintext credential transmission

4. **Resource Cleanup:**
   - Always clean up resources on failure
   - Use context timeouts to prevent resource leaks
   - Track created resources for deletion

5. **gosec Linter:**
   - Enabled in CI to catch security issues
   - Address all gosec warnings before merging

### Safe Coding Practices

1. **Context Usage:**
   - Always pass `context.Context` to functions that may block
   - Respect context cancellation in loops
   - Use `context.WithTimeout` for operations with time limits

2. **Resource Management:**
   - Close HTTP response bodies
   - Clean up port-forwards
   - Use defer for cleanup when appropriate

3. **Concurrent Access:**
   - Avoid shared mutable state
   - Use proper synchronization when necessary

4. **Error Handling:**
   - Never ignore errors silently
   - Log errors appropriately
   - Provide actionable error messages to users

---

## Common Errors and Debugging

### Build Errors

**Issue:** `vendor/` directory out of sync
```
# Fix:
go mod tidy
go mod vendor
```

**Issue:** Linting failures
```
# Auto-fix where possible:
make lint-fix

# Then manually address remaining issues
```

### Testing Errors

**Issue:** Tests fail due to missing dependencies
```
# Ensure all test dependencies are present:
go mod download
go test ./...
```

**Issue:** Ginkgo tests not running
```
# Ensure ginkgo is available:
go install github.com/onsi/ginkgo/ginkgo
```

### Runtime Errors

**Issue:** `OLM is not installed`
- OLM must be installed in the cluster
- Install: https://olm.operatorframework.io/docs/getting-started/

**Issue:** Port-forwarding failures (OLMv1)
- Verify catalogd is running: `kubectl get pods -n olmv1-system`
- Check catalogd service exists
- Verify network connectivity

**Issue:** Timeout waiting for CSV
- Increase timeout with `--timeout` flag
- Check operator installation logs
- Verify InstallPlan status

### Debugging Techniques

1. **Enable verbose logging:**
   ```bash
   # Most commands support -v flag
   kubectl operator install <package> -v
   ```

2. **Check Kubernetes resources:**
   ```bash
   # OLMv0
   kubectl get subscriptions -n <namespace>
   kubectl get csv -n <namespace>
   kubectl get installplans -n <namespace>

   # OLMv1
   kubectl get clusterextensions
   kubectl get clustercatalogs
   ```

3. **Review resource events:**
   ```bash
   kubectl describe subscription <name> -n <namespace>
   kubectl describe csv <name> -n <namespace>
   ```

4. **Check operator logs:**
   ```bash
   kubectl logs -n olm deployment/catalog-operator
   kubectl logs -n olm deployment/olm-operator
   ```

5. **Examine action behavior:**
   - Add log statements in action implementations
   - Use the injectable `Logf` function in actions
   - Rebuild and test: `make build && ./bin/kubectl-operator ...`

---

## Quick Reference

### Common Commands

**Testing the plugin locally:**
```bash
# Build and install
make install

# Verify installation
kubectl operator --help
which kubectl-operator  # Should be in $(go env GOPATH)/bin
```

**OLMv0 operations:**
```bash
# List installed operators
kubectl operator list

# List available operators from catalogs
kubectl operator list-available

# Install an operator
kubectl operator install <package-name> --channel=<channel> --namespace=<namespace>

# Upgrade an operator
kubectl operator upgrade <package-name>

# Uninstall an operator
kubectl operator uninstall <package-name> --namespace=<namespace>

# Catalog management
kubectl operator catalog add <name> <source>
kubectl operator catalog list
kubectl operator catalog remove <name>
```

**OLMv1 operations:**
```bash
# List ClusterExtensions
kubectl operator olmv1 get extensions

# List ClusterCatalogs
kubectl operator olmv1 get catalogs

# Install an extension
kubectl operator olmv1 install extension <name> --package=<package>

# Search catalogs
kubectl operator olmv1 search catalog <catalog-name> --query=<search-term>

# Create a catalog
kubectl operator olmv1 create catalog <name> --image=<image-ref>

# Delete resources
kubectl operator olmv1 delete extension <name>
kubectl operator olmv1 delete catalog <name>
```

### Resource Short Names

**OLMv0 resources (managed by OLM):**
```bash
kubectl get csv              # ClusterServiceVersions
kubectl get sub              # Subscriptions
kubectl get ip               # InstallPlans
kubectl get catsrc           # CatalogSources
kubectl get og               # OperatorGroups
```

**OLMv1 resources:**
```bash
kubectl get clusterextensions       # ClusterExtensions
kubectl get clustercatalogs         # ClusterCatalogs
```

### Common Build Targets

```bash
make build           # Build the binary
make install         # Build and install to $GOPATH/bin
make test            # Run all tests
make lint            # Run linters
make lint-fix        # Auto-fix linting issues
make bingo-upgrade   # Upgrade tool dependencies
```

### File Locations

```bash
# Binary output
bin/kubectl-operator

# Main entry point
main.go

# OLMv0 actions
internal/pkg/action/

# OLMv1 actions
internal/pkg/v1/action/

# Command definitions
internal/cmd/

# Public API
pkg/action/

# Tests
internal/pkg/action/*_test.go
internal/pkg/v1/action/*_test.go
```

---

## Agent-Specific Tips

### When Reading Code

**Start with these files in order:**

1. **Command structure** - `internal/cmd/root.go`
   - Understand the CLI hierarchy
   - See global flags and configuration setup
   - Review how commands are registered

2. **Action pattern** - `pkg/action/config.go`
   - Understand the Configuration struct
   - See how Kubernetes clients are initialized
   - Review field ownership setup

3. **Simple action example** - `internal/pkg/action/catalog_list.go`
   - See a straightforward action implementation
   - Understand the Run() method pattern
   - Review error handling

4. **Complex action example** - `internal/pkg/action/operator_install.go`
   - See full operator installation flow
   - Understand wait/poll patterns
   - Review cleanup on failure logic

5. **OLMv1 differences** - `internal/pkg/v1/action/extension_install.go`
   - Compare with OLMv0 approach
   - Understand new resource types
   - See dry-run support

**Understanding the dual architecture:**
- OLMv0 code: `internal/pkg/action/`, `internal/cmd/catalog.go`, `internal/cmd/operator.go`
- OLMv1 code: `internal/pkg/v1/`, `internal/cmd/olmv1.go`
- NO code sharing between versions - they're completely separate

### When Making Changes

**Critical guidelines:**

1. **Always read existing code first**
   - NEVER propose changes to files you haven't read
   - Understand the full context before modifying
   - Look for similar existing implementations

2. **Follow the Action pattern**
   - Business logic goes in `internal/pkg/action/` or `internal/pkg/v1/action/`
   - Create new action structs with `Run(ctx context.Context)` method
   - CLI code in `internal/cmd/` should only handle flags and output

3. **Respect the dual architecture**
   - Don't try to share code between OLMv0 and OLMv1
   - Keep them completely separate
   - Each has different resource types and workflows

4. **Use the Configuration pattern**
   - Pass `action.Configuration` to all actions
   - Don't create separate Kubernetes clients
   - Respect namespace context from configuration

5. **Implement cleanup**
   - Track resources created during action execution
   - Implement cleanup on failure with configurable timeout
   - Use context cancellation for cleanup

6. **Test locally before submitting**
   - Build and install: `make install`
   - Test with real cluster: install OLM first
   - Try both success and failure scenarios
   - Verify cleanup works correctly

7. **Update tests**
   - Add Ginkgo/Gomega tests for new actions
   - Use table-driven tests for multiple scenarios
   - Test error conditions, not just success paths

8. **Run quality checks**
   - `make lint` before committing
   - `make test` to verify tests pass
   - Fix all linter errors (don't skip them)

### When Debugging

**Systematic debugging approach:**

1. **Start with error messages**
   - Read the full error message carefully
   - Look for wrapped error context
   - Check if it's a Kubernetes API error

2. **Check resource state**
   ```bash
   # For OLMv0 installs
   kubectl get subscriptions -n <ns>
   kubectl describe subscription <name> -n <ns>
   kubectl get installplans -n <ns>
   kubectl get csv -n <ns>

   # For OLMv1 installs
   kubectl get clusterextensions
   kubectl describe clusterextension <name>
   kubectl get clustercatalogs
   kubectl describe clustercatalog <name>
   ```

3. **Review OLM operator logs**
   ```bash
   # OLMv0 operators
   kubectl logs -n olm deployment/catalog-operator
   kubectl logs -n olm deployment/olm-operator

   # OLMv1 operators (if installed)
   kubectl logs -n olmv1-system deployment/catalogd-controller
   kubectl logs -n olmv1-system deployment/operator-controller
   ```

4. **Add debug logging**
   - Use the injectable `Logf` function in actions
   - Add temporary log statements to understand flow
   - Rebuild and test: `make build && ./bin/kubectl-operator ...`

5. **Use verbose flags**
   - Many commands support `-v` or `--verbose` flags
   - Enable to see detailed operation logs

6. **Test in isolation**
   - Create a minimal test case
   - Test one operation at a time
   - Eliminate variables systematically

### Code Quality Standards for AI Agents

**When generating or modifying code:**

- **Avoid over-engineering**: Only change what's necessary
- **No premature abstraction**: Don't create helpers for one-time use
- **Security first**: Validate at system boundaries
- **Delete unused code**: No backwards-compatibility hacks
- **Simple solutions**: Minimum complexity for the task
- **Explicit over clever**: Prefer readable code over clever tricks
- **Context awareness**: Always pass and respect context.Context
- **Proper cleanup**: Use defer and handle failures gracefully

**When in doubt:**
- Look for similar existing code in the repository
- Follow established patterns rather than inventing new ones
- Ask for clarification rather than guessing

---

## Reporting Back to User

This section provides guidelines for how AI agents should communicate results to users.

### When Completing Search Tasks

Provide:
1. Exact file paths with line numbers
2. Relevant code snippets
3. Context about what the code does
4. Related files user might need to check

### When Completing Code Modification Tasks

Report:
1. Files modified with brief description of changes
2. Make commands that need to be run
3. Test results if tests were run
4. Any remaining work or considerations

### When Encountering Issues

Report:
1. Exact error messages
2. What was attempted
3. Relevant logs or output
4. Suggestions for resolution or what to investigate

### Communication Best Practices

1. Be concise but complete
2. Be accurate - do not assume or guess details
3. Ask when uncertain, if requirements are ambiguous or multiple valid approaches are identified
4. Be clear about impact of changes

---

## Additional Resources

- **Project Documentation:** See [README.md](README.md)
- [OLMv0 - operator-lifecycle-manager](https://github.com/operator-framework/operator-lifecycle-manager)
- [OLMv1 - operator-controller](https://github.com/operator-framework/operator-controller)
- **OLM Documentation:** https://olm.operatorframework.io/
- **Cobra Documentation:** https://cobra.dev/
- **Kubernetes Client-Go:** https://github.com/kubernetes/client-go
- **Controller Runtime:** https://github.com/kubernetes-sigs/controller-runtime

---

**For questions or contributions, please open an issue or pull request on GitHub.**

---

This document should be updated as the project evolves. For specific implementation details, refer to the source code and inline documentation.
