package seed

import (
	"encoding/json"
	"github.com/konveyor/tackle2-hub/model"
	"github.com/konveyor/tackle2-hub/settings"
	"gorm.io/gorm"
)

var Settings = &settings.Settings



//
// Seed the database with models.
func Seed(db *gorm.DB) {
	settings := []model.Setting{
		{Key: "review.assessment.required", Value: []byte("true")},
		{Key: "download.html.enabled", Value: []byte("true")},
		{Key: "download.csv.enabled", Value: []byte("true")},
	}
	_ = db.Create(settings)

	var Bundles = []RuleBundle{
		{
			image: imgEAP,
			RuleBundle: model.RuleBundle{
				Kind: "category",
				Name: "Application server migration to",
				Description: "Upgrade to the latest Release of JBoss EAP or migrate your applications" +
					" to JBoss EAP from other Enterprise Application Server (e.g. Oracle WebLogic Server).",
				RuleSets: []model.RuleSet{
					{
						Name: "Boss EAP 7",
						Description: "Boss EAP 7",
						Metadata:    Target("eap7"),
					},
					{
						Name: "Boss EAP 6",
						Description: "Boss EAP 6",
						Metadata:    Target("eap6"),
					},
				},
			},
		},
		{
			image: imgCloud,
			RuleBundle: model.RuleBundle{
				Name: "Containerization",
				Description: "A comprehensive set of cloud and container readiness rules to" +
					" assess applications for suitability for deployment on Kubernetes.",
				RuleSets: []model.RuleSet{
					{
						Name: "Containerization",
						Metadata: Target("cloud-readiness"),
					},
				},
			},
		},
		{
			image: imgMigration,
			RuleBundle: model.RuleBundle{
				Name:        "Quarkus",
				Description: "Rules to support the migration of Spring Boot applications to Quarkus.",
				RuleSets: []model.RuleSet{
					{
						Name: "Quarkus",
						Metadata: Target("quarkus"),
					},
				},
			},
		},
		{
			image: imgMug,
			RuleBundle: model.RuleBundle{
				Name:        "OracleJDK to OpenJDK",
				Description: "Rules to support the migration to OpenJDK from OracleJDK.",
				RuleSets: []model.RuleSet{
					{
						Name: "OpenJDK",
						Metadata: Target("openjdk"),
					},
				},
			},
		},
		{
			image: imgMug,
			RuleBundle: model.RuleBundle{
				Kind:        "category",
				Name:        "OpenJDK",
				Description: "Rules to support upgrading the version of OpenJDK. Migrate to OpenJDK 11 or OpenJDK 17.",
				RuleSets: []model.RuleSet{
					{
						Name: "OpenJDK 11",
						Description: "OpenJDK 11",
						Metadata:    Target("openjdk11"),
					},
					{
						Name: "OpenJDK 17",
						Description: "OpenJDK 17",
						Metadata:    Target("openjdk17"),
					},
				},
			},
		},
		{
			image: imgServer,
			RuleBundle: model.RuleBundle{
				Name:        "Linux",
				Description: "Ensure there are no Microsoft Windows paths hard coded into your applications.",
				RuleSets: []model.RuleSet{
					{
						Name: "Linux",
						Metadata: Target("linux"),
					},
				},
			},
		},
		{
			image: imgMigration,
			RuleBundle: model.RuleBundle{
				Name: "Jakarta EE 9",
				Description: "A collection of rules to support migrating applications from" +
					" Java EE 8 to Jakarta EE 9. The rules cover project dependencies, package" +
					" renaming, updating XML Schema namespaces, the renaming of application" +
					" configuration properties and bootstraping files.",
				RuleSets: []model.RuleSet{
					{
						Name: "Jakarta",
						Metadata: Target("jakarta-ee"),
					},
				},
			},
		},
		{
			image: imgMigration,
			RuleBundle: model.RuleBundle{
				Name: "Spring Boot on Red Hat Runtimes",
				Description: "A set of rules for assessing the compatibility of applications" +
					" against the versions of Spring Boot libraries supported by Red Hat Runtimes.",
				RuleSets: []model.RuleSet{
					{
						Name: "Spring Boot",
						Metadata: Target("rhr"),
					},
				},
			},
		},
		{
			image: imgMigration,
			RuleBundle: model.RuleBundle{
				Name: "Open Liberty",
				Description: "A comprehensive set of rulesfor migrating traditional WebSphere" +
					" applications to Open Liberty.",
				RuleSets: []model.RuleSet{
					{
						Name: "Open Liberty",
						Metadata: Target("openliberty"),
					},
				},
			},
		},
		{
			image: imgMultiply,
			RuleBundle: model.RuleBundle{
				Name:        "Camel",
				Description: "A comprehensive set of rules for migration from Apache Camel 2 to Apache Camel 3.",
				RuleSets: []model.RuleSet{
					{
						Name: "Camel",
						Metadata: Target("camel"),
					},
				},
			},
		},
		{
			image:    imgVirt,
			excluded: Settings.Product,
			RuleBundle: model.RuleBundle{
				Kind:        "category",
				Name:        "Azure",
				Description: "Upgrade your Java application so it can be deployed in different flavors of Azure.",
				RuleSets: []model.RuleSet{
					{
						Name:  "Azure App Service",
						Description: "Azure App Service",
						Metadata:    Target("azure-appservice"),
					},
					{
						Name: "Azure Kubernetes Service",
						Description: "Azure Kubernetes Service",
						Metadata:    Target("azure-aks"),
					},
				},
			},
		},
		{
			image:    imgVirt,
			excluded: !Settings.Product,
			RuleBundle: model.RuleBundle{
				Name:        "Azure",
				Description: "Upgrade your Java application so it can be deployed in different flavors of Azure.",
				RuleSets: []model.RuleSet{
					{
						Name:  "Azure App Service",
						Description: "Azure App Service",
						Metadata:    Target("azure-appservice"),
					},
				},
			},
		},
	}
	order := []uint{}
	for _, b := range Bundles {
		if !b.excluded {
			b.Create(db)
			order = append(order, b.ID)
		}
	}
	setting := &model.Setting{Key: "ui.bundle.order"}
	setting.Value, _ = json.Marshal(order)
	_ = db.Create(setting)

	return
}
