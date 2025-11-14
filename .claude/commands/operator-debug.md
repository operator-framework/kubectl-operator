---
description: Troubleshoot kubectl operator issues
---

Help the user debug and troubleshoot kubectl operator issues.

## Instructions

1. **Read olmv1.md** to understand all documented error scenarios
2. **Ask the user to provide**:
   - The command they ran
   - The error message they received
   - Output of relevant `get` commands

3. **Follow this diagnostic process**:

   **For extension installation failures**:
   ```bash
   # Check extension status
   kubectl operator olmv1 get extension <name>

   # Check detailed conditions
   kubectl operator olmv1 get extension <name> -o yaml | yq eval '.status.conditions'

   # Check catalogs are serving
   kubectl operator olmv1 get catalog

   # Search for the package
   kubectl operator olmv1 search catalog --package <package-name>
   ```

   **For catalog creation failures**:
   ```bash
   # Check catalog status
   kubectl operator olmv1 get catalog <name>

   # Check detailed conditions
   kubectl operator olmv1 get catalog <name> -o yaml | yq eval '.status.conditions[] | select(.type=="Progressing")'
   ```

4. **Match error messages to documented scenarios**:
   - Read the "Common errors" sections in olmv1.md
   - For install extension errors: see lines ~380-420
   - For create catalog errors: see lines ~237-255

5. **Provide**:
   - Root cause analysis
   - Specific fix from the documentation
   - Verification commands
   - Prevention tips

Now help the user debug their issue.
