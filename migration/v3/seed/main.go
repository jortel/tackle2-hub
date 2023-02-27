package seed

import (
	"encoding/json"
	"github.com/konveyor/tackle2-hub/model"
	"github.com/konveyor/tackle2-hub/settings"
	"gorm.io/gorm"
)

var Settings = &settings.Settings

const imgEAP = `
<svg id="fa78262c-b038-4be5-ad05-45590268c81b" data-name="Icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 36 36">
  <g>
    <path d="M27,23.37a.63.63,0,0,0-.62.63.63.63,0,0,0,1.25,0A.63.63,0,0,0,27,23.37Z"/>
    <path d="M18,9.38a.63.63,0,0,0-.62.63.63.63,0,0,0,1.25,0A.63.63,0,0,0,18,9.38Z"/>
    <path d="M20,9.38a.63.63,0,0,0-.62.63.63.63,0,0,0,1.25,0A.63.63,0,0,0,20,9.38Z"/>
    <g>
      <path d="M31,18.38H5a.62.62,0,0,0-.62.62V29a.62.62,0,0,0,.62.62H31a.62.62,0,0,0,.62-.62V19A.62.62,0,0,0,31,18.38Zm-.62,10H5.62V19.62H30.38Z"/>
      <path d="M9,24.62h6a.62.62,0,0,0,0-1.24H9a.62.62,0,0,0,0,1.24Z"/>
      <path d="M13,17.62H23a.62.62,0,0,0,.62-.62V7A.62.62,0,0,0,23,6.38H13a.62.62,0,0,0-.62.62V17A.62.62,0,0,0,13,17.62Zm.62-10h8.76v8.76H13.62Z"/>
    </g>
  </g>
</svg>
`
const imgCloud = `
<svg id="be70a7c5-0b8d-49cc-a04e-2075cdc15b11" data-name="Icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 36 36">
  <path d="M28.62,17.52v0a9.12,9.12,0,0,0-17.28-4.07,7.12,7.12,0,1,0-.84,14.19H28a.75.75,0,0,0,.26-.05,5.11,5.11,0,0,0,.36-10ZM27.5,26.38h-17a5.88,5.88,0,1,1,1.09-11.65.64.64,0,0,0,.69-.36,7.87,7.87,0,0,1,15.1,3.13c0,.13,0,.26,0,.4V18a.63.63,0,0,0,.56.66,3.87,3.87,0,0,1-.41,7.71Z"/>
</svg>
`
const imgMigration = `
<svg id="af27d7cd-0cec-48a8-afb1-973862026367" data-name="Icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 36 36">
  <g>
    <path d="M21,26.38a.61.61,0,0,0-.62.62v3.38H8.62V18.62H18a.62.62,0,0,0,0-1.24H8a.61.61,0,0,0-.62.62V31a.61.61,0,0,0,.62.62H21a.61.61,0,0,0,.62-.62V27A.61.61,0,0,0,21,26.38Z"/>
    <path d="M28,4.38H15a.61.61,0,0,0-.62.62V15a.62.62,0,0,0,1.24,0V5.62H27.38V17.38H24a.62.62,0,0,0,0,1.24h4a.61.61,0,0,0,.62-.62V5A.61.61,0,0,0,28,4.38Z"/>
    <path d="M23.56,15.44a.62.62,0,0,0,.88-.88l-3-3a.66.66,0,0,0-.88,0l-3,3c-.58.56.32,1.46.88.88l1.94-1.93V24a.62.62,0,0,0,1.24,0V13.51Z"/>
  </g>
</svg>
`
const imgMug = `
<svg id="a6896f88-91d8-4d59-ae46-7fd3a01d2680" data-name="Icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 36 36">
  <g>
    <path d="M27,30.38H4a.62.62,0,0,0,0,1.24H27A.62.62,0,0,0,27,30.38Z"/>
    <path d="M25.72,8V5a.63.63,0,0,0-.63-.62H5A.62.62,0,0,0,4.38,5V21A6.63,6.63,0,0,0,11,27.62h8.09A6.63,6.63,0,0,0,25.72,21C33.53,20.05,33.53,8.86,25.72,8ZM24.47,21a5.39,5.39,0,0,1-5.38,5.38H11A5.39,5.39,0,0,1,5.62,21V5.62H24.47Zm1.25-1.32V9.23A5.26,5.26,0,0,1,25.72,19.68Z"/>
  </g>
</svg>
`
const imgServer = `
<svg id="e84c18b7-6f4f-41b8-ba35-b8f1d6b56c9d" data-name="Icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 36 36">
  <path d="M27,18.62a.62.62,0,1,0-.62-.62A.61.61,0,0,0,27,18.62Z"/>
  <g>
    <path d="M31,12.38H5a.61.61,0,0,0-.62.62V23a.61.61,0,0,0,.62.62H31a.61.61,0,0,0,.62-.62V13A.61.61,0,0,0,31,12.38Zm-.62,10H5.62V13.62H30.38Z"/>
    <path d="M9,18.62h6a.62.62,0,0,0,0-1.24H9a.62.62,0,0,0,0,1.24Z"/>
  </g>
</svg>
`
const imgMultiply = `
<svg id="a7fb6d4d-772e-4cb1-945c-4f2228f674ca" data-name="Icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 36 36">
  <path d="M31.58,18.24a.63.63,0,0,0-.14-.68l-3-3c-.56-.58-1.46.32-.88.88l1.93,1.93h-10A8.59,8.59,0,0,0,23.62,10V8.51l1.94,1.93a.62.62,0,0,0,.88-.88l-3-3a.63.63,0,0,0-.68-.14,1.43,1.43,0,0,0-.2.14l-3,3c-.58.56.32,1.47.88.88l1.94-1.93c.46,4.7-2.47,8.76-7.36,8.86H11.56a3.62,3.62,0,1,0,0,1.25H15c4.9.09,7.84,4.17,7.38,8.87l-1.94-1.93c-.56-.58-1.46.32-.88.88l3,3a.63.63,0,0,0,.88,0l3-3c.58-.56-.32-1.47-.88-.88l-1.94,1.93V26a8.62,8.62,0,0,0-4.17-7.38h10l-1.93,1.94a.62.62,0,0,0,.88.88l3-3A.72.72,0,0,0,31.58,18.24ZM8,20.38a2.38,2.38,0,0,1,0-4.75A2.38,2.38,0,0,1,8,20.38Z"/>
</svg>
`
const imgVirt = `
<svg id="afd7eb3f-d4b0-41f4-9c4a-5a156398987f" data-name="Icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 36 36">
  <g>
    <path d="M27,25.37a.62.62,0,0,0-.62.63.63.63,0,0,0,.62.63.64.64,0,0,0,.63-.63A.63.63,0,0,0,27,25.37Z"/>
    <path d="M27,17.37a.62.62,0,0,0-.62.63.63.63,0,0,0,.62.63.64.64,0,0,0,.63-.63A.63.63,0,0,0,27,17.37Z"/>
    <path d="M27,9.37a.62.62,0,0,0-.62.63.63.63,0,0,0,.62.63.64.64,0,0,0,.63-.63A.63.63,0,0,0,27,9.37Z"/>
    <path d="M2.5,3.13a.63.63,0,1,0-.63-.62A.62.62,0,0,0,2.5,3.13Z"/>
    <path d="M2.49,32.88a.62.62,0,1,0,0,1.24.62.62,0,1,0,0-1.24Z"/>
    <path d="M33.5,32.87a.63.63,0,0,0,0,1.26.63.63,0,0,0,0-1.26Z"/>
    <path d="M33.49,3.12a.63.63,0,0,0,.63-.62.63.63,0,0,0-.63-.62.62.62,0,0,0-.62.62A.61.61,0,0,0,33.49,3.12Z"/>
    <path d="M31,4.38H5A.61.61,0,0,0,4.38,5V31a.61.61,0,0,0,.62.62H31a.61.61,0,0,0,.62-.62V5A.61.61,0,0,0,31,4.38Zm-.62,26H5.62V5.62H30.38Z"/>
    <path d="M9,10.62h6a.62.62,0,0,0,0-1.24H9a.62.62,0,0,0,0,1.24Z"/>
    <path d="M9,26.62h6a.62.62,0,0,0,0-1.24H9a.62.62,0,1,0,0,1.24Z"/>
    <path d="M9,18.62h6a.62.62,0,0,0,0-1.24H9a.62.62,0,0,0,0,1.24Z"/>
    <path d="M21.1,32.88a.62.62,0,1,0,0,1.24.62.62,0,1,0,0-1.24Z"/>
    <path d="M24.2,32.88a.62.62,0,1,0,.62.62A.61.61,0,0,0,24.2,32.88Z"/>
    <path d="M27.3,32.88a.62.62,0,1,0,0,1.24.62.62,0,0,0,0-1.24Z"/>
    <path d="M30.4,32.88a.62.62,0,0,0,0,1.24.62.62,0,1,0,0-1.24Z"/>
    <path d="M8.7,32.88a.62.62,0,1,0,.62.62A.61.61,0,0,0,8.7,32.88Z"/>
    <path d="M14.9,32.88a.62.62,0,1,0,0,1.24.62.62,0,1,0,0-1.24Z"/>
    <path d="M11.8,32.88a.62.62,0,1,0,0,1.24.62.62,0,1,0,0-1.24Z"/>
    <path d="M5.6,32.88a.62.62,0,0,0,0,1.24.62.62,0,1,0,0-1.24Z"/>
    <path d="M18,32.88a.62.62,0,1,0,0,1.24.62.62,0,0,0,0-1.24Z"/>
    <path d="M2.5,31a.63.63,0,0,0,.62-.63.63.63,0,0,0-1.25,0A.63.63,0,0,0,2.5,31Z"/>
    <path d="M2.5,21.72a.63.63,0,1,0-.63-.62A.62.62,0,0,0,2.5,21.72Z"/>
    <path d="M2.5,12.42a.63.63,0,1,0-.63-.62A.62.62,0,0,0,2.5,12.42Z"/>
    <path d="M2.5,15.52a.63.63,0,1,0-.63-.62A.62.62,0,0,0,2.5,15.52Z"/>
    <path d="M2.5,6.22a.63.63,0,1,0-.63-.62A.62.62,0,0,0,2.5,6.22Z"/>
    <path d="M2.5,24.82a.63.63,0,1,0-.63-.62A.62.62,0,0,0,2.5,24.82Z"/>
    <path d="M2.5,18.62A.63.63,0,1,0,1.87,18,.62.62,0,0,0,2.5,18.62Z"/>
    <path d="M2.5,27.92a.63.63,0,1,0-.63-.62A.62.62,0,0,0,2.5,27.92Z"/>
    <path d="M2.5,9.32a.63.63,0,1,0-.63-.62A.62.62,0,0,0,2.5,9.32Z"/>
    <path d="M11.8,3.12a.62.62,0,0,0,.62-.62.62.62,0,0,0-.62-.62.63.63,0,0,0-.63.62A.63.63,0,0,0,11.8,3.12Z"/>
    <path d="M14.9,3.12a.63.63,0,0,0,.63-.62.63.63,0,0,0-.63-.62.62.62,0,0,0-.62.62A.61.61,0,0,0,14.9,3.12Z"/>
    <path d="M18,3.12a.61.61,0,0,0,.62-.62A.62.62,0,0,0,18,1.88a.62.62,0,0,0-.62.62A.62.62,0,0,0,18,3.12Z"/>
    <path d="M5.6,3.12a.63.63,0,0,0,.63-.62.63.63,0,0,0-.63-.62A.62.62,0,0,0,5,2.5.62.62,0,0,0,5.6,3.12Z"/>
    <path d="M8.7,3.12a.61.61,0,0,0,.62-.62.62.62,0,1,0-1.24,0A.61.61,0,0,0,8.7,3.12Z"/>
    <path d="M21.1,3.12a.63.63,0,0,0,.63-.62.63.63,0,0,0-.63-.62.62.62,0,0,0-.62.62A.61.61,0,0,0,21.1,3.12Z"/>
    <path d="M27.3,3.12a.62.62,0,0,0,.62-.62.62.62,0,0,0-.62-.62.63.63,0,0,0-.63.62A.63.63,0,0,0,27.3,3.12Z"/>
    <path d="M30.4,3.12A.63.63,0,0,0,31,2.5a.63.63,0,0,0-.63-.62.62.62,0,0,0-.62.62A.61.61,0,0,0,30.4,3.12Z"/>
    <path d="M24.2,3.12a.61.61,0,0,0,.62-.62.62.62,0,0,0-1.24,0A.61.61,0,0,0,24.2,3.12Z"/>
    <path d="M33.5,26.67a.63.63,0,0,0,0,1.26.63.63,0,0,0,0-1.26Z"/>
    <path d="M33.5,5a.63.63,0,0,0,0,1.26A.63.63,0,0,0,33.5,5Z"/>
    <path d="M33.5,29.77a.63.63,0,0,0,0,1.26.63.63,0,0,0,0-1.26Z"/>
    <path d="M33.5,23.57a.62.62,0,0,0-.62.63.63.63,0,0,0,.62.63.64.64,0,0,0,.63-.63A.63.63,0,0,0,33.5,23.57Z"/>
    <path d="M33.5,11.17a.62.62,0,0,0-.62.63.63.63,0,0,0,.62.63.64.64,0,0,0,.63-.63A.63.63,0,0,0,33.5,11.17Z"/>
    <path d="M33.5,20.47a.62.62,0,0,0-.62.63.63.63,0,0,0,.62.63.64.64,0,0,0,.63-.63A.63.63,0,0,0,33.5,20.47Z"/>
    <path d="M33.5,8.07a.62.62,0,0,0-.62.63.63.63,0,0,0,.62.63.64.64,0,0,0,.63-.63A.63.63,0,0,0,33.5,8.07Z"/>
    <path d="M33.5,14.27a.63.63,0,0,0,0,1.26.63.63,0,1,0,0-1.26Z"/>
    <path d="M33.5,17.37a.63.63,0,0,0,0,1.26.63.63,0,0,0,0-1.26Z"/>
  </g>
</svg>
`

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
						Name:        "Boss EAP 7",
						Description: "Boss EAP 7",
						Metadata:    Target("eap7"),
					},
					{
						Name:        "Boss EAP 6",
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
						Name:     "Containerization",
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
						Name:     "Quarkus",
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
						Name:     "OpenJDK",
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
						Name:        "OpenJDK 11",
						Description: "OpenJDK 11",
						Metadata:    Target("openjdk11"),
					},
					{
						Name:        "OpenJDK 17",
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
						Name:     "Linux",
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
						Name:     "Jakarta",
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
						Name:     "Spring Boot",
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
						Name:     "Open Liberty",
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
						Name:     "Camel",
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
						Name:        "Azure App Service",
						Description: "Azure App Service",
						Metadata:    Target("azure-appservice"),
					},
					{
						Name:        "Azure Kubernetes Service",
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
						Name:        "Azure App Service",
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
