---
description: Manage operator catalogs using kubectl operator olmv1
---

Help the user manage ClusterCatalogs using kubectl operator olmv1 catalog commands.

## Instructions

1. **Read the create catalog section from olmv1.md** (around line 61-258)
2. **Follow the catalog workflow** from "Recommended Workflows" (around line 1240-1258)
3. **Ask the user what they want to do**:
   - Create a new catalog
   - List existing catalogs
   - Update a catalog
   - Delete a catalog
   - Search a catalog for packages

4. **For catalog creation, provide the complete workflow**:
   ```bash
   # Step 1: Verify catalog doesn't exist
   kubectl operator olmv1 get catalog <catalog-name>

   # Step 2: Create catalog
   kubectl operator olmv1 create catalog <catalog-name> <image>

   # Step 3: Poll for serving status
   kubectl operator olmv1 get catalog <catalog-name>

   # Step 4: Verify packages available
   kubectl operator olmv1 search catalog --catalog <catalog-name>
   ```

5. **Include**:
   - Prerequisites (image accessibility, valid FBC)
   - Expected success message: `catalog "<name>" created`
   - Polling guidance (wait for SERVING: True)
   - How to check conditions with yq if catalog fails to serve
   - Common errors and fixes from the docs

Now help the user with their catalog operation.
