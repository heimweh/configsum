package client

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/lifesum/configsum/pkg/errors"
	"github.com/lifesum/configsum/pkg/pg"
)

const (
	pgCreateSchema = `CREATE SCHEMA IF NOT EXISTS client`

	pgClientCreateTable = `
		CREATE TABLE IF NOT EXISTS client.clients(
			id TEXT NOT NULL PRIMARY KEY,
			deleted BOOL DEFAULT FALSE,
			name TEXT NOT NULL UNIQUE,
			created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'utc')
		)`
	pgClientDropTable = `DROP TABLE IF EXISTS client.clients CASCADE`
	pgClientList      = `
		/* pgClientList */
		SELECT
			created_at, deleted, id, name
		FROM
			client.clients
		WHERE
			deleted = :deleted
		ORDER BY
			created_at DESC`
	pgClientLookup = `
		/* pgClientLookup */
		SELECT
			deleted, id, name, created_at
		FROM
			client.clients
		WHERE
			deleted = :deleted
			AND id = :id
		LIMIT
			1`
	pgClientStore = `
		/* pgClientStore */
		INSERT INTO
			client.clients(
				deleted,
				id,
				name
			)
			VALUES(
				:deleted,
				:id,
				:name
			)`

	pgTokenCreateTable = `
		CREATE TABLE IF NOT EXISTS client.tokens(
			secret TEXT NOT NULL PRIMARY KEY,
			deleted BOOL DEFAULT FALSE,
			client_id TEXT NOT NULL,
			created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'utc')
		)`
	pgTokenDropTable = `DROP TABLE IF EXISTS client.tokens CASCADE`
	pgTokenGetLatest = `
		/* pgTokenGetLatest */
		SELECT
			client_id, deleted, secret, created_at
		FROM
			client.tokens
		WHERE
			client_id = :id
			AND deleted = :deleted
		ORDER BY
			created_at DESC
		LIMIT
			1`
	pgTokenLookup = `
		/* pgTokenLookup */
		SELECT
			client_id, deleted, secret, created_at
		FROM
			client.tokens
		WHERE
			deleted = :deleted
			AND secret = :secret
		LIMIT
			1`
	pgTokenStore = `
		/* pgClientStore */
		INSERT INTO
			client.tokens(
				client_id,
				secret
			)
			VALUES(
				:clientId,
				:secret
			)`
)

type pgRepo struct {
	db *sqlx.DB
}

// NewPostgresRepo returns a Postgres backed Repo implementation.
func NewPostgresRepo(db *sqlx.DB) Repo {
	return &pgRepo{
		db: db,
	}
}

func (r *pgRepo) List() (List, error) {
	rows, err := r.db.NamedQuery(pgClientList, map[string]interface{}{
		"deleted": false,
	})
	if err != nil {
		switch errors.Cause(pg.Wrap(err)) {
		case pg.ErrRelationNotFound:
			if err := r.setup(); err != nil {
				return nil, err
			}

			return r.List()
		default:
			return nil, errors.Wrap(err, "List query")
		}
	}

	cs := List{}

	for rows.Next() {
		c := Client{}

		err := rows.Scan(
			&c.createdAt,
			&c.deleted,
			&c.id,
			&c.name,
		)
		if err != nil {
			return nil, errors.Wrap(err, "List scan")
		}

		cs = append(cs, c)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "List rows")
	}

	return cs, nil
}

func (r *pgRepo) Lookup(id string) (Client, error) {
	query, args, err := r.db.BindNamed(pgClientLookup, map[string]interface{}{
		"deleted": false,
		"id":      id,
	})
	if err != nil {
		return Client{}, errors.Wrap(err, "bind named")
	}

	raw := struct {
		Deleted   bool      `db:"deleted"`
		ID        string    `db:"id"`
		Name      string    `db:"name"`
		CreatedAt time.Time `db:"created_at"`
	}{}

	err = r.db.Get(&raw, query, args...)
	if err != nil {
		switch errors.Cause(pg.Wrap(err)) {
		case pg.ErrRelationNotFound:
			if err := r.setup(); err != nil {
				return Client{}, err
			}

			return r.Lookup(id)
		case sql.ErrNoRows:
			return Client{}, errors.Wrap(errors.ErrNotFound, "client lookup")
		default:
			return Client{}, errors.Wrap(err, "client lookup")
		}
	}

	return Client{
		deleted:   raw.Deleted,
		id:        raw.ID,
		name:      raw.Name,
		createdAt: raw.CreatedAt,
	}, nil
}

