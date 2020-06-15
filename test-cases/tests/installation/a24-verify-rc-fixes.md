---
---

# A24 - Verify RC fixes

Note. This only applies to subsequent RCs. Changes made to the first release cut should be handled by separate test-cases

## Steps

1. Go to Jira and search for the bugs that were fixed in currently tested RC (update `fixVersion` and `Fix Build` to match currently tested version)
   - e.g. `project = INTLY && fixVersion = 2.x.x && "Fix Build" = RCx && type = Bug`
2. Reverify all issues found by the search above
