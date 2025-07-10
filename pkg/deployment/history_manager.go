package deployment

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// PostgresHistoryManager manages deployment history in PostgreSQL
type PostgresHistoryManager struct {
	db *sql.DB
}

// NewPostgresHistoryManager creates a new PostgreSQL history manager
func NewPostgresHistoryManager(connectionString string) (*PostgresHistoryManager, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	manager := &PostgresHistoryManager{db: db}

	// Create tables if they don't exist
	if err := manager.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return manager, nil
}

// createTables creates the necessary database tables
func (m *PostgresHistoryManager) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS deployments (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			version VARCHAR(255) NOT NULL,
			platform VARCHAR(50) NOT NULL,
			environment VARCHAR(50) NOT NULL,
			status VARCHAR(50) NOT NULL,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP,
			configuration JSONB,
			progress JSONB,
			health_checks JSONB,
			error TEXT,
			rollback_info JSONB,
			metadata JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS deployment_components (
			id SERIAL PRIMARY KEY,
			deployment_id VARCHAR(255) REFERENCES deployments(id),
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL,
			version VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP,
			health_checks JSONB,
			error TEXT,
			metadata JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS deployment_history (
			id SERIAL PRIMARY KEY,
			deployment_id VARCHAR(255) REFERENCES deployments(id),
			timestamp TIMESTAMP NOT NULL,
			event VARCHAR(100) NOT NULL,
			details JSONB,
			actor VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status)`,
		`CREATE INDEX IF NOT EXISTS idx_deployments_environment ON deployments(environment)`,
		`CREATE INDEX IF NOT EXISTS idx_deployments_platform ON deployments(platform)`,
		`CREATE INDEX IF NOT EXISTS idx_deployments_start_time ON deployments(start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_deployment_history_deployment_id ON deployment_history(deployment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_deployment_history_timestamp ON deployment_history(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_deployment_components_deployment_id ON deployment_components(deployment_id)`,
	}

	for _, query := range queries {
		if _, err := m.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// RecordDeployment records a new deployment
func (m *PostgresHistoryManager) RecordDeployment(deployment *Deployment) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Convert complex fields to JSON
	configJSON, _ := json.Marshal(deployment.Configuration)
	progressJSON, _ := json.Marshal(deployment.Progress)
	healthChecksJSON, _ := json.Marshal(deployment.HealthChecks)
	rollbackInfoJSON, _ := json.Marshal(deployment.RollbackInfo)
	metadataJSON, _ := json.Marshal(deployment.Metadata)

	// Insert or update deployment
	query := `
		INSERT INTO deployments (
			id, name, version, platform, environment, status, 
			start_time, end_time, configuration, progress, health_checks, 
			error, rollback_info, metadata, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			end_time = EXCLUDED.end_time,
			progress = EXCLUDED.progress,
			health_checks = EXCLUDED.health_checks,
			error = EXCLUDED.error,
			rollback_info = EXCLUDED.rollback_info,
			metadata = EXCLUDED.metadata,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err = tx.Exec(query,
		deployment.ID,
		deployment.Name,
		deployment.Version,
		deployment.Platform,
		deployment.Environment,
		deployment.Status,
		deployment.StartTime,
		deployment.EndTime,
		configJSON,
		progressJSON,
		healthChecksJSON,
		deployment.Error,
		rollbackInfoJSON,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to insert deployment: %w", err)
	}

	// Insert components
	for _, component := range deployment.Components {
		healthChecksJSON, _ := json.Marshal(component.HealthChecks)
		metadataJSON, _ := json.Marshal(component.Metadata)

		componentQuery := `
			INSERT INTO deployment_components (
				deployment_id, name, type, version, status,
				start_time, end_time, health_checks, error, metadata
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`

		_, err = tx.Exec(componentQuery,
			deployment.ID,
			component.Name,
			component.Type,
			component.Version,
			component.Status,
			component.StartTime,
			component.EndTime,
			healthChecksJSON,
			component.Error,
			metadataJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to insert component: %w", err)
		}
	}

	// Record deployment started event
	eventDetails := map[string]interface{}{
		"version":     deployment.Version,
		"platform":    deployment.Platform,
		"environment": deployment.Environment,
	}
	if err := m.recordEventTx(tx, deployment.ID, "deployment_started", eventDetails, ""); err != nil {
		return err
	}

	return tx.Commit()
}

// RecordEvent records a deployment event
func (m *PostgresHistoryManager) RecordEvent(deploymentID string, event string, details map[string]interface{}) error {
	return m.recordEventWithActor(deploymentID, event, details, "")
}

// recordEventWithActor records an event with an actor
func (m *PostgresHistoryManager) recordEventWithActor(deploymentID string, event string, details map[string]interface{}, actor string) error {
	detailsJSON, _ := json.Marshal(details)

	query := `
		INSERT INTO deployment_history (deployment_id, timestamp, event, details, actor)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := m.db.Exec(query, deploymentID, time.Now(), event, detailsJSON, actor)
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	return nil
}

// recordEventTx records an event within a transaction
func (m *PostgresHistoryManager) recordEventTx(tx *sql.Tx, deploymentID string, event string, details map[string]interface{}, actor string) error {
	detailsJSON, _ := json.Marshal(details)

	query := `
		INSERT INTO deployment_history (deployment_id, timestamp, event, details, actor)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := tx.Exec(query, deploymentID, time.Now(), event, detailsJSON, actor)
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	return nil
}

// GetHistory retrieves deployment history
func (m *PostgresHistoryManager) GetHistory(filters HistoryFilters) ([]DeploymentHistory, error) {
	query := `
		SELECT id, deployment_id, timestamp, event, details, actor
		FROM deployment_history
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	if filters.DeploymentID != "" {
		argCount++
		query += fmt.Sprintf(" AND deployment_id = $%d", argCount)
		args = append(args, filters.DeploymentID)
	}

	if filters.StartTime != nil {
		argCount++
		query += fmt.Sprintf(" AND timestamp >= $%d", argCount)
		args = append(args, *filters.StartTime)
	}

	if filters.EndTime != nil {
		argCount++
		query += fmt.Sprintf(" AND timestamp <= $%d", argCount)
		args = append(args, *filters.EndTime)
	}

	if filters.Event != "" {
		argCount++
		query += fmt.Sprintf(" AND event = $%d", argCount)
		args = append(args, filters.Event)
	}

	query += " ORDER BY timestamp DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filters.Limit)
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var history []DeploymentHistory
	for rows.Next() {
		var h DeploymentHistory
		var detailsJSON []byte
		var actor sql.NullString

		err := rows.Scan(&h.ID, &h.DeploymentID, &h.Timestamp, &h.Event, &detailsJSON, &actor)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if err := json.Unmarshal(detailsJSON, &h.Details); err != nil {
			return nil, fmt.Errorf("failed to unmarshal details: %w", err)
		}

		if actor.Valid {
			h.Actor = actor.String
		}

		history = append(history, h)
	}

	return history, nil
}

// GetDeployment retrieves a specific deployment
func (m *PostgresHistoryManager) GetDeployment(deploymentID string) (*Deployment, error) {
	query := `
		SELECT id, name, version, platform, environment, status,
			start_time, end_time, configuration, progress, health_checks,
			error, rollback_info, metadata
		FROM deployments
		WHERE id = $1
	`

	var deployment Deployment
	var endTime sql.NullTime
	var configJSON, progressJSON, healthChecksJSON, rollbackInfoJSON, metadataJSON []byte
	var errorStr sql.NullString

	err := m.db.QueryRow(query, deploymentID).Scan(
		&deployment.ID,
		&deployment.Name,
		&deployment.Version,
		&deployment.Platform,
		&deployment.Environment,
		&deployment.Status,
		&deployment.StartTime,
		&endTime,
		&configJSON,
		&progressJSON,
		&healthChecksJSON,
		&errorStr,
		&rollbackInfoJSON,
		&metadataJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("deployment not found: %s", deploymentID)
		}
		return nil, fmt.Errorf("failed to query deployment: %w", err)
	}

	// Handle nullable fields
	if endTime.Valid {
		deployment.EndTime = &endTime.Time
	}
	if errorStr.Valid {
		deployment.Error = errorStr.String
	}

	// Unmarshal JSON fields
	json.Unmarshal(configJSON, &deployment.Configuration)
	json.Unmarshal(progressJSON, &deployment.Progress)
	json.Unmarshal(healthChecksJSON, &deployment.HealthChecks)
	json.Unmarshal(rollbackInfoJSON, &deployment.RollbackInfo)
	json.Unmarshal(metadataJSON, &deployment.Metadata)

	// Load components
	components, err := m.getDeploymentComponents(deploymentID)
	if err != nil {
		return nil, err
	}
	deployment.Components = components

	return &deployment, nil
}

// GetDeployments retrieves multiple deployments
func (m *PostgresHistoryManager) GetDeployments(filters DeploymentFilters) ([]Deployment, error) {
	query := `
		SELECT id, name, version, platform, environment, status,
			start_time, end_time, configuration, progress, health_checks,
			error, rollback_info, metadata
		FROM deployments
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	if filters.Platform != "" {
		argCount++
		query += fmt.Sprintf(" AND platform = $%d", argCount)
		args = append(args, filters.Platform)
	}

	if filters.Environment != "" {
		argCount++
		query += fmt.Sprintf(" AND environment = $%d", argCount)
		args = append(args, filters.Environment)
	}

	if filters.Status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filters.Status)
	}

	if filters.StartTime != nil {
		argCount++
		query += fmt.Sprintf(" AND start_time >= $%d", argCount)
		args = append(args, *filters.StartTime)
	}

	if filters.EndTime != nil {
		argCount++
		query += fmt.Sprintf(" AND start_time <= $%d", argCount)
		args = append(args, *filters.EndTime)
	}

	query += " ORDER BY start_time DESC"

	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filters.Limit)
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query deployments: %w", err)
	}
	defer rows.Close()

	var deployments []Deployment
	for rows.Next() {
		var deployment Deployment
		var endTime sql.NullTime
		var configJSON, progressJSON, healthChecksJSON, rollbackInfoJSON, metadataJSON []byte
		var errorStr sql.NullString

		err := rows.Scan(
			&deployment.ID,
			&deployment.Name,
			&deployment.Version,
			&deployment.Platform,
			&deployment.Environment,
			&deployment.Status,
			&deployment.StartTime,
			&endTime,
			&configJSON,
			&progressJSON,
			&healthChecksJSON,
			&errorStr,
			&rollbackInfoJSON,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Handle nullable fields
		if endTime.Valid {
			deployment.EndTime = &endTime.Time
		}
		if errorStr.Valid {
			deployment.Error = errorStr.String
		}

		// Unmarshal JSON fields
		json.Unmarshal(configJSON, &deployment.Configuration)
		json.Unmarshal(progressJSON, &deployment.Progress)
		json.Unmarshal(healthChecksJSON, &deployment.HealthChecks)
		json.Unmarshal(rollbackInfoJSON, &deployment.RollbackInfo)
		json.Unmarshal(metadataJSON, &deployment.Metadata)

		deployments = append(deployments, deployment)
	}

	return deployments, nil
}

// getDeploymentComponents retrieves components for a deployment
func (m *PostgresHistoryManager) getDeploymentComponents(deploymentID string) ([]Component, error) {
	query := `
		SELECT name, type, version, status, start_time, end_time,
			health_checks, error, metadata
		FROM deployment_components
		WHERE deployment_id = $1
		ORDER BY start_time
	`

	rows, err := m.db.Query(query, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query components: %w", err)
	}
	defer rows.Close()

	var components []Component
	for rows.Next() {
		var component Component
		var endTime sql.NullTime
		var healthChecksJSON, metadataJSON []byte
		var errorStr sql.NullString

		err := rows.Scan(
			&component.Name,
			&component.Type,
			&component.Version,
			&component.Status,
			&component.StartTime,
			&endTime,
			&healthChecksJSON,
			&errorStr,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan component: %w", err)
		}

		// Handle nullable fields
		if endTime.Valid {
			component.EndTime = &endTime.Time
		}
		if errorStr.Valid {
			component.Error = errorStr.String
		}

		// Unmarshal JSON fields
		json.Unmarshal(healthChecksJSON, &component.HealthChecks)
		json.Unmarshal(metadataJSON, &component.Metadata)

		components = append(components, component)
	}

	return components, nil
}

// Close closes the database connection
func (m *PostgresHistoryManager) Close() error {
	return m.db.Close()
}