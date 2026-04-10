package database

import (
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RunMigrations runs database migrations
func RunMigrations(db *gorm.DB, migrationsPath string, logger *zap.Logger) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// Configure connection settings for migrations to prevent SSL timeouts
	// Temporarily increase connection lifetime during migration
	originalMaxLifetime := 1 * time.Hour
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
	defer sqlDB.SetConnMaxLifetime(originalMaxLifetime)

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	// Get current version
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	if dirty {
		logger.Warn("database is in dirty state, forcing version", zap.Uint("version", version))
		if err := m.Force(int(version)); err != nil {
			return fmt.Errorf("failed to force version: %w", err)
		}
	}

	logger.Info("running database migrations", zap.Uint("current_version", version))

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Get new version
	newVersion, _, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get new migration version: %w", err)
	}

	if newVersion != version {
		logger.Info("migrations completed", zap.Uint("new_version", newVersion))
	} else {
		logger.Info("database is up to date")
	}

	// Verify critical tables exist
	if err := verifyTablesExist(db, logger); err != nil {
		logger.Error("database verification failed after migrations", zap.Error(err))
		return fmt.Errorf("database verification failed: %w", err)
	}

	return nil
}

// verifyTablesExist checks that critical tables exist in the database
func verifyTablesExist(db *gorm.DB, logger *zap.Logger) error {
	tables := []string{"solar_sites", "solar_devices", "solar_measurements"}
	
	for _, tableName := range tables {
		var exists bool
		err := db.Raw(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = $1
			)
		`, tableName).Scan(&exists).Error
		
		if err != nil {
			return fmt.Errorf("failed to check if %s table exists: %w", tableName, err)
		}
		
		if !exists {
			logger.Error("table does not exist despite migrations being marked as complete",
				zap.String("table", tableName),
				zap.String("hint", "database may be in inconsistent state - reset migrations with: DELETE FROM schema_migrations; then restart"))
			return fmt.Errorf("%s table does not exist - please reset migration state and restart", tableName)
		}
		
		logger.Debug("table verified", zap.String("table", tableName))
	}
	
	logger.Info("database verification passed", zap.Strings("tables", tables))
	return nil
}
