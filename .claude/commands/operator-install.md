---
description: Install a Kubernetes operator using kubectl operator olmv1
---

Help the user install a Kubernetes operator using the kubectl operator olmv1 install extension command.

## Instructions

1. **Read the install extension section from olmv1.md** (around line 274-420)
2. **Follow the documented workflow** from the "Recommended Workflows" section (around line 1260-1287)
3. **Ask the user for required information if not provided**:
   - Package name (use search if they don't know it)
   - Namespace to install into
   - ServiceAccount name (default: "default")

4. **Provide the complete workflow**:
   ```bash
   # Step 1: Search for the package
   kubectl operator olmv1 search catalog --package <package-name>

   # Step 2: Verify catalogs are serving
   kubectl operator olmv1 get catalog

   # Step 3: Install the extension
   kubectl operator olmv1 install extension <name> -n <namespace> -p <package-name>

   # Step 4: Verify installation
   kubectl operator olmv1 get extension <name>
   ```

5. **Include**:
   - Prerequisites from the docs
   - Expected outputs at each step
   - How to verify success (INSTALLED: True)
   - Common errors and how to fix them
   - Polling guidance (wait for INSTALLED: True)

Now help the user install their operator.
