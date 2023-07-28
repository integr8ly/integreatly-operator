package common

import (
	"context"
	"github.com/integr8ly/integreatly-operator/pkg/products/obo"
	"k8s.io/apimachinery/pkg/labels"

	packageOperatorV1alpha1 "package-operator.run/apis/core/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestClusterObjectTemplateState(t TestingTB, ctx *TestingContext) {
	cotList := &packageOperatorV1alpha1.ClusterObjectTemplateList{}

	label, err := obo.GetOboClusterPackageLabel(ctx.Client)
	if err != nil {
		t.Errorf("Failed to get label", err)
	}

	parsedLabel, err := labels.Parse(label)
	if err != nil {
		t.Errorf("Failed to parse label", err)
	}

	opts := &client.ListOptions{
		LabelSelector: parsedLabel,
	}

	err = ctx.Client.List(context.TODO(), cotList, opts)
	if err != nil {
		t.Errorf("Failed to list ClusterObjectTemplates", err)
	}

	cotCount := 0
	activeCount := 0
	var problemCotList []string

	for _, cot := range cotList.Items {
		cotCount++
		if cot.Status.Phase == "Active" {
			activeCount++
		} else {
			problemCotList = append(problemCotList, cot.Name)
		}
	}

	if cotCount == 0 {
		t.Fatal("No ClusterObjectTemplates found")
	}

	if cotCount != activeCount {
		t.Log("ClusterObjectTemplates not in Active state")
		for _, i := range problemCotList {
			t.Logf("\t %s", i)
		}
		t.Fatalf("%d ClusterObjectTemplates are not in an active state", len(problemCotList))
	}

}