func (r *pgRepo) Store(id, name string) (Client, error) {
	_, err := r.db.NamedExec(pgClientStore, map[string]interface{}{
		"deleted": false,
		"id":      id,
		"name":    name,
	})
	if err != nil {
		switch errors.Cause(pg.Wrap(err)) {
		case pg.ErrDuplicateKey:
			return Client{}, errors.Wrap(errors.ErrExists, "client")
		case pg.ErrRelationNotFound:
			if err := r.setup(); err != nil {
				return Client{}, err
			}

			return r.Store(id, name)
		default:
			return Client{}, errors.Wrap(err, "named exec")
		}
	}

	return Client{
		id:        id,
		name:      name,
		createdAt: time.Now(),
	}, nil
}

func (r *pgRepo) setup() error {
	for _, q := range []string{
		pgCreateSchema,
		pgClientCreateTable,
	} {
		_, err := r.db.Exec(q)
		if err != nil {
			return errors.Wrap(err, "pgRepo.setup()")
		}
	}

	return nil
}

func (r *pgRepo) teardown() error {
	for _, q := range []string{
		pgClientDropTable,
	} {
		_, err := r.db.Exec(q)
		if err != nil {
			return errors.Wrap(err, "pgRepo.teardowm()")
		}
	}

	return nil
}

type pgTokenRepo struct {
	db *sqlx.DB
}

// NewPostgresTokenRepo returns a Postgres backed TokenRepo implementation.
func NewPostgresTokenRepo(db *sqlx.DB) TokenRepo {
	return &pgTokenRepo{
		db: db,
	}
}

func (r *pgTokenRepo) GetLatest(clientID string) (Token, error) {
	query, args, err := r.db.BindNamed(pgTokenGetLatest, map[string]interface{}{
		"id":      clientID,
		"deleted": false,
	})
	if err != nil {
		return Token{}, errors.Wrap(err, "bind named")
	}

	raw := struct {
		ClientID  string    `db:"client_id"`
		Deleted   bool      `db:"deleted"`
		Secret    string    `db:"secret"`
		CreatedAt time.Time `db:"created_at"`
	}{}

	err = r.db.Get(&raw, query, args...)
	if err != nil {
		switch errors.Cause(pg.Wrap(err)) {
		case pg.ErrRelationNotFound:
			if err := r.setup(); err != nil {
				return Token{}, err
			}

			return r.GetLatest(clientID)
		case sql.ErrNoRows:
			return Token{}, errors.Wrap(errors.ErrNotFound, "token lookup")
		default:
			return Token{}, errors.Wrap(err, "token lookup")
		}
	}

	return Token{
		clientID:  raw.ClientID,
		deleted:   raw.Deleted,
		secret:    raw.Secret,
		createdAt: raw.CreatedAt,
	}, nil
}

func (r *pgTokenRepo) Lookup(secret string) (Token, error) {
	query, args, err := r.db.BindNamed(pgTokenLookup, map[string]interface{}{
		"deleted": false,
		"secret":  secret,
	})
	if err != nil {
		return Token{}, errors.Wrap(err, "bind named")
	}

	raw := struct {
		ClientID  string    `db:"client_id"`
		Deleted   bool      `db:"deleted"`
		Secret    string    `db:"secret"`
		CreatedAt time.Time `db:"created_at"`
	}{}

	err = r.db.Get(&raw, query, args...)
	if err != nil {
		switch errors.Cause(pg.Wrap(err)) {
		case pg.ErrRelationNotFound:
			if err := r.setup(); err != nil {
				return Token{}, err
			}

			return r.Lookup(secret)
		case sql.ErrNoRows:
			return Token{}, errors.Wrap(errors.ErrNotFound, "token lookup")
		default:
			return Token{}, errors.Wrap(err, "token lookup")
		}
	}

	return Token{
		clientID:  raw.ClientID,
		deleted:   raw.Deleted,
		secret:    raw.Secret,
		createdAt: raw.CreatedAt,
	}, nil
}

func (r *pgTokenRepo) Store(clientID, secret string) (Token, error) {
	_, err := r.db.NamedExec(pgTokenStore, map[string]interface{}{
		"clientId": clientID,
		"secret":   secret,
	})
	if err != nil {
		switch errors.Cause(pg.Wrap(err)) {
		case pg.ErrRelationNotFound:
			if err := r.setup(); err != nil {
				return Token{}, err
			}

			return r.Store(clientID, secret)
		default:
			return Token{}, errors.Wrap(err, "named exec")
		}
	}

	return Token{
		clientID:  clientID,
		deleted:   false,
		secret:    secret,
		createdAt: time.Now(),
	}, nil
}

func (r *pgTokenRepo) setup() error {
	for _, q := range []string{
		pgCreateSchema,
		pgTokenCreateTable,
	} {
		_, err := r.db.Exec(q)
		if err != nil {
			return errors.Wrap(err, "pgTokenRepo.setup()")
		}
	}

	return nil
}

func (r *pgTokenRepo) teardown() error {
	for _, q := range []string{
		pgTokenDropTable,
	} {
		_, err := r.db.Exec(q)
		if err != nil {
			return errors.Wrap(err, "pgTokenRepo.teardown()")
		}
	}

	return nil
}
