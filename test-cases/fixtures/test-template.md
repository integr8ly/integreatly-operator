---
# See the metatadata section in the README.md for details on the
# allowed fields and values
automation:
  - INTLY-0000
components:
  - product-3scale
  - product-amq
environemnts:
  - osd-fresh-install
  - rhpds
estimate: 1h
targets:
  - 0.0.0
tags:
  - automated
  - destructive
---

# [ID] - [Title]

## Description [Optional]

[An arbitrary section that can be used list or describe content that will be referenced from the steps]

## Prerequisites [Optional]

[Describe here the prerequisites for the test, like the user that the tester should use, the environment the test should run against, if the requirements are other tests from the epic put them in the metadata section]

## Steps

[List here the the steps that the tester need to perform, the expected result of the step is quoted after the step]

1. [Step 1]

   > [Expected result]

2. [Step 2 without expected result]

3. [Step 3]

   ```bash
   some help cmd
   ```

   > [Expected result]
