package config

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/lifesum/configsum/pkg/pg"
)

const (
	pgUserCreateSchema = `CREATE SCHEMA IF NOT EXISTS config`
	pgUserCreateTable  = `
		CREATE TABLE IF NOT EXISTS config.users(
			id TEXT NOT NULL PRIMARY KEY,
			user_id TEXT NOT NULL,
			base_id TEXT NOT NULL,
			rendered JSONB NOT NULL,
			rule_ids TEXT[],
			created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'utc')
		)`
	pgUserDropTable = `DROP TABLE IF EXISTS config.users CASCADE`

	pgUserInsert = `
		/* pgUserInsert*/
		INSERT INTO
			config.users(base_id, id, rendered, rule_ids, user_id) VALUES(
			:baseId,
			:id,
			:rendered,
			:ruleIds,
			:userId)`
	pgUserGetLatest = `
		/* pgUserGetLatest */
		SELECT
			id, user_id, base_id, rendered, rule_ids, created_at
		FROM
			config.users
		WHERE
			base_id = :baseId
			AND user_id = :userId
		ORDER BY
			created_at DESC
		LIMIT
			1`
)

type pgUserRepo struct {
	db *sqlx.DB
}

// NewPostgresUserRepo returns a Postgres backed UserRepo implementation.
func NewPostgresUserRepo(db *sqlx.DB) (UserRepo, error) {
	return &pgUserRepo{
		db: db,
	}, nil
}

func (r *pgUserRepo) Append(
	id, baseID, userID string,
	ruleIDs []string,
	render rendered,
) (UserConfig, error) {
	raw, err := json.Marshal(render)
	if err != nil {
		return UserConfig{}, fmt.Errorf("marashl rendered: %s", err)
	}

	_, err = r.db.NamedExec(pgUserInsert, map[string]interface{}{
		"baseId":   baseID,
		"id":       id,
		"rendered": raw,
		"ruleIds":  pq.StringArray(ruleIDs),
		"userId":   userID,
	})
	if err != nil {
		switch errors.Cause(pg.Wrap(err)) {
		case pg.ErrDuplicateKey:
			return UserConfig{}, errors.Wrap(ErrExists, "user config")
		case pg.ErrRelationNotFound:
			if err := r.Setup(); err != nil {
				return UserConfig{}, err
			}

			return r.Append(id, baseID, userID, ruleIDs, render)
		default:
			return UserConfig{}, fmt.Errorf("named exec: %s", err)
		}
	}

	return UserConfig{
		baseID:    baseID,
		id:        id,
		userID:    userID,
		rendered:  render,
		createdAt: time.Now(),
	}, nil
}

func (r *pgUserRepo) GetLatest(baseID, userID string) (UserConfig, error) {
	query, args, err := r.db.BindNamed(pgUserGetLatest, map[string]interface{}{
		"baseId": baseID,
		"userId": userID,
	})
	if err != nil {
		return UserConfig{}, fmt.Errorf("named query: %s", err)
	}

	raw := struct {
		BaseID    string         `db:"base_id"`
		ID        string         `db:"id"`
		Rendered  []byte         `db:"rendered"`
		RuleIDs   pq.StringArray `db:"rule_ids"`
		UserID    string         `db:"user_id"`
		CreatedAt time.Time      `db:"created_at"`
	}{}

	err = r.db.Get(&raw, query, args...)
	if err != nil {
		if pg.IsRelationNotFound(pg.Wrap(err)) {
			if err := r.Setup(); err != nil {
				return UserConfig{}, err
			}

			return r.GetLatest(baseID, userID)
		}

		if err == sql.ErrNoRows {
			return UserConfig{}, errors.Wrap(ErrNotFound, "get user config")
		}

		return UserConfig{}, fmt.Errorf("get: %s", err)
	}

	render := rendered{}

	if err := json.Unmarshal(raw.Rendered, &render); err != nil {
		return UserConfig{}, fmt.Errorf("rendered unmarshal: %s", err)
	}

	return UserConfig{
		baseID:    raw.BaseID,
		id:        raw.ID,
		rendered:  render,
		ruleIDs:   []string(raw.RuleIDs),
		userID:    raw.UserID,
		createdAt: raw.CreatedAt,
	}, nil
}

func (r *pgUserRepo) Setup() error {
	for _, q := range []string{
		pgUserCreateSchema,
		pgUserCreateTable,
	} {
		_, err := r.db.Exec(q)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *pgUserRepo) Teardown() error {
	for _, q := range []string{
		pgUserDropTable,
	} {
		_, err := r.db.Exec(q)
		if err != nil {
			return err
		}
	}

	return nil
}
