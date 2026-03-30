package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/TeXmeijin/ccmon/internal/model"
)

type Store struct {
	db *sql.DB
}

func Open(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	d, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=3000")
	if err != nil {
		return nil, err
	}

	s := &Store{db: d}
	if err := s.migrate(); err != nil {
		d.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			source_namespace TEXT NOT NULL,
			session_id       TEXT NOT NULL,
			cwd              TEXT NOT NULL DEFAULT '',
			cwd_label        TEXT NOT NULL DEFAULT '',
			status           TEXT NOT NULL DEFAULT 'running',
			started_at       TEXT NOT NULL,
			last_event_at    TEXT NOT NULL,
			ended_at         TEXT,
			current_action   TEXT NOT NULL DEFAULT '',
			headline         TEXT NOT NULL DEFAULT '',
			headline_source  TEXT NOT NULL DEFAULT 'none',
			session_title    TEXT NOT NULL DEFAULT '',
			short_id         TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (source_namespace, session_id)
		);

		CREATE TABLE IF NOT EXISTS events (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			source_namespace  TEXT NOT NULL,
			session_id        TEXT NOT NULL,
			event_name        TEXT NOT NULL,
			event_at          TEXT NOT NULL,
			tool_name         TEXT,
			notification_type TEXT,
			preview           TEXT
		);

		CREATE INDEX IF NOT EXISTS idx_events_session
			ON events (source_namespace, session_id, event_at DESC);
	`)
	if err != nil {
		return err
	}

	// Migrations for existing DBs
	s.db.Exec(`ALTER TABLE sessions ADD COLUMN headline_source TEXT NOT NULL DEFAULT 'none'`)
	s.db.Exec(`ALTER TABLE sessions ADD COLUMN session_title TEXT NOT NULL DEFAULT ''`)
	s.db.Exec(`ALTER TABLE sessions ADD COLUMN ghostty_terminal_id TEXT NOT NULL DEFAULT ''`)

	return nil
}

func (s *Store) UpsertSession(sess *model.Session) error {
	_, err := s.db.Exec(`
		INSERT INTO sessions (source_namespace, session_id, cwd, cwd_label, status, started_at, last_event_at, ended_at, current_action, headline, headline_source, session_title, short_id, ghostty_terminal_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_namespace, session_id) DO UPDATE SET
			cwd = CASE WHEN excluded.cwd != '' THEN excluded.cwd ELSE sessions.cwd END,
			cwd_label = CASE WHEN excluded.cwd_label != '' THEN excluded.cwd_label ELSE sessions.cwd_label END,
			status = excluded.status,
			last_event_at = excluded.last_event_at,
			ended_at = COALESCE(excluded.ended_at, sessions.ended_at),
			current_action = CASE WHEN excluded.current_action != '' THEN excluded.current_action ELSE sessions.current_action END,
			headline = CASE
				WHEN excluded.headline != '' THEN excluded.headline
				WHEN excluded.status = 'running' AND sessions.headline_source = 'notification' THEN ''
				ELSE sessions.headline END,
			headline_source = CASE
				WHEN excluded.headline != '' THEN excluded.headline_source
				WHEN excluded.status = 'running' AND sessions.headline_source = 'notification' THEN 'none'
				ELSE sessions.headline_source END,
			session_title = CASE WHEN sessions.session_title = '' AND excluded.session_title != '' THEN excluded.session_title ELSE sessions.session_title END,
			ghostty_terminal_id = CASE WHEN excluded.ghostty_terminal_id != '' THEN excluded.ghostty_terminal_id ELSE sessions.ghostty_terminal_id END
	`,
		sess.SourceNamespace, sess.SessionID, sess.Cwd, sess.CwdLabel,
		string(sess.Status), sess.StartedAt.Format(time.RFC3339),
		sess.LastEventAt.Format(time.RFC3339), formatOptionalTime(sess.EndedAt),
		sess.CurrentAction, sess.Headline, string(sess.HeadlineSource), sess.SessionTitle, sess.ShortID,
		sess.GhosttyTerminalID,
	)
	return err
}

func (s *Store) InsertEvent(evt *model.Event) error {
	_, err := s.db.Exec(`
		INSERT INTO events (source_namespace, session_id, event_name, event_at, tool_name, notification_type, preview)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		evt.SourceNamespace, evt.SessionID, evt.EventName,
		evt.EventAt.Format(time.RFC3339),
		evt.ToolName, evt.NotificationType, evt.Preview,
	)
	return err
}

func (s *Store) ClearGhosttyTerminalID(sourceNamespace, sessionID string) error {
	_, err := s.db.Exec(`
		UPDATE sessions
		SET ghostty_terminal_id = ''
		WHERE source_namespace = ? AND session_id = ?
	`, sourceNamespace, sessionID)
	return err
}

func (s *Store) ListSessions() ([]model.Session, error) {
	rows, err := s.db.Query(`
		SELECT source_namespace, session_id, cwd, cwd_label, status, started_at, last_event_at, ended_at, current_action, headline, headline_source, session_title, short_id, ghostty_terminal_id
		FROM sessions
		ORDER BY last_event_at DESC, session_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []model.Session
	for rows.Next() {
		var sess model.Session
		var status, headlineSource string
		var startedAt, lastEventAt string
		var endedAt sql.NullString
		err := rows.Scan(
			&sess.SourceNamespace, &sess.SessionID, &sess.Cwd, &sess.CwdLabel,
			&status, &startedAt, &lastEventAt, &endedAt,
			&sess.CurrentAction, &sess.Headline, &headlineSource, &sess.SessionTitle, &sess.ShortID,
			&sess.GhosttyTerminalID,
		)
		if err != nil {
			return nil, err
		}
		sess.Status = model.Status(status)
		sess.HeadlineSource = model.HeadlineSource(headlineSource)
		sess.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		sess.LastEventAt, _ = time.Parse(time.RFC3339, lastEventAt)
		if endedAt.Valid {
			t, _ := time.Parse(time.RFC3339, endedAt.String)
			sess.EndedAt = &t
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

func (s *Store) RecentEvents(sourceNamespace, sessionID string, limit int) ([]model.Event, error) {
	rows, err := s.db.Query(`
		SELECT id, source_namespace, session_id, event_name, event_at, tool_name, notification_type, preview
		FROM events
		WHERE source_namespace = ? AND session_id = ?
		ORDER BY event_at DESC
		LIMIT ?
	`, sourceNamespace, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		var evt model.Event
		var eventAt string
		err := rows.Scan(&evt.ID, &evt.SourceNamespace, &evt.SessionID, &evt.EventName,
			&eventAt, &evt.ToolName, &evt.NotificationType, &evt.Preview)
		if err != nil {
			return nil, err
		}
		evt.EventAt, _ = time.Parse(time.RFC3339, eventAt)
		events = append(events, evt)
	}
	return events, rows.Err()
}

func formatOptionalTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}
