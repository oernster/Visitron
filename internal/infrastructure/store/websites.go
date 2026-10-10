package store

import (
	"database/sql"
	"errors"
	"strings"

	"visitron/internal/application"
	"visitron/internal/domain"
)

// Websites answers every website with its chosen repos, in the order added.
func (s *Store) Websites() ([]application.Website, error) {
	rows, err := s.db.Query(`SELECT id, host, path FROM websites ORDER BY id`)
	if err != nil {
		return nil, err
	}
	var sites []application.Website
	for rows.Next() {
		var w application.Website
		if err := rows.Scan(&w.ID, &w.Address.Host, &w.Address.Path); err != nil {
			_ = rows.Close()
			return nil, err
		}
		sites = append(sites, w)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return nil, err
	}
	for i := range sites {
		repos, err := s.repos(sites[i].ID)
		if err != nil {
			return nil, err
		}
		sites[i].Repos = repos
	}
	return sites, nil
}

func (s *Store) repos(id int64) ([]domain.Repo, error) {
	rows, err := s.db.Query(`SELECT owner, name FROM website_repos WHERE website_id = ? ORDER BY position`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var repos []domain.Repo
	for rows.Next() {
		var r domain.Repo
		if err := rows.Scan(&r.Owner, &r.Name); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

// AddWebsite records a website, answering its id.
func (s *Store) AddWebsite(w application.Website) (int64, error) {
	var id int64
	err := s.inTransaction(func(tx *sql.Tx) error {
		res, err := tx.Exec(`INSERT INTO websites (host, path) VALUES (?, ?)`, w.Address.Host, w.Address.Path)
		if err != nil {
			return err
		}
		if id, err = res.LastInsertId(); err != nil {
			return err
		}
		return writeRepos(tx, id, w.Repos)
	})
	return id, err
}

// UpdateWebsite replaces a website's address and chosen repos.
func (s *Store) UpdateWebsite(w application.Website) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		res, err := tx.Exec(`UPDATE websites SET host = ?, path = ? WHERE id = ?`, w.Address.Host, w.Address.Path, w.ID)
		if err != nil {
			return err
		}
		if err := mustHaveChanged(res); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM website_repos WHERE website_id = ?`, w.ID); err != nil {
			return err
		}
		return writeRepos(tx, w.ID, w.Repos)
	})
}

func writeRepos(tx *sql.Tx, id int64, repos []domain.Repo) error {
	for i, r := range repos {
		if _, err := tx.Exec(`INSERT INTO website_repos (website_id, position, owner, name) VALUES (?, ?, ?, ?)`,
			id, i, r.Owner, r.Name); err != nil {
			return err
		}
	}
	return nil
}

// DeleteWebsite removes a website with its history (FR-010): the release
// files of every repo no remaining website chose. Page loads stay, since they
// belong to whichever website owns their path.
func (s *Store) DeleteWebsite(id int64) error {
	return s.inTransaction(func(tx *sql.Tx) error {
		res, err := tx.Exec(`DELETE FROM websites WHERE id = ?`, id)
		if err != nil {
			return err
		}
		if err := mustHaveChanged(res); err != nil {
			return err
		}
		_, err = tx.Exec(`DELETE FROM release_files WHERE repo NOT IN
			(SELECT lower(owner || '/' || name) FROM website_repos)`)
		return err
	})
}

func mustHaveChanged(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return application.ErrNoSuchSite
	}
	return nil
}

// repoKey is how release files are filed: owner/name in lower case, since
// GitHub ignores case.
func repoKey(r domain.Repo) string { return strings.ToLower(r.String()) }
