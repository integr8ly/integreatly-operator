### Example usage:

Setting up the detector:
```go
    //you would most likely have this code in your main.go
    dc, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
    if err != nil {
        panic("Could not create discovery client")
    }

    d, err := detector.NewAutoDetect(dc)
    if err != nil {
        panic("error creating autodetector: " + err.Error())
    }

    d.Start(5 * time.Second) //scan for new CRDs every 5 seconds
```

Triggering an action when a particular CRD shows up:
```go
    // and pass this detector instance to the `add` function of your operator's controller, where you could run:
    d.AddCRDTrigger(&package.CRD{
        TypeMeta: metav1.TypeMeta{
            Kind:       package.CrdKind,
            APIVersion: package.SchemeGroupVersion.String(),
        },
    }, func(crd runtime.Object) {
        // Do actions now that the package.CRD exists in the API, e.g begin watching it:
        c.Watch(&source.Kind{Type: &package.CRD{}}, &EnqueueForObject{})
    })
```

Triggering an action when any of multiple CRDs show up:
```go
    // and pass this detector instance to the `add` function of your operator's controller, where you could run:
    d.AddCRDsTrigger([]runtime.Object{
        &package.CRD{
            TypeMeta: metav1.TypeMeta{
                Kind:       package.CrdKind,
                APIVersion: package.SchemeGroupVersion.String(),
            },
        },
        &package.OtherCrd{
            TypeMeta: metav1.TypeMeta{
                Kind: package.OtherCrdKind:
                APIVersion: package.SchemeGroupVersion.String(),
            },
        },
    }, func(crd runtime.Object) {
        // Do actions now that the package.CRD exists in the API, e.g begin watching it:
        c.Watch(&source.Kind{Type: crd}, &EnqueueForObject{})
    })
```