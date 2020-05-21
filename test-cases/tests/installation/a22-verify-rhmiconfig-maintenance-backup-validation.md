---
targets:
- 2.3.0
estimate: 30m
---

# A22 - Verify RHMIConfig maintenance and backup validation
We have a RHMIConfig validation webhook, this test is to verify the maintenance and backup validation works as expected

The expected value formats are:
```yaml
spec:
  backup: 
    applyOn : "HH:mm"
  maintenance: 
     applyFrom : "DDD HH:mm"
```

1. Add new values in correct format to ensure the validation works with no error

We require the expected values outlined above, we also expect these values to be parsed as a 1hour window
These windows can not overlap, the following steps we need to add a number of poorly formatted values and overlapping window values and ensure 
the validation webhook stops these values from being updated
2. Add poorly formatted values eg `"wefwef:wfwefwef", 12:111, sqef 12:13, 42:23` etc
4. Add overlapping values,
```yaml
spec:
  backup: 
    applyOn : "22:01"
  maintenance: 
     applyFrom : "Mon 22:20"
```

