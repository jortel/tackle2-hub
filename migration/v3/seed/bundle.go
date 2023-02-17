package seed

import (
	"github.com/konveyor/tackle2-hub/model"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"os"
)

//
// Image SVG
type Image struct {
	Name string `yaml:"name"`
	Content string `yaml:"content"`
}

//
// RuleSet seed object.
type RuleSet struct {
	Name string `yaml:"name"`
	Description string `yaml:"description"`
	Metadata Metadata `yaml:"metadata"`
}

func (r *RuleSet) model() (m *model.RuleSet) {
	return
}


//
// Metadata windup metadata.
type Metadata struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

//
// RuleBundle seed object.
type RuleBundle struct {
	Kind string `yaml:"kind"`
	Name string `yaml:"name"`
	Description string `yaml:"description"`
	Image string `yaml:"image"`
	RuleSets []RuleSet `yaml:"rulesets"`
}

//
// ImgMap image map.
type ImgMap map[string]string

func (r *RuleBundle) Create(db *gorm.DB, imgMap ImgMap) () {
	m := r.model()
	m.Image = &model.File{Name: r.Image + ".svg"}
	err := db.Create(r.Image).Error
	if err != nil {
		return
	}
	f, err := os.Create(m.Image.Path)
	if err != nil {
		return
	}
	defer func() {
		_ = f.Close()
	}()
	_, err = f.WriteString(imgMap[imgMap[r.Image]])
	if err != nil {
		return
	}
	_ = db.Create(m)
}

func (r *RuleBundle) model() (m *model.RuleBundle) {
	m = &model.RuleBundle{
		Kind: r.Kind,
		Name: r.Name,
		Description: r.Description,
		RuleSets: []model.RuleSet{},
	}
	for _, ruleset := range r.RuleSets {
		m.RuleSets = append(m.RuleSets, *ruleset.model())
	}
	return
}

//
// Bundles seed object.
type Bundles struct {
	Images []Image `yaml:"images"`
	Bundles []RuleBundle `yaml:"bundles"`
}

//
// Reconcile seeded bundles.
func (r *Bundles) Reconcile(db *gorm.DB, path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(b, r)
	if err != nil {
		return
	}
	imgMap := r.imgMap()
	for _, bundle := range r.Bundles {
		bundle.Create(db, imgMap)
	}
}

func (r *Bundles) create(db *gorm.DB) (err error) {
	return
}

func (r *Bundles) imgMap() (m map[string]string) {
	m = map[string]string{}
	for _, img := range r.Images {
		m[img.Name] = img.Content
	}
	return
}
