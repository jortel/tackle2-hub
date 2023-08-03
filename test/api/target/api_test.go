package target

import (
	"testing"

	"github.com/konveyor/tackle2-hub/api"
	"github.com/konveyor/tackle2-hub/test/assert"
)

func TestTargetCRUD(t *testing.T) {
	for _, r := range Samples {
		t.Run(r.Name, func(t *testing.T) {
			// Image.
			image, err := RichClient.File.Put(r.Image.Name)
			if err != nil {
				t.Errorf(err.Error())
			}
			r.Image.ID = image.ID
			// RuleSet
			if r.RuleSet != nil {
				ruleFiles := []api.File{}
				rules := []api.Rule{}
				for _, rule := range r.RuleSet.Rules {
					ruleFile, err := RichClient.File.Put(rule.File.Name)
					assert.Should(t, err)
					rules = append(rules, api.Rule{
						File: &api.Ref{
							ID: ruleFile.ID,
						},
					})
					ruleFiles = append(ruleFiles, *ruleFile)
				}
				r.RuleSet.Rules = rules
			}

			// Create.
			err = Target.Create(&r)
			if err != nil {
				t.Errorf(err.Error())
			}

			// Get.
			got, err := Target.Get(r.ID)
			if err != nil {
				t.Errorf(err.Error())
			}
			if assert.FlatEqual(got, r) {
				t.Errorf("Different response error. Got %v, expected %v", got, r)
			}

			// Update.
			r.Name = "Updated " + r.Name
			err = Target.Update(&r)
			if err != nil {
				t.Errorf(err.Error())
			}

			got, err = Target.Get(r.ID)
			if err != nil {
				t.Errorf(err.Error())
			}
			if got.Name != r.Name {
				t.Errorf("Different response error. Got %s, expected %s", got.Name, r.Name)
			}

			// Delete.
			err = Target.Delete(r.ID)
			if err != nil {
				t.Errorf(err.Error())
			}

			_, err = Target.Get(r.ID)
			if err == nil {
				t.Errorf("Resource exits, but should be deleted: %v", r)
			}
			if r.RuleSet != nil {
				_, err = RuleSet.Get(r.RuleSet.ID)
				if err == nil {
					t.Errorf("Resource exits, but should be deleted: %v", r)
				}
			}
		})
	}
}
