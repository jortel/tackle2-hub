package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	liberr "github.com/konveyor/controller/pkg/error"
	"github.com/konveyor/controller/pkg/logging"
	"github.com/konveyor/tackle2-hub/api"
	"github.com/konveyor/tackle2-hub/auth"
	"github.com/konveyor/tackle2-hub/controller"
	"github.com/konveyor/tackle2-hub/importer"
	crd "github.com/konveyor/tackle2-hub/k8s/api"
	"github.com/konveyor/tackle2-hub/model"
	"github.com/konveyor/tackle2-hub/settings"
	"github.com/konveyor/tackle2-hub/tasking"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"io/ioutil"
	"k8s.io/client-go/kubernetes/scheme"
	"net/http"
	"os"
	"path"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	"strings"
	"syscall"
)

//
// DB constants
const (
	ConnectionString = "file:%s?_foreign_keys=yes"
)

var Settings = &settings.Settings

var log = logging.WithName("hub")

func init() {
	_ = Settings.Load()
}

//
// setupModels
func setupModels() (db *gorm.DB, err error) {
	db, err = open()
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	if seeded(db) {
		log.Info("Database already seeded, skipping.")
		return
	}
	var sqlDB *sql.DB
	sqlDB, err = db.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
	err = os.Remove(Settings.DB.Path)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	db, err = open()
	if err != nil {
		return
	}
	err = seed(db, model.All())
	if err != nil {
		err = liberr.Wrap(err)
		return
	}

	return
}

//
// open and migrate the DB.
func open() (db *gorm.DB, err error) {
	db, err = gorm.Open(
		sqlite.Open(fmt.Sprintf(ConnectionString, Settings.DB.Path)),
		&gorm.Config{
			NamingStrategy: &schema.NamingStrategy{
				SingularTable: true,
				NoLowerCase:   true,
			},
		})
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	err = db.AutoMigrate(append(model.All())...)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	return
}

//
// seeded returns if the DB has been seeded.
func seeded(db *gorm.DB) (seeded bool) {
	result := db.Find(&model.Setting{Key: ".hub.db.seeded"})
	return result.RowsAffected > 0
}

//
// buildScheme adds CRDs to the k8s scheme.
func buildScheme() (err error) {
	err = crd.AddToScheme(scheme.Scheme)
	return
}

//
// Seed the database with the contents of json
// files contained in DB_SEED_PATH.
func seed(db *gorm.DB, models []interface{}) (err error) {
	for _, m := range models {
		err = func() (err error) {
			kind := reflect.TypeOf(m).Name()
			fileName := strings.ToLower(kind) + ".json"
			filePath := path.Join(settings.Settings.DB.SeedPath, fileName)
			file, err := os.Open(filePath)
			if err != nil {
				err = nil
				return
			}
			defer file.Close()
			jsonBytes, err := ioutil.ReadAll(file)
			if err != nil {
				err = liberr.Wrap(err)
				return
			}

			var unmarshalled []map[string]interface{}
			err = json.Unmarshal(jsonBytes, &unmarshalled)
			if err != nil {
				err = liberr.Wrap(err)
				return
			}
			for i := range unmarshalled {
				result := db.Model(&m).Create(unmarshalled[i])
				if result.Error != nil {
					err = result.Error
					return
				}
			}
			return
		}()
		if err != nil {
			err = liberr.Wrap(err)
			return
		}
	}

	seeded, _ := json.Marshal(true)
	setting := model.Setting{Key: ".hub.db.seeded", Value: seeded}
	result := db.Create(&setting)
	if result.Error != nil {
		err = liberr.Wrap(result.Error)
		return
	}
	log.Info("Database seeded.")
	return
}

//
// addonManager
func addonManager() (mgr manager.Manager, err error) {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		_ = http.ListenAndServe(":2112", nil)
	}()
	cfg, err := config.GetConfig()
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	mgr, err = manager.New(
		cfg,
		manager.Options{
			MetricsBindAddress: Settings.Metrics.Address(),
		})
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	err = controller.Add(mgr)
	if err != nil {
		err = liberr.Wrap(err)
		return
	}
	return
}

//
// main.
func main() {
	syscall.Umask(0)
	log.Info("Started", "settings", Settings)
	var err error
	defer func() {
		if err != nil {
			log.Trace(err)
		}
	}()
	//
	// Addon controller
	err = buildScheme()
	if err != nil {
		return
	}
	addonManager, err := addonManager()
	if err != nil {
		return
	}
	go func() {
		err = addonManager.Start(signals.SetupSignalHandler())
		if err != nil {
			err = liberr.Wrap(err)
			return
		}
	}()
	//
	// Models
	db, err := setupModels()
	if err != nil {
		panic(err)
	}
	//
	// Auth provider.
	var provider auth.Provider
	if settings.Settings.Auth.Required {
		k := auth.NewKeycloak(
			settings.Settings.Auth.Keycloak.Host,
			settings.Settings.Auth.Keycloak.Realm,
			settings.Settings.Auth.Keycloak.ClientID,
			settings.Settings.Auth.Keycloak.ClientSecret)
		provider = &k
	} else {
		provider = &auth.NoAuth{}
	}
	//
	// Webserver API.
	router := gin.Default()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	for _, h := range api.All() {
		h.With(db, addonManager.GetClient(), provider)
		h.AddRoutes(router)
	}
	//
	// Task manager.
	taskManager := tasking.Manager{
		Client: addonManager.GetClient(),
		DB:     db,
	}
	taskManager.Run(context.Background())
	importManager := importer.Manager{
		DB: db,
	}
	//
	// Application import.
	importManager.Run(context.Background())
	err = router.Run()
}
